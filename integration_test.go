package ocfworker

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteWorkflow tests the entire flow from job creation to result download
func TestCompleteWorkflow(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	// Generate test IDs
	jobID := uuid.New()
	courseID := uuid.New()

	// Test files
	sourceFiles := []FileUpload{
		MockFileUpload("slides.md", "---\ntheme: default\n---\n\n# Test Presentation\n\n---\n\n## Slide 2\nContent"),
		MockFileUpload("style.css", "body { font-family: Arial; }"),
	}

	// Setup API responses
	setupCompleteWorkflowResponses(t, server, jobID, courseID)

	client := server.TestClient()
	ctx, _ := TestContext(30 * time.Second)

	t.Run("1. Check service health", func(t *testing.T) {
		health, err := client.Health.Check(ctx)
		require.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)
	})

	t.Run("2. Create job", func(t *testing.T) {
		req := &models.GenerationRequest{
			JobID:      jobID,
			CourseID:   courseID,
			SourcePath: "/sources",
		}

		job, err := client.Jobs.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, jobID, job.ID)
		assert.Equal(t, models.StatusPending, job.Status)
	})

	t.Run("3. Upload source files", func(t *testing.T) {
		uploadResp, err := client.Storage.UploadSources(ctx, jobID.String(), sourceFiles)
		require.NoError(t, err)
		assert.Equal(t, 2, uploadResp.Count)
	})

	t.Run("4. Wait for job completion", func(t *testing.T) {
		opts := &WaitOptions{
			Interval: 100 * time.Millisecond,
			Timeout:  10 * time.Second,
		}

		job, err := client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)
		require.NoError(t, err)
		assert.Equal(t, models.StatusCompleted, job.Status)
	})

	t.Run("5. List and download results", func(t *testing.T) {
		results, err := client.Storage.ListResults(ctx, courseID.String())
		require.NoError(t, err)
		assert.Contains(t, results.Files, "presentation.pdf")
		assert.Contains(t, results.Files, "slides.html")

		// Download PDF result
		pdfReader, err := client.Storage.DownloadResult(ctx, courseID.String(), "presentation.pdf")
		require.NoError(t, err)
		defer pdfReader.Close()

		pdfContent, err := io.ReadAll(pdfReader)
		require.NoError(t, err)
		assert.Contains(t, string(pdfContent), "PDF content")
	})

	t.Run("6. Download archive", func(t *testing.T) {
		archiveReader, err := client.Archive.DownloadArchive(ctx, courseID.String(), &DownloadArchiveOptions{
			Format:   "zip",
			Compress: &[]bool{true}[0],
		})
		require.NoError(t, err)
		defer archiveReader.Close()

		archiveContent, err := io.ReadAll(archiveReader)
		require.NoError(t, err)
		assert.NotEmpty(t, archiveContent)
	})
}

// TestErrorHandlingWorkflow tests how the SDK handles various error scenarios
func TestErrorHandlingWorkflow(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := server.TestClient()
	ctx, _ := TestContext()

	t.Run("invalid job creation", func(t *testing.T) {
		// Setup error response for job creation
		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			apiErr := NewAPIError().
				WithStatus(http.StatusBadRequest).
				WithMessage("Validation failed").
				WithValidationErrors([]ValidationError{
					{
						Field:   "course_id",
						Code:    "required",
						Message: "course_id is required",
					},
				}).
				Build()
			RespondAPIError(w, apiErr)
		})

		req := &models.GenerationRequest{
			JobID:      uuid.New(),
			SourcePath: "/sources",
			// Missing CourseID
		}

		_, err := client.Jobs.Create(ctx, req)
		require.Error(t, err)

		apiErr, ok := err.(*APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
		assert.Len(t, apiErr.Details, 1)
		assert.Equal(t, "course_id", apiErr.Details[0].Field)
	})

	t.Run("job fails during processing", func(t *testing.T) {
		jobID := uuid.New()
		callCount := 0

		// Create job successfully
		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			RespondJSON(w, http.StatusCreated, job)
		})

		// Simulate job failure
		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var job *models.JobResponse

			if callCount == 1 {
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusProcessing).Build()
			} else {
				job = NewJobResponse().
					WithID(jobID).
					WithStatus(models.StatusFailed).
					WithError("Theme compilation failed").
					Build()
			}

			RespondJSON(w, http.StatusOK, job)
		})

		req := MockGenerationRequest()
		req.JobID = jobID

		_, err := client.Jobs.Create(ctx, req)
		require.NoError(t, err)

		opts := &WaitOptions{
			Interval: 50 * time.Millisecond,
			Timeout:  2 * time.Second,
		}

		_, err = client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job failed with status failed")
		assert.Contains(t, err.Error(), "Theme compilation failed")
	})

	t.Run("storage errors", func(t *testing.T) {
		jobID := uuid.New()

		// Upload fails due to quota
		server.On("POST", "/api/v1/storage/jobs/"+jobID.String()+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusInsufficientStorage, "Storage quota exceeded")
		})

		files := []FileUpload{MockFileUpload("test.md", "content")}
		_, err := client.Storage.UploadSources(ctx, jobID.String(), files)

		AssertAPIError(t, err, http.StatusInsufficientStorage, "Storage quota exceeded")
	})

	t.Run("service unavailable", func(t *testing.T) {
		// Health check returns service unavailable
		server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			healthResp := MockHealthResponse("unhealthy")
			RespondJSON(w, http.StatusServiceUnavailable, healthResp)
		})

		health, err := client.Health.Check(ctx)
		require.NoError(t, err) // Health check doesn't return error for 503
		assert.Equal(t, "unhealthy", health.Status)
	})
}

