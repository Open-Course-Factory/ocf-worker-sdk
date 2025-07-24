# OCF Worker SDK for Go

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-AGPLv3-green.svg)](LICENSE)
[![Coverage](https://img.shields.io/badge/coverage-80%25-brightgreen.svg)](#)
[![GoDoc](https://godoc.org/github.com/Open-Course-Factory/ocf-worker-sdk?status.svg)](https://godoc.org/github.com/Open-Course-Factory/ocf-worker-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/Open-Course-Factory/ocf-worker-sdk)](https://goreportcard.com/report/github.com/Open-Course-Factory/ocf-worker-sdk)

The official Go SDK for interacting with the OCF Worker API. This SDK provides a clean, idiomatic Go interface for managing presentation generation jobs, uploading files, managing themes, and downloading results.

## 🚀 Features

- **Simple & Clean API**: Intuitive interface following Go best practices
- **Full API Coverage**: Complete support for all OCF Worker endpoints
- **Type Safety**: Strongly typed requests and responses
- **Error Handling**: Structured errors with detailed information
- **Context Support**: Full context.Context support for cancellation and timeouts
- **Concurrent Safe**: Thread-safe client that can be shared across goroutines
- **Flexible Configuration**: Extensible options pattern for client configuration
- **Comprehensive Testing**: Extensive test suite with mocks and helpers

## 📦 Installation

```bash
go get github.com/Open-Course-Factory/ocf-worker-sdk
```

**Requirements:**
- Go 1.23 or later
- Access to an OCF Worker instance

## 🏁 Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    ocfworker "github.com/Open-Course-Factory/ocf-worker-sdk"
    "github.com/Open-Course-Factory/ocf-worker/pkg/models"
    "github.com/google/uuid"
)

func main() {
    // Create client
    client := ocfworker.NewClient(
        "http://localhost:8081",
        ocfworker.WithTimeout(60*time.Second),
    )

    ctx := context.Background()

    // Check service health
    health, err := client.Health.Check(ctx)
    if err != nil {
        log.Fatalf("Health check failed: %v", err)
    }
    fmt.Printf("Service status: %s\n", health.Status)

    // Create and process a job
    jobID := uuid.New()
    courseID := uuid.New()

    req := &models.GenerationRequest{
        JobID:      jobID,
        CourseID:   courseID,
        SourcePath: "/sources",
    }

    // Create job and wait for completion
    job, err := client.Jobs.CreateAndWait(ctx, req, &ocfworker.WaitOptions{
        Interval: 5 * time.Second,
        Timeout:  10 * time.Minute,
    })
    if err != nil {
        log.Fatalf("Job failed: %v", err)
    }

    fmt.Printf("Job completed: %s\n", job.ID)
}
```

## 🔧 Configuration

### Client Options

The SDK provides several configuration options:

```go
client := ocfworker.NewClient("http://localhost:8081",
    
    // Custom timeout (default: 30s)
    ocfworker.WithTimeout(2*time.Minute),
    
    // Custom HTTP client
    ocfworker.WithHTTPClient(&http.Client{
        Transport: customTransport,
        Timeout:   30 * time.Second,
    }),
    
    // Custom logger
    ocfworker.WithLogger(&CustomLogger{}),
)
```

### Environment Variables

You can also configure the client using environment variables:

```bash
export OCF_WORKER_URL="http://localhost:8081"
export OCF_WORKER_TOKEN="your-api-token"
export OCF_WORKER_TIMEOUT="60s"
```

```go
// Load configuration from environment
client := ocfworker.NewClientFromEnv()
```

## 📖 Usage Guide

### Job Management

#### Creating and Monitoring Jobs

```go
// Create a job
req := &models.GenerationRequest{
    JobID:      uuid.New(),
    CourseID:   uuid.New(),
    SourcePath: "/sources",
}

job, err := client.Jobs.Create(ctx, req)
if err != nil {
    log.Fatalf("Failed to create job: %v", err)
}

// Monitor job progress
for {
    status, err := client.Jobs.Get(ctx, job.ID.String())
    if err != nil {
        log.Fatalf("Failed to get job status: %v", err)
    }

    fmt.Printf("Job status: %s\n", status.Status)

    if status.Status == models.StatusCompleted {
        fmt.Println("Job completed successfully!")
        break
    } else if status.Status == models.StatusFailed {
        log.Fatalf("Job failed: %s", status.Error)
    }

    time.Sleep(5 * time.Second)
}
```

#### Automatic Polling

```go
// Create job and wait for completion automatically
job, err := client.Jobs.CreateAndWait(ctx, req, &ocfworker.WaitOptions{
    Interval: 5 * time.Second,  // Check every 5 seconds
    Timeout:  10 * time.Minute, // Give up after 10 minutes
})
if err != nil {
    log.Fatalf("Job failed: %v", err)
}
```

#### Listing Jobs

```go
// List all jobs
jobs, err := client.Jobs.List(ctx, nil)

// List with filters
jobs, err := client.Jobs.List(ctx, &ocfworker.ListJobsOptions{
    Status:   "completed",
    CourseID: courseID.String(),
    Limit:    50,
    Offset:   0,
})
```

### File Management

#### Uploading Source Files

```go
// Upload from memory
files := []ocfworker.FileUpload{
    {
        Name:        "slides.md",
        Content:     []byte("---\ntheme: default\n---\n\n# My Presentation"),
        ContentType: "text/markdown",
    },
    {
        Name:        "style.css",
        Content:     []byte("body { font-family: Arial; }"),
        ContentType: "text/css",
    },
}

uploadResp, err := client.Storage.UploadSources(ctx, jobID.String(), files)
if err != nil {
    log.Fatalf("Failed to upload files: %v", err)
}
fmt.Printf("Uploaded %d files\n", uploadResp.Count)
```

```go
// Upload from filesystem
filePaths := []string{
    "./slides.md",
    "./images/logo.png",
    "./styles/theme.css",
}

uploadResp, err := client.Storage.UploadSourceFiles(ctx, jobID.String(), filePaths)
if err != nil {
    log.Fatalf("Failed to upload files: %v", err)
}
```

#### Downloading Results

```go
// List available results
results, err := client.Storage.ListResults(ctx, courseID.String())
if err != nil {
    log.Fatalf("Failed to list results: %v", err)
}

// Download specific file
for _, filename := range results.Files {
    reader, err := client.Storage.DownloadResult(ctx, courseID.String(), filename)
    if err != nil {
        log.Printf("Failed to download %s: %v", filename, err)
        continue
    }
    defer reader.Close()

    // Save to file
    outFile, err := os.Create(filename)
    if err != nil {
        log.Printf("Failed to create file %s: %v", filename, err)
        continue
    }
    defer outFile.Close()

    _, err = io.Copy(outFile, reader)
    if err != nil {
        log.Printf("Failed to save %s: %v", filename, err)
    }
}
```

#### Archive Downloads

```go
// Download complete course archive
archiveReader, err := client.Archive.DownloadArchive(ctx, courseID.String(), &ocfworker.DownloadArchiveOptions{
    Format:   "zip",
    Compress: &[]bool{true}[0],
})
if err != nil {
    log.Fatalf("Failed to download archive: %v", err)
}
defer archiveReader.Close()

// Save archive
archiveFile, err := os.Create("course-results.zip")
if err != nil {
    log.Fatalf("Failed to create archive file: %v", err)
}
defer archiveFile.Close()

_, err = io.Copy(archiveFile, archiveReader)
if err != nil {
    log.Fatalf("Failed to save archive: %v", err)
}
```

### Theme Management

```go
// List available themes
themes, err := client.Themes.ListAvailable(ctx)
if err != nil {
    log.Fatalf("Failed to list themes: %v", err)
}

for _, theme := range themes.Themes {
    fmt.Printf("Theme: %s v%s - %s\n", theme.Name, theme.Version, theme.Description)
}

// Auto-install themes for a job
themeResult, err := client.Themes.AutoInstallForJob(ctx, jobID.String())
if err != nil {
    log.Printf("Theme installation warning: %v", err)
} else {
    fmt.Printf("Installed %d themes successfully\n", themeResult.Successful)
}
```

### Worker Management

```go
// Check worker health
health, err := client.Worker.Health(ctx)
if err != nil {
    log.Fatalf("Failed to check worker health: %v", err)
}

fmt.Printf("Worker Status: %s\n", health.Status)
fmt.Printf("Active Workers: %d/%d\n", 
    health.WorkerPool.ActiveWorkers, 
    health.WorkerPool.WorkerCount)

// List active workspaces
workspaces, err := client.Worker.ListWorkspaces(ctx, &ocfworker.ListWorkspacesOptions{
    Status: "active",
    Limit:  10,
})
if err != nil {
    log.Fatalf("Failed to list workspaces: %v", err)
}

// Clean up old workspaces
cleanup, err := client.Worker.CleanupOldWorkspaces(ctx, 24) // 24 hours
if err != nil {
    log.Printf("Cleanup failed: %v", err)
} else {
    fmt.Printf("Cleaned %d workspaces, freed %d bytes\n", 
        cleanup.CleanedCount, cleanup.TotalSizeFreed)
}
```

## 🚨 Error Handling

The SDK provides structured error handling with detailed error information:

### API Errors

```go
_, err := client.Jobs.Create(ctx, req)
if err != nil {
    if apiErr, ok := err.(*ocfworker.APIError); ok {
        switch apiErr.StatusCode {
        case 400:
            // Validation errors
            if apiErr.HasValidationErrors() {
                for _, detail := range apiErr.Details {
                    fmt.Printf("Field '%s': %s\n", detail.Field, detail.Message)
                }
            }
        case 401:
            fmt.Printf("Authentication failed: %s\n", apiErr.Message)
        case 404:
            fmt.Printf("Resource not found: %s\n", apiErr.Message)
        case 500:
            fmt.Printf("Server error: %s\n", apiErr.Message)
        }
        
        // Check error type
        if apiErr.IsClientError() {
            log.Printf("Client error - fix your request")
        } else if apiErr.IsServerError() {
            log.Printf("Server error - may be worth retrying")
        }
    }
}
```

### Specific Error Types

```go
// Job not found
_, err := client.Jobs.Get(ctx, jobID)
if err != nil {
    if jobErr, ok := err.(*ocfworker.JobNotFoundError); ok {
        fmt.Printf("Job not found: %s\n", jobErr.JobID)
    }
}

// Helper functions
if ocfworker.IsNotFoundError(err) {
    log.Printf("Resource not found")
}

if ocfworker.IsValidationError(err) {
    log.Printf("Validation failed")
}

if ocfworker.IsAuthenticationError(err) {
    log.Printf("Authentication required")
}

if ocfworker.IsTemporaryError(err) {
    log.Printf("Temporary error - retry may help")
}
```

## 🧪 Testing

### Testing with Mocks

The SDK provides interfaces that make testing easy:

```go
func TestMyService(t *testing.T) {
    // Create a mock jobs service
    mockJobs := &MockJobsService{}
    
    // Create client and inject mock
    client := ocfworker.NewClient("http://localhost:8081")
    client.Jobs = mockJobs
    
    // Configure mock expectations
    expectedJob := &models.JobResponse{
        ID:     uuid.New(),
        Status: models.StatusCompleted,
    }
    mockJobs.On("Create", mock.Anything, mock.Anything).Return(expectedJob, nil)
    
    // Test your code
    result := myService.ProcessJob(client)
    
    // Verify results
    assert.Equal(t, expectedJob.ID, result.JobID)
    mockJobs.AssertExpectations(t)
}
```

### Integration Tests

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    client := ocfworker.NewClient(
        os.Getenv("OCF_WORKER_URL"),

    )
    
    ctx := context.Background()
    
    // Test actual API calls
    health, err := client.Health.Check(ctx)
    require.NoError(t, err)
    assert.Equal(t, "healthy", health.Status)
}
```

## 🏗️ Advanced Usage

### Custom HTTP Client

```go
// Custom transport with retry logic
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}

httpClient := &http.Client{
    Transport: &RetryTransport{Base: transport},
    Timeout:   30 * time.Second,
}

client := ocfworker.NewClient(baseURL, 
    ocfworker.WithHTTPClient(httpClient))
```

### Custom Logger

```go
type CustomLogger struct {
    logger *zap.Logger
}

func (l *CustomLogger) Info(msg string, fields ...interface{}) {
    l.logger.Info(msg, zap.Any("fields", fields))
}

// Implement other methods...

client := ocfworker.NewClient(baseURL,
    ocfworker.WithLogger(&CustomLogger{logger: zapLogger}))
```

### Service Extensions

```go
// Cached jobs service
type CachedJobsService struct {
    underlying ocfworker.JobsServiceInterface
    cache      map[string]*models.JobResponse
    mutex      sync.RWMutex
}

func (c *CachedJobsService) Get(ctx context.Context, jobID string) (*models.JobResponse, error) {
    c.mutex.RLock()
    if cached, exists := c.cache[jobID]; exists {
        c.mutex.RUnlock()
        return cached, nil
    }
    c.mutex.RUnlock()
    
    job, err := c.underlying.Get(ctx, jobID)
    if err == nil {
        c.mutex.Lock()
        c.cache[jobID] = job
        c.mutex.Unlock()
    }
    return job, err
}

// Use cached service
client := ocfworker.NewClient(baseURL)
client.Jobs = &CachedJobsService{
    underlying: client.Jobs,
    cache:      make(map[string]*models.JobResponse),
}
```

## 🔍 Debugging

### Enable Debug Logging

```go
type DebugLogger struct{}

func (l *DebugLogger) Debug(msg string, fields ...interface{}) {
    log.Printf("[DEBUG] %s %+v", msg, fields)
}

func (l *DebugLogger) Info(msg string, fields ...interface{}) {
    log.Printf("[INFO] %s %+v", msg, fields)
}

// ... implement other methods

client := ocfworker.NewClient(baseURL,
    ocfworker.WithLogger(&DebugLogger{}))
```

### Request Tracing

```go
// Add request ID to context for tracing
ctx := context.WithValue(context.Background(), "request_id", uuid.New().String())

job, err := client.Jobs.Create(ctx, req)
```

## 📚 Examples

See the [examples](examples/) directory for a complete working example:

- [Basic Usage](examples/axample.go) - Job creation, monitoring, download result

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/Open-Course-Factory/ocf-worker-sdk.git
cd ocf-worker-sdk

# Install dependencies
go mod download

# Run tests
go test ./...

# Run linting
golangci-lint run

# Generate documentation
go doc -all ./...
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: [GoDoc](https://godoc.org/github.com/Open-Course-Factory/ocf-worker-sdk)
- **Issues**: [GitHub Issues](https://github.com/Open-Course-Factory/ocf-worker-sdk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Open-Course-Factory/ocf-worker-sdk/discussions)

---

Made with ❤️ by the OCF Team
