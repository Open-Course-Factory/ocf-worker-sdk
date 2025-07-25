package ocfworker

import (
	"context"
	"io"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
)

// JobsServiceInterface defines the contract for job management operations.
// Jobs represent presentation generation tasks that process source files
// and produce output files like PDFs and HTML presentations.
//
// Example usage:
//
//	// Create and wait for job completion
//	req := &models.GenerationRequest{
//		JobID:      uuid.New(),
//		CourseID:   uuid.New(),
//		SourcePath: "/sources",
//	}
//	job, err := client.Jobs.CreateAndWait(ctx, req, &ocfworker.WaitOptions{
//		Interval: 5 * time.Second,
//		Timeout:  10 * time.Minute,
//	})
type JobsServiceInterface interface {
	// Create submits a new presentation generation job to the worker.
	// The job will be queued and processed asynchronously.
	//
	// Returns the created job with initial status (usually "pending").
	// Use Get() or WaitForCompletion() to monitor job progress.
	Create(ctx context.Context, req *models.GenerationRequest) (*models.JobResponse, error)

	// Get retrieves the current status and details of a specific job.
	// This includes status, logs, error messages, and progress information.
	//
	// Returns JobNotFoundError if the job doesn't exist.
	Get(ctx context.Context, jobID string) (*models.JobResponse, error)

	// List retrieves a paginated list of jobs with optional filtering.
	// Supports filtering by status, course ID, and pagination parameters.
	//
	// Example:
	//	jobs, err := client.Jobs.List(ctx, &ocfworker.ListJobsOptions{
	//		Status: "completed",
	//		Limit:  50,
	//	})
	List(ctx context.Context, opts *ListJobsOptions) (*models.JobListResponse, error)

	// CreateAndWait creates a job and automatically polls until completion.
	// This is a convenience method that combines Create() and WaitForCompletion().
	//
	// The method will poll the job status at the specified interval until
	// the job completes (successfully or with error) or the timeout is reached.
	CreateAndWait(ctx context.Context, req *models.GenerationRequest, opts *WaitOptions) (*models.JobResponse, error)

	// WaitForCompletion polls a job until it reaches a terminal state.
	// Terminal states are: completed, failed, or timeout.
	//
	// The method respects both the context timeout and the WaitOptions timeout.
	// Returns an error if the job fails or if polling times out.
	WaitForCompletion(ctx context.Context, jobID string, opts *WaitOptions) (*models.JobResponse, error)
}

