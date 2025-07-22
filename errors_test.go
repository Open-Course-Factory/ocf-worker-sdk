package ocfworker

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError *APIError
		expected string
	}{
		{
			name: "basic API error",
			apiError: &APIError{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid request",
			},
			expected: "API error 400: Invalid request",
		},
		{
			name: "internal server error",
			apiError: &APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "Something went wrong",
			},
			expected: "API error 500: Something went wrong",
		},
		{
			name: "not found error",
			apiError: &APIError{
				StatusCode: http.StatusNotFound,
				Message:    "Resource not found",
			},
			expected: "API error 404: Resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.apiError.Error())
		})
	}
}

func TestJobNotFoundError_Error(t *testing.T) {
	jobID := "123e4567-e89b-12d3-a456-426614174000"
	err := &JobNotFoundError{JobID: jobID}

	expected := "job not found: " + jobID
	assert.Equal(t, expected, err.Error())
}

func TestParseAPIError(t *testing.T) {
	t.Run("valid JSON error response", func(t *testing.T) {
		apiErr := &APIError{
			Message:   "Validation failed",
			Path:      "/api/v1/jobs",
			RequestID: "req-123",
			Timestamp: "2024-01-01T00:00:00Z",
			Details: []ValidationError{
				{
					Field:   "course_id",
					Code:    "required",
					Message: "course_id is required",
					Value:   "",
				},
			},
		}

		body, err := json.Marshal(apiErr)
		require.NoError(t, err)

		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(string(body))),
		}

		result := parseAPIError(resp)
		require.Error(t, result)

		parsedErr, ok := result.(*APIError)
		require.True(t, ok)

		assert.Equal(t, http.StatusBadRequest, parsedErr.StatusCode)
		assert.Equal(t, apiErr.Message, parsedErr.Message)
		assert.Equal(t, apiErr.Path, parsedErr.Path)
		assert.Equal(t, apiErr.RequestID, parsedErr.RequestID)
		assert.Equal(t, apiErr.Timestamp, parsedErr.Timestamp)
		assert.Len(t, parsedErr.Details, 1)
		assert.Equal(t, apiErr.Details[0], parsedErr.Details[0])
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 Internal Server Error",
			Body:       io.NopCloser(strings.NewReader("invalid json")),
		}

		result := parseAPIError(resp)
		require.Error(t, result)

		apiErr, ok := result.(*APIError)
		require.True(t, ok)

		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		assert.Equal(t, "500 Internal Server Error", apiErr.Message)
	})

	t.Run("empty response body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Status:     "503 Service Unavailable",
			Body:       io.NopCloser(strings.NewReader("")),
		}

		result := parseAPIError(resp)
		require.Error(t, result)

		apiErr, ok := result.(*APIError)
		require.True(t, ok)

		assert.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
		assert.Equal(t, "503 Service Unavailable", apiErr.Message)
	})
}

func TestParseAPIErrorWithTestServer(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("structured API error", func(t *testing.T) {
		apiErr := NewAPIError().
			WithStatus(http.StatusUnprocessableEntity).
			WithMessage("Validation failed").
			WithPath("/api/v1/generate").
			WithValidationErrors([]ValidationError{
				{
					Field:   "job_id",
					Code:    "invalid_format",
					Message: "job_id must be a valid UUID",
					Value:   "invalid-uuid",
				},
				{
					Field:   "source_path",
					Code:    "required",
					Message: "source_path is required",
					Value:   "",
				},
			}).
			Build()

		server.On("POST", "/api/v1/test-error", func(w http.ResponseWriter, r *http.Request) {
			RespondAPIError(w, apiErr)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		resp, err := client.post(ctx, "/test-error", map[string]string{"test": "data"})
		require.NoError(t, err)
		defer resp.Body.Close()

		result := parseAPIError(resp)
		require.Error(t, result)

		parsedErr, ok := result.(*APIError)
		require.True(t, ok)

		assert.Equal(t, apiErr.StatusCode, parsedErr.StatusCode)
		assert.Equal(t, apiErr.Message, parsedErr.Message)
		assert.Equal(t, apiErr.Path, parsedErr.Path)
		assert.Len(t, parsedErr.Details, 2)

		// Check validation errors
		assert.Equal(t, "job_id", parsedErr.Details[0].Field)
		assert.Equal(t, "invalid_format", parsedErr.Details[0].Code)
		assert.Equal(t, "source_path", parsedErr.Details[1].Field)
		assert.Equal(t, "required", parsedErr.Details[1].Code)
	})

	t.Run("simple error response", func(t *testing.T) {
		server.On("GET", "/api/v1/test-simple-error", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusForbidden, "Access denied")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		resp, err := client.get(ctx, "/test-simple-error")
		require.NoError(t, err)
		defer resp.Body.Close()

		result := parseAPIError(resp)
		require.Error(t, result)

		apiErr, ok := result.(*APIError)
		require.True(t, ok)

		assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
		assert.Contains(t, apiErr.Message, "Access denied")
	})
}

func TestValidationError(t *testing.T) {
	valErr := ValidationError{
		Field:   "email",
		Code:    "invalid_format",
		Message: "email must be a valid email address",
		Value:   "invalid-email",
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(valErr)
	require.NoError(t, err)

	var unmarshaled ValidationError
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, valErr, unmarshaled)
}

func TestAPIErrorWithMultipleValidationErrors(t *testing.T) {
	validationErrors := []ValidationError{
		{
			Field:   "job_id",
			Code:    "required",
			Message: "job_id is required",
			Value:   "",
		},
		{
			Field:   "course_id",
			Code:    "invalid_format",
			Message: "course_id must be a valid UUID",
			Value:   "not-a-uuid",
		},
		{
			Field:   "source_path",
			Code:    "invalid_path",
			Message: "source_path must be an absolute path",
			Value:   "relative/path",
		},
	}

	apiErr := &APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "Request validation failed",
		Path:       "/api/v1/generate",
		RequestID:  "req-456",
		Timestamp:  "2024-01-01T12:00:00Z",
		Details:    validationErrors,
	}

	// Test the error message
	expected := "API error 400: Request validation failed"
	assert.Equal(t, expected, apiErr.Error())

	// Test JSON serialization
	data, err := json.Marshal(apiErr)
	require.NoError(t, err)

	var unmarshaled APIError
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, apiErr.StatusCode, unmarshaled.StatusCode)
	assert.Equal(t, apiErr.Message, unmarshaled.Message)
	assert.Equal(t, apiErr.Path, unmarshaled.Path)
	assert.Equal(t, apiErr.RequestID, unmarshaled.RequestID)
	assert.Equal(t, apiErr.Timestamp, unmarshaled.Timestamp)
	assert.Len(t, unmarshaled.Details, 3)

	for i, detail := range unmarshaled.Details {
		assert.Equal(t, validationErrors[i], detail)
	}
}

// Benchmark for parseAPIError function
func BenchmarkParseAPIError(b *testing.B) {
	apiErr := &APIError{
		Message:   "Test error",
		Path:      "/api/v1/test",
		RequestID: "req-123",
		Timestamp: "2024-01-01T00:00:00Z",
	}

	body, _ := json.Marshal(apiErr)

	for i := 0; i < b.N; i++ {
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(string(body))),
		}

		_ = parseAPIError(resp)
	}
}
