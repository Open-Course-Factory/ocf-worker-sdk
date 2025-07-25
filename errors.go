package ocfworker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents a structured error response from the OCF Worker API.
// It provides detailed error information including HTTP status codes,
// error messages, request context, and validation details.
//
// APIErrors are returned for all HTTP error responses and can be used
// for precise error handling and user-friendly error reporting.
//
// Example usage:
//
//	_, err := client.Jobs.Create(ctx, req)
//	if err != nil {
//		if apiErr, ok := err.(*ocfworker.APIError); ok {
//			switch apiErr.StatusCode {
//			case 400:
//				// Handle validation errors
//				for _, detail := range apiErr.Details {
//					fmt.Printf("Field %s: %s\n", detail.Field, detail.Message)
//				}
//			case 401:
//				// Handle authentication errors
//				fmt.Printf("Authentication failed: %s\n", apiErr.Message)
//			case 500:
//				// Handle server errors
//				fmt.Printf("Server error: %s\n", apiErr.Message)
//			}
//		}
//	}
type APIError struct {
	// StatusCode is the HTTP status code returned by the API
	StatusCode int `json:"status_code,omitempty"`

	// Message is the main error message describing what went wrong
	Message string `json:"message"`

	// Path is the API endpoint path where the error occurred
	Path string `json:"path,omitempty"`

	// RequestID is a unique identifier for the request, useful for debugging
	// and correlating errors with server logs
	RequestID string `json:"request_id,omitempty"`

	// Timestamp indicates when the error occurred on the server
	Timestamp string `json:"timestamp,omitempty"`

	// Details contains field-specific validation errors for request validation failures.
	// This is particularly useful for form validation and input sanitization errors.
	Details []ValidationError `json:"validation_errors,omitempty"`
}

// Error implements the error interface and returns a human-readable error message.
// The format is "API error {status}: {message}" for consistency and debugging ease.
func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// IsClientError returns true if the error is a client error (4xx status code).
// Client errors indicate problems with the request that should be fixed by the caller.
//
// Example:
//
//	if apiErr.IsClientError() {
//		log.Printf("Client error - fix your request: %s", apiErr.Message)
//	}
func (e *APIError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}

// IsServerError returns true if the error is a server error (5xx status code).
// Server errors indicate problems on the server side that may be temporary.
//
// Example:
//
//	if apiErr.IsServerError() {
//		log.Printf("Server error - may be worth retrying: %s", apiErr.Message)
//	}
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500
}

// HasValidationErrors returns true if the error includes field validation details.
// This is typically true for 400 Bad Request responses with form validation failures.
func (e *APIError) HasValidationErrors() bool {
	return len(e.Details) > 0
}

// ValidationError represents a field-specific validation error.
// These are included in APIError.Details for request validation failures.
//
// Example usage:
//
//	if apiErr.HasValidationErrors() {
//		for _, valErr := range apiErr.Details {
//			fmt.Printf("Field '%s' failed validation '%s': %s (value: '%s')\n",
//				valErr.Field, valErr.Code, valErr.Message, valErr.Value)
//		}
//	}
type ValidationError struct {
	// Field is the name of the request field that failed validation
	Field string `json:"field"`

	// Code is a machine-readable error code for the validation failure.
	// Common codes include: "required", "invalid_format", "too_long", "too_short"
	Code string `json:"code"`

	// Message is a human-readable description of the validation error
	Message string `json:"message"`

	// Value is the actual value that failed validation (may be empty for security)
	Value string `json:"value"`
}

// Error implements the error interface for ValidationError.
// This allows individual validation errors to be treated as Go errors.
func (v *ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", v.Field, v.Message)
}

// JobNotFoundError is a specialized error for when a requested job doesn't exist.
// This error is returned by job-related operations when the specified job ID
// cannot be found in the system.
//
// This error type allows for specific handling of missing jobs vs other API errors.
//
// Example usage:
//
//	job, err := client.Jobs.Get(ctx, jobID)
//	if err != nil {
//		if jobErr, ok := err.(*ocfworker.JobNotFoundError); ok {
//			fmt.Printf("Job %s was not found - it may have been deleted\n", jobErr.JobID)
//			// Handle missing job scenario...
//		} else {
//			// Handle other errors...
//		}
//	}
type JobNotFoundError struct {
	// JobID is the ID of the job that was not found
	JobID string
}

// Error implements the error interface and returns a descriptive error message
// including the job ID that was not found.
func (e *JobNotFoundError) Error() string {
	return fmt.Sprintf("job not found: %s", e.JobID)
}

// IsTemporary returns false since a missing job is typically a permanent condition
// unless the job is recreated with the same ID.
func (e *JobNotFoundError) IsTemporary() bool {
	return false
}

// parseAPIError extracts structured error information from an HTTP response.
// It attempts to parse the response body as JSON and create an appropriate error type.
//
// The function handles multiple error response formats:
// 1. Full APIError JSON with validation details
// 2. Simple error messages with "error" or "message" fields
// 3. Plain text or unparseable responses (falls back to HTTP status)
//
// This is an internal function used by service implementations to convert
// HTTP error responses into typed Go errors.
func parseAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	// Attempt to parse as a complete APIError structure first
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	// Try to parse as a simple error message
	var simpleErr struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &simpleErr); err == nil {
		message := simpleErr.Error
		if message == "" {
			message = simpleErr.Message
		}
		if message != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(body),
			}
		}
	}

	// Fall back to HTTP status if JSON parsing fails
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    resp.Status,
	}
}

// Common error checking helpers

// IsNotFoundError returns true if the error indicates a resource was not found.
// This works for both JobNotFoundError and APIError with 404 status.
//
// Example:
//
//	_, err := client.Jobs.Get(ctx, jobID)
//	if ocfworker.IsNotFoundError(err) {
//		log.Printf("Job does not exist: %s", jobID)
//		// Handle missing resource...
//	}
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific JobNotFoundError
	if _, ok := err.(*JobNotFoundError); ok {
		return true
	}

	// Check for APIError with 404 status
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}

	return false
}

// IsValidationError returns true if the error is a validation error with field details.
// This helps distinguish validation errors from other client errors.
//
// Example:
//
//	_, err := client.Jobs.Create(ctx, req)
//	if ocfworker.IsValidationError(err) {
//		log.Printf("Request validation failed - check your input data")
//		// Show field-specific error messages to user...
//	}
func IsValidationError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusBadRequest && apiErr.HasValidationErrors()
	}
	return false
}

// IsAuthenticationError returns true if the error indicates authentication failure.
// This typically means the API token is missing, invalid, or expired.
//
// Example:
//
//	_, err := client.Jobs.Create(ctx, req)
//	if ocfworker.IsAuthenticationError(err) {
//		log.Printf("Authentication failed - check your API token")
//		// Refresh token or prompt for new credentials...
//	}
func IsAuthenticationError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsTemporaryError returns true if the error might be resolved by retrying the request.
// This includes server errors (5xx) and certain client errors like rate limiting.
//
// Example:
//
//	_, err := client.Jobs.Create(ctx, req)
//	if ocfworker.IsTemporaryError(err) {
//		log.Printf("Temporary error - retrying may help")
//		// Implement retry logic...
//	}
func IsTemporaryError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		// Server errors are generally temporary
		if apiErr.IsServerError() {
			return true
		}
		// Rate limiting and service unavailable are temporary
		if apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode == http.StatusServiceUnavailable {
			return true
		}
	}
	return false
}