// StorageServiceInterface defines the contract for file storage operations.
// This includes uploading source files, downloading results, and managing
// job-related storage resources.
//
// Example usage:
//
//	// Upload source files
//	files := []ocfworker.FileUpload{{
//		Name:    "slides.md",
//		Content: []byte("# My Presentation\n## Slide 1"),
//		ContentType: "text/markdown",
//	}}
//	resp, err := client.Storage.UploadSources(ctx, jobID, files)
//
//	// Download results
//	results, err := client.Storage.ListResults(ctx, courseID)
//	for _, filename := range results.Files {
//		reader, err := client.Storage.DownloadResult(ctx, courseID, filename)
//		// Process file...
//	}
type StorageServiceInterface interface {
	// UploadSources uploads source files for a job using in-memory content.
	// This method is suitable for small to medium-sized files that can be
	// loaded entirely into memory.
	//
	// Files are uploaded as multipart/form-data to the API.
	UploadSources(ctx context.Context, jobID string, files []FileUpload) (*models.FileUploadResponse, error)

	// UploadSourcesStream uploads source files using streaming I/O.
	// This method is more memory-efficient for large files as it doesn't
	// require loading the entire file content into memory.
	UploadSourcesStream(ctx context.Context, jobID string, uploads []StreamUpload) (*models.FileUploadResponse, error)

	// UploadSourceFiles is a convenience method that uploads files directly
	// from the filesystem. It handles opening files and setting up streaming uploads.
	//
	// Example:
	//	filePaths := []string{"./slides.md", "./images/logo.png"}
	//	resp, err := client.Storage.UploadSourceFiles(ctx, jobID, filePaths)
	UploadSourceFiles(ctx context.Context, jobID string, filePaths []string) (*models.FileUploadResponse, error)

	// ListSources returns a list of all source files uploaded for a job.
	// This is useful for verifying successful uploads or debugging.
	ListSources(ctx context.Context, jobID string) (*models.FileListResponse, error)

	// DownloadSource downloads a specific source file by name.
	// Returns a ReadCloser that must be closed by the caller.
	//
	// The caller is responsible for closing the returned ReadCloser.
	DownloadSource(ctx context.Context, jobID, filename string) (io.ReadCloser, error)

	// ListResults returns all output files generated by a completed job.
	// Results are organized by course ID and may include PDFs, HTML files,
	// and other generated assets.
	ListResults(ctx context.Context, courseID string) (*models.FileListResponse, error)

	// DownloadResult downloads a specific result file by name.
	// Returns a ReadCloser that must be closed by the caller.
	//
	// Common result files include "presentation.pdf", "slides.html", etc.
	DownloadResult(ctx context.Context, courseID, filename string) (io.ReadCloser, error)

	// GetLogs retrieves the complete log output from a job's execution.
	// This includes compilation logs, error messages, and debug information.
	//
	// Logs are particularly useful for debugging failed jobs.
	GetLogs(ctx context.Context, jobID string) (string, error)

	// GetStorageInfo returns storage quota and usage information.
	// This can be used to monitor storage consumption and plan capacity.
	GetStorageInfo(ctx context.Context) (*models.StorageInfo, error)
}

// WorkerServiceInterface defines the contract for worker pool management.
// This interface provides insights into the worker system's health,
// active workspaces, and resource utilization.
//
// Example usage:
//
//	// Check worker health
//	health, err := client.Worker.Health(ctx)
//	if health.Status != "healthy" {
//		log.Printf("Worker pool degraded: %s", health.Status)
//	}
//
//	// Clean up old workspaces
//	cleanup, err := client.Worker.CleanupOldWorkspaces(ctx, 24) // 24 hours
//	log.Printf("Cleaned %d workspaces, freed %d bytes",
//		cleanup.CleanedCount, cleanup.TotalSizeFreed)
type WorkerServiceInterface interface {
	// Health checks the overall health of the worker pool system.
	// Returns information about active workers, queue size, and system status.
	//
	// Status can be "healthy", "degraded", or "unhealthy".
	Health(ctx context.Context) (*models.WorkerHealthResponse, error)

	// Stats returns detailed statistics about worker pool performance.
	// This includes metrics like jobs processed, average processing time,
	// and resource utilization.
	Stats(ctx context.Context) (*models.WorkerStatsResponse, error)

	// ListWorkspaces returns a list of active worker workspaces.
	// Workspaces are temporary directories created for each job.
	//
	// Supports filtering by status and pagination.
	ListWorkspaces(ctx context.Context, opts *ListWorkspacesOptions) (*models.WorkspaceListResponse, error)

	// GetWorkspace returns detailed information about a specific workspace.
	// This includes disk usage, activity status, and workspace metadata.
	GetWorkspace(ctx context.Context, jobID string) (*models.WorkspaceInfoResponse, error)

	// DeleteWorkspace manually removes a job's workspace and frees disk space.
	// This is typically done automatically, but can be useful for cleanup
	// or troubleshooting.
	//
	// Returns information about the cleanup operation including freed space.
	DeleteWorkspace(ctx context.Context, jobID string) (*models.WorkspaceCleanupResponse, error)

	// CleanupOldWorkspaces removes workspaces older than the specified age.
	// This helps maintain disk space by cleaning up abandoned or old workspaces.
	//
	// maxAgeHours: workspaces older than this (in hours) will be removed.
	// Set to 0 to use the server's default age threshold.
	CleanupOldWorkspaces(ctx context.Context, maxAgeHours int) (*models.WorkspaceCleanupBatchResponse, error)
}

