// Package ocfworker provides a Go SDK for interacting with the OCF Worker API.
//
// The OCF Worker SDK allows you to manage presentation generation jobs,
// upload source files, manage themes, and download results through a simple Go interface.
//
// # Basic Usage
//
//	client := ocfworker.NewClient("http://localhost:8081",
//		ocfworker.WithAuth("your-token"),
//		ocfworker.WithTimeout(60*time.Second),
//	)
//
//	// Create a job
//	req := &models.GenerationRequest{
//		JobID:      uuid.New(),
//		CourseID:   uuid.New(),
//		SourcePath: "/sources",
//	}
//	job, err := client.Jobs.Create(ctx, req)
//
// # Authentication
//
// The SDK supports Bearer token authentication:
//
//	client := ocfworker.NewClient(baseURL, ocfworker.WithAuth("your-bearer-token"))
//
// # Error Handling
//
// The SDK provides typed errors for better error handling:
//
//	_, err := client.Jobs.Get(ctx, jobID)
//	if err != nil {
//		if jobErr, ok := err.(*ocfworker.JobNotFoundError); ok {
//			log.Printf("Job not found: %s", jobErr.JobID)
//		} else if apiErr, ok := err.(*ocfworker.APIError); ok {
//			log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
//		}
//	}
package ocfworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	// Import des types existants
)

// Logger defines the interface for logging within the OCF Worker SDK.
// Users can provide their own logger implementation to integrate with
// their existing logging infrastructure.
//
// Example with logrus:
//
//	type LogrusAdapter struct {
//		logger *logrus.Logger
//	}
//
//	func (l *LogrusAdapter) Info(msg string, fields ...interface{}) {
//		l.logger.WithFields(logrus.Fields{"data": fields}).Info(msg)
//	}
//
//	client := ocfworker.NewClient(baseURL, ocfworker.WithLogger(&LogrusAdapter{logger: logrus.New()}))
type Logger interface {
	// Debug logs debug-level messages with optional key-value pairs
	Debug(msg string, fields ...interface{})
	// Info logs info-level messages with optional key-value pairs
	Info(msg string, fields ...interface{})
	// Warn logs warning-level messages with optional key-value pairs
	Warn(msg string, fields ...interface{})
	// Error logs error-level messages with optional key-value pairs
	Error(msg string, fields ...interface{})
}

// Client is the main OCF Worker SDK client that provides access to all services.
// It manages HTTP communication, authentication, and logging for all API operations.
//
// The client is thread-safe and can be shared across goroutines.
// It should be reused rather than creating multiple instances.
//
// Example:
//
//	client := ocfworker.NewClient("http://localhost:8081",
//		ocfworker.WithAuth("bearer-token"),
//		ocfworker.WithTimeout(30*time.Second),
//	)
//	defer client.Close() // If implemented
type Client struct {
	// httpClient is the underlying HTTP client used for all requests
	httpClient *http.Client
	// baseURL is the base URL of the OCF Worker API
	baseURL string
	// logger handles all logging operations
	logger Logger

	// Services provide access to different API endpoints through well-defined interfaces.
	// This allows for easy testing and extensibility.

	// Jobs manages presentation generation jobs
	Jobs JobsServiceInterface
	// Storage handles file upload/download operations
	Storage StorageServiceInterface
	// Worker provides worker pool management capabilities
	Worker WorkerServiceInterface
	// Themes manages Slidev theme installation and detection
	Themes ThemesServiceInterface
	// Health provides service health monitoring
	Health HealthServiceInterface
	// Archive handles course archive downloads
	Archive ArchiveServiceInterface
}

// Option is a functional option for configuring the Client.
// Options are applied during client creation and allow for flexible configuration.
//
// Example:
//
//	client := ocfworker.NewClient(baseURL,
//		ocfworker.WithTimeout(60*time.Second),
//		ocfworker.WithAuth("token"),
//		ocfworker.WithLogger(customLogger),
//	)
type Option func(*Client)

