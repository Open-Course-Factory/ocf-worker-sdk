package ocfworker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestServer wraps httptest.Server with OCF Worker API helpers
type TestServer struct {
	*httptest.Server
	routes map[string]http.HandlerFunc
}

// NewTestServer creates a new test server for OCF Worker API
func NewTestServer() *TestServer {
	ts := &TestServer{
		routes: make(map[string]http.HandlerFunc),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ts.routeHandler)

	ts.Server = httptest.NewServer(mux)
	return ts
}

// routeHandler dispatches requests to registered routes
func (ts *TestServer) routeHandler(w http.ResponseWriter, r *http.Request) {
	key := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

	if handler, exists := ts.routes[key]; exists {
		handler(w, r)
		return
	}

	// Default 404 response
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "route not found",
		"path":  r.URL.Path,
	})
}

// On registers a route handler
func (ts *TestServer) On(method, path string, handler http.HandlerFunc) {
	key := fmt.Sprintf("%s %s", method, path)
	ts.routes[key] = handler
}

// TestClient creates a client configured for testing
func (ts *TestServer) TestClient(opts ...Option) *Client {
	defaultOpts := []Option{
		WithTimeout(5 * time.Second),
	}

	allOpts := append(defaultOpts, opts...)
	return NewClient(ts.URL, allOpts...)
}

// Close closes the test server
func (ts *TestServer) Close() {
	ts.Server.Close()
}

// Response builders for common API responses

// JobResponseBuilder helps build JobResponse objects for testing
type JobResponseBuilder struct {
	job *models.JobResponse
}

// NewJobResponse creates a new JobResponse builder
func NewJobResponse() *JobResponseBuilder {
	return &JobResponseBuilder{
		job: &models.JobResponse{
			ID:        uuid.New(),
			CourseID:  uuid.New(),
			Status:    models.StatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

// WithID sets the job ID
func (b *JobResponseBuilder) WithID(id uuid.UUID) *JobResponseBuilder {
	b.job.ID = id
	return b
}

// WithCourseID sets the course ID
func (b *JobResponseBuilder) WithCourseID(courseID uuid.UUID) *JobResponseBuilder {
	b.job.CourseID = courseID
	return b
}

// WithStatus sets the job status
func (b *JobResponseBuilder) WithStatus(status models.JobStatus) *JobResponseBuilder {
	b.job.Status = status
	return b
}

// WithError sets the job error
func (b *JobResponseBuilder) WithError(err string) *JobResponseBuilder {
	b.job.Error = err
	return b
}

// WithLogs sets the job logs
func (b *JobResponseBuilder) WithLogs(logs []string) *JobResponseBuilder {
	b.job.Logs = logs
	return b
}

// Build returns the built JobResponse
func (b *JobResponseBuilder) Build() *models.JobResponse {
	return b.job
}

// APIErrorBuilder helps build API error responses
type APIErrorBuilder struct {
	err *APIError
}

// NewAPIError creates a new API error builder
func NewAPIError() *APIErrorBuilder {
	return &APIErrorBuilder{
		err: &APIError{
			StatusCode: http.StatusInternalServerError,
			Message:    "Internal server error",
			Timestamp:  time.Now().Format(time.RFC3339),
		},
	}
}

// WithStatus sets the HTTP status code
func (b *APIErrorBuilder) WithStatus(code int) *APIErrorBuilder {
	b.err.StatusCode = code
	return b
}

// WithMessage sets the error message
func (b *APIErrorBuilder) WithMessage(msg string) *APIErrorBuilder {
	b.err.Message = msg
	return b
}

// WithPath sets the request path
func (b *APIErrorBuilder) WithPath(path string) *APIErrorBuilder {
	b.err.Path = path
	return b
}

// WithValidationErrors adds validation errors
func (b *APIErrorBuilder) WithValidationErrors(errors []ValidationError) *APIErrorBuilder {
	b.err.Details = errors
	return b
}

// Build returns the built APIError
func (b *APIErrorBuilder) Build() *APIError {
	return b.err
}

// HTTP response helpers

// RespondJSON writes a JSON response
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// RespondError writes an error response
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{
		"error":   message,
		"message": message,
	})
}

// RespondAPIError writes an APIError response
func RespondAPIError(w http.ResponseWriter, err *APIError) {
	RespondJSON(w, err.StatusCode, err)
}

// Test assertion helpers

// AssertJobResponse asserts a JobResponse matches expectations
func AssertJobResponse(t *testing.T, expected, actual *models.JobResponse) {
	t.Helper()

	require.Equal(t, expected.ID, actual.ID, "job ID mismatch")
	require.Equal(t, expected.CourseID, actual.CourseID, "course ID mismatch")
	require.Equal(t, expected.Status, actual.Status, "status mismatch")

	if expected.Error != "" {
		require.Equal(t, expected.Error, actual.Error, "error message mismatch")
	}
}

// AssertAPIError asserts an error is an APIError with expected properties
func AssertAPIError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	t.Helper()

	require.Error(t, err, "expected an error")

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "expected APIError, got %T", err)
	require.Equal(t, expectedStatus, apiErr.StatusCode, "status code mismatch")

	if expectedMessage != "" {
		require.Contains(t, apiErr.Message, expectedMessage, "error message mismatch")
	}
}