// ThemesServiceInterface defines the contract for Slidev theme management.
// Themes control the visual appearance and layout of generated presentations.
//
// Example usage:
//
//	// List available themes
//	themes, err := client.Themes.ListAvailable(ctx)
//	for _, theme := range themes.Themes {
//		fmt.Printf("Theme: %s v%s - %s\n",
//			theme.Name, theme.Version, theme.Description)
//	}
//
//	// Auto-install themes for a job
//	result, err := client.Themes.AutoInstallForJob(ctx, jobID)
//	fmt.Printf("Installed %d themes successfully\n", result.Successful)
type ThemesServiceInterface interface {
	// ListAvailable returns all Slidev themes available for installation.
	// This includes both official themes and custom themes registered
	// with the OCF Worker instance.
	ListAvailable(ctx context.Context) (*models.ThemeListResponse, error)

	// Install manually installs a specific Slidev theme by name.
	// The theme will be downloaded and made available for use in presentations.
	//
	// Returns installation status and any error messages.
	Install(ctx context.Context, themeName string) (*models.ThemeInstallResponse, error)

	// DetectForJob analyzes a job's source files to determine which themes
	// are required but not yet installed.
	//
	// This is useful for understanding dependencies before job execution.
	DetectForJob(ctx context.Context, jobID string) (*models.ThemeDetectionResponse, error)

	// AutoInstallForJob automatically detects and installs all themes
	// required by a job's source files.
	//
	// This is typically called after uploading sources but before starting
	// job processing. Returns detailed results for each theme installation attempt.
	AutoInstallForJob(ctx context.Context, jobID string) (*models.ThemeAutoInstallResponse, error)
}

// HealthServiceInterface defines the contract for service health monitoring.
// This provides system-wide health information beyond just the worker pool.
//
// Example usage:
//
//	health, err := client.Health.Check(ctx)
//	if health.Status != "healthy" {
//		log.Printf("Service unhealthy: %+v", health)
//		// Handle degraded service...
//	}
type HealthServiceInterface interface {
	// Check performs a comprehensive health check of the OCF Worker service.
	// This includes database connectivity, storage availability, worker pool status,
	// and other critical system components.
	//
	// The response includes overall status and details about individual components.
	// HTTP status may be 200 (healthy) or 503 (unhealthy), but both return
	// a valid HealthResponse.
	Check(ctx context.Context) (*models.HealthResponse, error)
}

// ArchiveServiceInterface defines the contract for course archive operations.
// Archives provide a convenient way to download all results from a course
// in a single compressed file.
//
// Example usage:
//
//	// Download course archive as ZIP
//	reader, err := client.Archive.DownloadArchive(ctx, courseID, &ocfworker.DownloadArchiveOptions{
//		Format:   "zip",
//		Compress: &[]bool{true}[0],
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer reader.Close()
//
//	// Save to file
//	file, err := os.Create("course-results.zip")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer file.Close()
//	io.Copy(file, reader)
type ArchiveServiceInterface interface {
	// DownloadArchive creates and downloads a compressed archive containing
	// all result files for a course.
	//
	// Supports multiple formats (zip, tar) and compression options.
	// Returns a ReadCloser that must be closed by the caller.
	//
	// The archive includes all generated files: PDFs, HTML, images, etc.
	DownloadArchive(ctx context.Context, courseID string, opts *DownloadArchiveOptions) (io.ReadCloser, error)
}

// Compile-time interface compliance checks.
// These ensure that concrete implementations actually satisfy their interfaces.
// If an implementation doesn't match its interface, compilation will fail here.
var (
	_ JobsServiceInterface    = (*JobsService)(nil)
	_ StorageServiceInterface = (*StorageService)(nil)
	_ WorkerServiceInterface  = (*WorkerService)(nil)
	_ ThemesServiceInterface  = (*ThemesService)(nil)
	_ HealthServiceInterface  = (*HealthService)(nil)
	_ ArchiveServiceInterface = (*ArchiveService)(nil)
)