// TestConcurrentRequests tests the SDK under concurrent load
func TestConcurrentRequests(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := server.TestClient()

	// Setup health endpoint
	server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		// Add small delay to simulate network latency
		time.Sleep(10 * time.Millisecond)
		RespondJSON(w, http.StatusOK, MockHealthResponse("healthy"))
	})

	t.Run("concurrent health checks", func(t *testing.T) {
		const numRequests = 50
		results := make(chan error, numRequests)

		// Launch concurrent requests
		for i := 0; i < numRequests; i++ {
			go func() {
				ctx, _ := TestContext()
				_, err := client.Health.Check(ctx)
				results <- err
			}()
		}

		// Collect results
		for i := 0; i < numRequests; i++ {
			err := <-results
			assert.NoError(t, err, "Request %d failed", i)
		}
	})

	t.Run("concurrent job operations", func(t *testing.T) {
		const numJobs = 10
		results := make(chan error, numJobs*2) // Create + Get operations

		// Setup job endpoints
		for i := 0; i < numJobs; i++ {
			jobID := uuid.New()

			server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(5 * time.Millisecond)
				job := NewJobResponse().WithID(jobID).Build()
				RespondJSON(w, http.StatusCreated, job)
			})

			server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(5 * time.Millisecond)
				job := NewJobResponse().WithID(jobID).Build()
				RespondJSON(w, http.StatusOK, job)
			})
		}

		// Launch concurrent operations
		for i := 0; i < numJobs; i++ {
			go func(index int) {
				ctx, _ := TestContext()
				req := MockGenerationRequest()

				// Create job
				job, err := client.Jobs.Create(ctx, req)
				results <- err

				if err == nil {
					// Get job
					_, err = client.Jobs.Get(ctx, job.ID.String())
					results <- err
				} else {
					results <- nil // Skip get if create failed
				}
			}(i)
		}

		// Collect results
		for i := 0; i < numJobs*2; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent operation %d failed: %v", i, err)
			}
		}
	})
}

// TestTimeoutHandling tests context timeout handling
func TestTimeoutHandling(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := server.TestClient()

	t.Run("request timeout", func(t *testing.T) {
		server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			time.Sleep(200 * time.Millisecond)
			RespondJSON(w, http.StatusOK, MockHealthResponse("healthy"))
		})

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := client.Health.Check(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("wait timeout", func(t *testing.T) {
		jobID := uuid.New()

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			// Always return pending status
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			RespondJSON(w, http.StatusOK, job)
		})

		ctx, _ := TestContext(5 * time.Second) // Contexte plus long
		opts := &WaitOptions{
			Interval: 50 * time.Millisecond,
			Timeout:  100 * time.Millisecond, // Plus court que le contexte
		}

		_, err := client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for job completion")
	})
}

// Helper function to setup responses for complete workflow test
func setupCompleteWorkflowResponses(t *testing.T, server *TestServer, jobID, courseID uuid.UUID) {
	// Health check
	server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, MockHealthResponse("healthy"))
	})

	// Job creation
	server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		job := NewJobResponse().
			WithID(jobID).
			WithCourseID(courseID).
			WithStatus(models.StatusPending).
			Build()
		RespondJSON(w, http.StatusCreated, job)
	})

	// File upload
	server.On("POST", "/api/v1/storage/jobs/"+jobID.String()+"/sources", func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusCreated, &models.FileUploadResponse{Count: 2})
	})

	// Job status polling (pending -> processing -> completed)
	callCount := 0
	server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var job *models.JobResponse

		switch callCount {
		case 1:
			job = NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
		case 2:
			job = NewJobResponse().WithID(jobID).WithStatus(models.StatusProcessing).Build()
		default:
			job = NewJobResponse().WithID(jobID).WithStatus(models.StatusCompleted).Build()
		}

		RespondJSON(w, http.StatusOK, job)
	})

	// List results
	server.On("GET", "/api/v1/storage/courses/"+courseID.String()+"/results", func(w http.ResponseWriter, r *http.Request) {
		results := MockFileList("presentation.pdf", "slides.html", "notes.txt")
		RespondJSON(w, http.StatusOK, results)
	})

	// Download result
	server.On("GET", "/api/v1/storage/courses/"+courseID.String()+"/results/presentation.pdf", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PDF content here"))
	})

	// Download archive
	server.On("GET", "/api/v1/storage/courses/"+courseID.String()+"/archive", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ZIP archive content"))
	})
}