// WithTimeout configures the HTTP client timeout for all requests.
// This sets both the connection timeout and the overall request timeout.
//
// Default timeout is 30 seconds.
//
// Example:
//
//	client := ocfworker.NewClient(baseURL, ocfworker.WithTimeout(2*time.Minute))
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithAuth configures Bearer token authentication for all requests.
// The token will be automatically added to the Authorization header
// for every API call.
//
// Example:
//
//	client := ocfworker.NewClient(baseURL, ocfworker.WithAuth("your-api-token"))
func WithAuth(token string) Option {
	return func(c *Client) {
		// Ajouter l'auth aux headers par défaut
		if c.httpClient.Transport == nil {
			c.httpClient.Transport = &authTransport{
				token: token,
				base:  http.DefaultTransport,
			}
		}
	}
}

// WithLogger sets a custom logger for the client.
// If not provided, a simple logger that writes to the standard log package is used.
//
// The logger will receive debug, info, warning, and error messages
// throughout the SDK's operation.
//
// Example:
//
//	logger := &CustomLogger{} // implements Logger interface
//	client := ocfworker.NewClient(baseURL, ocfworker.WithLogger(logger))
func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithHTTPClient allows you to provide a custom HTTP client.
// This is useful for advanced configuration like custom TLS settings,
// proxy configuration, or connection pooling.
//
// Example:
//
//	httpClient := &http.Client{
//		Transport: &http.Transport{
//			MaxIdleConns:        100,
//			MaxIdleConnsPerHost: 10,
//		},
//		Timeout: 30 * time.Second,
//	}
//	client := ocfworker.NewClient(baseURL, ocfworker.WithHTTPClient(httpClient))
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new OCF Worker SDK client with the given base URL and options.
//
// The baseURL should point to your OCF Worker instance (e.g., "http://localhost:8081").
// Multiple options can be provided to customize the client behavior.
//
// Example:
//
//	// Basic client
//	client := ocfworker.NewClient("http://localhost:8081")
//
//	// Client with custom logger
//	client := ocfworker.NewClient("http://localhost:8081",
//		ocfworker.WithLogger(&CustomLogger{}),
//	)
func NewClient(baseURL string, opts ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		logger:  &simpleLogger{},
	}

	for _, opt := range opts {
		opt(client)
	}

	client.Jobs = &JobsService{client: client}
	client.Storage = &StorageService{client: client}
	client.Worker = &WorkerService{client: client}
	client.Themes = &ThemesService{client: client}
	client.Health = &HealthService{client: client}
	client.Archive = &ArchiveService{client: client}

	return client
}

// authTransport is an HTTP transport wrapper that adds Bearer token authentication
// to all outgoing requests.
type authTransport struct {
	token string
	base  http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface and adds
// the Authorization header to each request.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

// simpleLogger is the default logger implementation that writes to Go's standard log package.
// It's used when no custom logger is provided via WithLogger option.
type simpleLogger struct{}

// Debug logs debug messages with the [DEBUG] prefix
func (l *simpleLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] %s %v", msg, fields)
}

// Info logs info messages with the [INFO] prefix
func (l *simpleLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

// Warn logs warning messages with the [WARN] prefix
func (l *simpleLogger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

// Error logs error messages with the [ERROR] prefix
func (l *simpleLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

// get performs a GET request to the specified API path.
// It automatically prepends the base URL and API version prefix.
//
// This is an internal method used by service implementations.
func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1"+path, nil)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("GET request", "url", req.URL.String())
	return c.httpClient.Do(req)
}

// post performs a POST request to the specified API path with JSON body.
// It automatically prepends the base URL and API version prefix,
// and sets the appropriate Content-Type header.
//
// This is an internal method used by service implementations.
func (c *Client) post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("failed to encode request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1"+path, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("POST request", "url", req.URL.String())
	return c.httpClient.Do(req)
}