// Context helpers

// TestContext creates a context with timeout for testing
func TestContext(timeout ...time.Duration) (context.Context, context.CancelFunc) {
	t := 5 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}
	return context.WithTimeout(context.Background(), t)
}

// Mock data generators

// MockGenerationRequest creates a test GenerationRequest
func MockGenerationRequest() *models.GenerationRequest {
	return &models.GenerationRequest{
		JobID:      uuid.New(),
		CourseID:   uuid.New(),
		SourcePath: "/test/sources",
	}
}

// MockFileUpload creates test file uploads
func MockFileUpload(name, content string) FileUpload {
	return FileUpload{
		Name:        name,
		Content:     []byte(content),
		ContentType: "text/plain",
	}
}

// MockFileList creates a test file list response
func MockFileList(files ...string) *models.FileListResponse {
	return &models.FileListResponse{
		Files: files,
		Count: len(files),
	}
}

// MockHealthResponse creates a test health response
func MockHealthResponse(status string) *models.HealthResponse {
	return &models.HealthResponse{
		Status: status,
		// Add other fields as needed based on the actual model
	}
}

// HTTP request helpers

// ReadJSONBody reads and unmarshals JSON from request body
func ReadJSONBody(t *testing.T, r *http.Request, dest interface{}) {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	require.NoError(t, err, "failed to read request body")

	err = json.Unmarshal(body, dest)
	require.NoError(t, err, "failed to unmarshal JSON body")
}

// AssertAuthHeader checks if the Authorization header is present and correct
func AssertAuthHeader(t *testing.T, r *http.Request, expectedToken string) {
	t.Helper()

	auth := r.Header.Get("Authorization")
	require.NotEmpty(t, auth, "Authorization header missing")
	require.True(t, strings.HasPrefix(auth, "Bearer "), "Authorization header should start with 'Bearer '")

	if expectedToken != "" {
		require.Equal(t, "Bearer "+expectedToken, auth, "token mismatch")
	}
}

// AssertContentType checks the Content-Type header
func AssertContentType(t *testing.T, r *http.Request, expectedType string) {
	t.Helper()

	contentType := r.Header.Get("Content-Type")
	require.Contains(t, contentType, expectedType, "Content-Type mismatch")
}

// Utility functions

// MustMarshalJSON marshals to JSON or fails the test
func MustMarshalJSON(t *testing.T, v interface{}) string {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err, "failed to marshal JSON")
	return string(data)
}

// MustParseUUID parses a UUID string or fails the test
func MustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()

	id, err := uuid.Parse(s)
	require.NoError(t, err, "failed to parse UUID: %s", s)
	return id
}
