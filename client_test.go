package ocfworker

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "valid URL without options",
			baseURL: "http://localhost:8081",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "valid URL with timeout option",
			baseURL: "http://localhost:8081",
			opts:    []Option{WithTimeout(30 * time.Second)},
			wantErr: false,
		},
		{
			name:    "valid URL with auth option",
			baseURL: "http://localhost:8081",
			opts:    []Option{WithAuth("test-token")},
			wantErr: false,
		},
		{
			name:    "valid URL with multiple options",
			baseURL: "http://localhost:8081",
			opts: []Option{
				WithTimeout(60 * time.Second),
				WithAuth("test-token"),
				WithLogger(&simpleLogger{}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, tt.opts...)

			// Basic assertions
			require.NotNil(t, client)
			assert.Equal(t, tt.baseURL, client.baseURL)
			assert.NotNil(t, client.httpClient)
			assert.NotNil(t, client.logger)

			// Service initialization
			assert.NotNil(t, client.Jobs)
			assert.NotNil(t, client.Storage)
			assert.NotNil(t, client.Worker)
			assert.NotNil(t, client.Themes)
			assert.NotNil(t, client.Health)
			assert.NotNil(t, client.Archive)

			// Verify services have client reference
			assert.Equal(t, client, client.Jobs.client)
			assert.Equal(t, client, client.Storage.client)
			assert.Equal(t, client, client.Worker.client)
			assert.Equal(t, client, client.Themes.client)
			assert.Equal(t, client, client.Health.client)
			assert.Equal(t, client, client.Archive.client)
		})
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 45 * time.Second
	client := NewClient("http://localhost:8081", WithTimeout(timeout))

	assert.Equal(t, timeout, client.httpClient.Timeout)
}

func TestWithAuth(t *testing.T) {
	token := "test-auth-token"
	client := NewClient("http://localhost:8081", WithAuth(token))

	// Verify auth transport is set
	require.NotNil(t, client.httpClient.Transport)

	authTransport, ok := client.httpClient.Transport.(*authTransport)
	require.True(t, ok, "expected authTransport")
	assert.Equal(t, token, authTransport.token)
}

func TestWithLogger(t *testing.T) {
	customLogger := &testLogger{}
	client := NewClient("http://localhost:8081", WithLogger(customLogger))

	assert.Equal(t, customLogger, client.logger)
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 120 * time.Second,
	}
	client := NewClient("http://localhost:8081", WithHTTPClient(customClient))

	assert.Equal(t, customClient, client.httpClient)
}

func TestAuthTransport(t *testing.T) {
	token := "test-token-123"
	transport := &authTransport{
		token: token,
		base:  http.DefaultTransport,
	}

	// Create a test server to capture the request
	server := NewTestServer()
	defer server.Close()

	var capturedAuth string
	server.On("GET", "/test", func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Create a client with the auth transport
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify the Authorization header was set correctly
	expectedAuth := "Bearer " + token
	assert.Equal(t, expectedAuth, capturedAuth)
}

func TestClientHTTPMethods(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := server.TestClient()
	ctx := context.Background()

	t.Run("GET request", func(t *testing.T) {
		server.On("GET", "/api/v1/test", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			RespondJSON(w, http.StatusOK, map[string]string{"method": "GET"})
		})

		resp, err := client.get(ctx, "/test")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("POST request", func(t *testing.T) {
		server.On("POST", "/api/v1/test", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			AssertContentType(t, r, "application/json")

			var body map[string]string
			ReadJSONBody(t, r, &body)
			assert.Equal(t, "test", body["key"])

			RespondJSON(w, http.StatusCreated, map[string]string{"method": "POST"})
		})

		testData := map[string]string{"key": "test"}
		resp, err := client.post(ctx, "/test", testData)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

func TestClientWithAuthenticatedRequests(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	token := "test-auth-token"
	client := server.TestClient(WithAuth(token))
	ctx := context.Background()

	server.On("GET", "/api/v1/authenticated", func(w http.ResponseWriter, r *http.Request) {
		AssertAuthHeader(t, r, token)
		RespondJSON(w, http.StatusOK, map[string]string{"authenticated": "true"})
	})

	resp, err := client.get(ctx, "/authenticated")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientContextCancellation(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := server.TestClient()

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.get(ctx, "/test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestSimpleLogger(t *testing.T) {
	logger := &simpleLogger{}

	// These shouldn't panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

// testLogger is a logger implementation for testing
type testLogger struct {
	logs []string
}

func (l *testLogger) Debug(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "DEBUG: "+msg)
}

func (l *testLogger) Info(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "INFO: "+msg)
}

func (l *testLogger) Warn(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "WARN: "+msg)
}

func (l *testLogger) Error(msg string, fields ...interface{}) {
	l.logs = append(l.logs, "ERROR: "+msg)
}

func (l *testLogger) Contains(substring string) bool {
	for _, log := range l.logs {
		if strings.Contains(log, substring) {
			return true
		}
	}
	return false
}

func (l *testLogger) Clear() {
	l.logs = nil
}
