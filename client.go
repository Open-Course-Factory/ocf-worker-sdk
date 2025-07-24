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

// Logger interface simple pour le logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// Client principal du SDK
type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     Logger

	// Services - Utilisation des interfaces pour le découplage
	Jobs    JobsServiceInterface
	Storage StorageServiceInterface
	Worker  WorkerServiceInterface
	Themes  ThemesServiceInterface
	Health  HealthServiceInterface
	Archive ArchiveServiceInterface
}

// Option pour configurer le client
type Option func(*Client)

// WithTimeout configure le timeout HTTP
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithAuth configure l'authentification
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

// WithLogger configure le logger
func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithHTTPClient permet d'utiliser un client HTTP personnalisé
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient crée un nouveau client OCF Worker
func NewClient(baseURL string, opts ...Option) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		logger:  &simpleLogger{},
	}

	// Appliquer les options
	for _, opt := range opts {
		opt(client)
	}

	// Initialiser les services avec leurs implémentations concrètes
	client.Jobs = &JobsService{client: client}
	client.Storage = &StorageService{client: client}
	client.Worker = &WorkerService{client: client}
	client.Themes = &ThemesService{client: client}
	client.Health = &HealthService{client: client}
	client.Archive = &ArchiveService{client: client}

	return client
}

// authTransport ajoute l'authentification aux requêtes
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

// Simple logger implementation
type simpleLogger struct{}

func (l *simpleLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] %s %v", msg, fields)
}

func (l *simpleLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (l *simpleLogger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

func (l *simpleLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

// helper methods pour le client HTTP
func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1"+path, nil)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("GET request", "url", req.URL.String())
	return c.httpClient.Do(req)
}

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
