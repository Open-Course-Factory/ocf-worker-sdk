# OCF Worker SDK

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-AGPLv3-green.svg)](LICENSE)
[![Coverage](https://img.shields.io/badge/coverage-80%25-brightgreen.svg)](#)

A comprehensive Go SDK for interacting with the **Open Course Factory (OCF) Worker** service. This SDK provides a clean, idiomatic Go interface for managing course generation jobs, file storage, themes, and worker health monitoring.

## 🚀 Features

- **🔄 Job Management**: Create, monitor, and manage course generation jobs
- **📁 File Storage**: Upload source files and download generated results  
- **🎨 Theme Management**: Install and manage Slidev themes automatically
- **👀 Health Monitoring**: Check service and worker pool health status
- **📦 Archive Downloads**: Download complete course archives in ZIP/TAR formats
- **⚡ Concurrent Safe**: Built with goroutine safety and race condition protection
- **🔧 Flexible Configuration**: Customizable timeouts, authentication, and logging
- **🧪 Comprehensive Testing**: 80%+ test coverage with integration tests

## 📦 Installation

```bash
go get github.com/Open-Course-Factory/ocf-worker-sdk
```

## 🎯 Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    ocfworker "github.com/your-org/ocf-worker-sdk"
    "github.com/Open-Course-Factory/ocf-worker/pkg/models"
    "github.com/google/uuid"
)

func main() {
    // Create client with authentication
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

    // Create a course generation job
    jobID := uuid.New()
    courseID := uuid.New()

    req := &models.GenerationRequest{
        JobID:      jobID,
        CourseID:   courseID,
        SourcePath: "/sources",
    }

    job, err := client.Jobs.Create(ctx, req)
    if err != nil {
        log.Fatalf("Failed to create job: %v", err)
    }
    
    fmt.Printf("Job created: %s\n", job.ID)
}
```

## 📘 Detailed Usage

### Client Configuration

```go
// Basic client
client := ocfworker.NewClient("http://localhost:8081")

// Client with all options
client := ocfworker.NewClient(
    "http://localhost:8081",
    ocfworker.WithTimeout(60*time.Second),
    ocfworker.WithLogger(customLogger),
    ocfworker.WithHTTPClient(customHTTPClient),
)
```

### Job Management

#### Create and Monitor Jobs

```go
// Create a job
req := &models.GenerationRequest{
    JobID:      uuid.New(),
    CourseID:   uuid.New(),
    SourcePath: "/sources",
}

job, err := client.Jobs.Create(ctx, req)
if err != nil {
    return err
}

// Manual polling
for {
    status, err := client.Jobs.Get(ctx, job.ID.String())
    if err != nil {
        return err
    }

    if status.Status == models.StatusCompleted {
        fmt.Println("Job completed!")
        break
    } else if status.Status == models.StatusFailed {
        return fmt.Errorf("job failed: %s", status.Error)
    }

    time.Sleep(5 * time.Second)
}
```

#### Automatic Job Waiting

```go
// Create job and wait for completion automatically
job, err := client.Jobs.CreateAndWait(ctx, req, &ocfworker.WaitOptions{
    Interval: 5 * time.Second,
    Timeout:  10 * time.Minute,
})
if err != nil {
    return err
}
fmt.Printf("Job completed: %s\n", job.Status)
```

#### List Jobs with Filters

```go
jobs, err := client.Jobs.List(ctx, &ocfworker.ListJobsOptions{
    Status:   "completed",
    CourseID: courseID.String(),
    Limit:    50,
    Offset:   0,
})
if err != nil {
    return err
}

fmt.Printf("Found %d jobs\n", jobs.TotalCount)
```

### File Storage

#### Upload Source Files

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

result, err := client.Storage.UploadSources(ctx, jobID.String(), files)
if err != nil {
    return err
}
fmt.Printf("Uploaded %d files\n", result.Count)

// Upload from filesystem  
filePaths := []string{"./slides.md", "./assets/style.css"}
result, err = client.Storage.UploadSourceFiles(ctx, jobID.String(), filePaths)
```

#### Download Results

```go
// List available results
results, err := client.Storage.ListResults(ctx, courseID.String())
if err != nil {
    return err
}

// Download specific file
reader, err := client.Storage.DownloadResult(ctx, courseID.String(), "presentation.pdf")
if err != nil {
    return err
}
defer reader.Close()

// Save to file
outFile, err := os.Create("presentation.pdf")
if err != nil {
    return err
}
defer outFile.Close()

_, err = io.Copy(outFile, reader)
```

#### Get Job Logs

```go
logs, err := client.Storage.GetLogs(ctx, jobID.String())
if err != nil {
    return err
}
fmt.Println("Job logs:")
fmt.Println(logs)
```

### Theme Management

#### List Available Themes

```go
themes, err := client.Themes.ListAvailable(ctx)
if err != nil {
    return err
}

for _, theme := range themes.Themes {
    fmt.Printf("Theme: %s v%s - %s\n", theme.Name, theme.Version, theme.Description)
}
```

#### Install Themes

```go
// Install specific theme
result, err := client.Themes.Install(ctx, "academic")
if err != nil {
    return err
}

if result.Success {
    fmt.Printf("Theme '%s' installed successfully\n", result.Theme)
} else {
    fmt.Printf("Failed to install theme: %s\n", result.Message)
}

// Auto-detect and install required themes for a job
autoResult, err := client.Themes.AutoInstallForJob(ctx, jobID.String())
if err != nil {
    return err
}
fmt.Printf("Successfully installed %d themes\n", autoResult.Successful)
```

### Health Monitoring

#### Service Health

```go
health, err := client.Health.Check(ctx)
if err != nil {
    return err
}

fmt.Printf("Service status: %s\n", health.Status)
```

#### Worker Pool Health

```go
workerHealth, err := client.Worker.Health(ctx)
if err != nil {
    return err
}

fmt.Printf("Worker pool status: %s\n", workerHealth.Status)
fmt.Printf("Active workers: %d/%d\n", 
    workerHealth.WorkerPool.ActiveWorkers,
    workerHealth.WorkerPool.WorkerCount)
```

#### Workspace Management

```go
// List active workspaces
workspaces, err := client.Worker.ListWorkspaces(ctx, &ocfworker.ListWorkspacesOptions{
    Status: "active",
    Limit:  20,
})

// Get specific workspace info
workspace, err := client.Worker.GetWorkspace(ctx, jobID.String())
if err != nil {
    return err
}
fmt.Printf("Workspace disk usage: %d bytes\n", workspace.Usage.DiskUsage.TotalBytes)

// Cleanup old workspaces
cleanup, err := client.Worker.CleanupOldWorkspaces(ctx, 24) // 24 hours
if err != nil {
    return err
}
fmt.Printf("Cleaned %d workspaces, freed %d bytes\n", 
    cleanup.CleanedCount, cleanup.TotalSizeFreed)
```

### Archive Downloads

```go
// Download as ZIP
archiveReader, err := client.Archive.DownloadArchive(ctx, courseID.String(), &ocfworker.DownloadArchiveOptions{
    Format:   "zip",
    Compress: &[]bool{true}[0],
})
if err != nil {
    return err
}
defer archiveReader.Close()

// Save archive
outFile, err := os.Create("course.zip")
if err != nil {
    return err
}
defer outFile.Close()

_, err = io.Copy(outFile, archiveReader)
```

## 🔧 Configuration Options

### Client Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithTimeout(duration)` | HTTP request timeout | 30 seconds |
| `WithLogger(logger)` | Custom logger implementation | Simple console logger |
| `WithHTTPClient(client)` | Custom HTTP client | Default Go HTTP client |

### Wait Options

| Option | Description | Default |
|--------|-------------|---------|
| `Interval` | Polling interval for job status | 5 seconds |
| `Timeout` | Maximum wait time | 10 minutes |

## 🚨 Error Handling

The SDK provides structured error handling with specific error types:

```go
job, err := client.Jobs.Get(ctx, "invalid-job-id")
if err != nil {
    // Check for specific error types
    if jobErr, ok := err.(*ocfworker.JobNotFoundError); ok {
        fmt.Printf("Job not found: %s\n", jobErr.JobID)
        return
    }
    
    // Check for API errors
    if apiErr, ok := err.(*ocfworker.APIError); ok {
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        
        // Check validation errors
        for _, valErr := range apiErr.Details {
            fmt.Printf("Field %s: %s\n", valErr.Field, valErr.Message)
        }
        return
    }
    
    // Other errors
    fmt.Printf("Unexpected error: %v\n", err)
}
```

## 🧪 Testing

The SDK includes comprehensive test coverage:

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test suite
go test -v ./jobs_test.go
```

### Test Utilities

The SDK provides test utilities for easy mocking:

```go
func TestMyFunction(t *testing.T) {
    server := ocfworker.NewTestServer()
    defer server.Close()
    
    server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
        ocfworker.RespondJSON(w, http.StatusOK, ocfworker.MockHealthResponse("healthy"))
    })
    
    client := server.TestClient()
    // ... test your code
}
```

## 📋 Requirements

- **Go**: 1.23 or higher
- **OCF Worker Service**: Compatible with OCF Worker API v1

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/your-org/ocf-worker-sdk.git
cd ocf-worker-sdk

# Install dependencies
go mod download

# Run tests
go test -v ./...

# Run linting
golangci-lint run
```

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📚 Examples

Complete examples are available in the [`examples/`](examples/) directory:

- [`examples/example.go`](examples/example.go) - Complete workflow example
- More examples coming soon!

## 🆘 Support

- **Documentation**: [OCF Worker API Docs](https://docs.ocf-worker.com)
- **Issues**: [GitHub Issues](https://github.com/your-org/ocf-worker-sdk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/ocf-worker-sdk/discussions)

---

Made with ❤️ by the OCF Team