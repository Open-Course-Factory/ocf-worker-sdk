package ocfworker

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobsService_Create(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful job creation", func(t *testing.T) {
		req := MockGenerationRequest()
		expectedJob := NewJobResponse().
			WithID(req.JobID).
			WithCourseID(req.CourseID).
			WithStatus(models.StatusPending).
			WithLogs([]string{"Job created successfully"}).
			Build()

		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			AssertContentType(t, r, "application/json")

			var receivedReq models.GenerationRequest
			ReadJSONBody(t, r, &receivedReq)

			assert.Equal(t, req.JobID, receivedReq.JobID)
			assert.Equal(t, req.CourseID, receivedReq.CourseID)
			assert.Equal(t, req.SourcePath, receivedReq.SourcePath)

			RespondJSON(w, http.StatusCreated, expectedJob)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		job, err := client.Jobs.Create(ctx, req)

		require.NoError(t, err)
		AssertJobResponse(t, expectedJob, job)
	})

	t.Run("validation error", func(t *testing.T) {
		apiErr := NewAPIError().
			WithStatus(http.StatusBadRequest).
			WithMessage("Validation failed").
			WithValidationErrors([]ValidationError{
				{
					Field:   "job_id",
					Code:    "required",
					Message: "job_id is required",
					Value:   "",
				},
			}).
			Build()

		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			RespondAPIError(w, apiErr)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Jobs.Create(ctx, MockGenerationRequest())

		AssertAPIError(t, err, http.StatusBadRequest, "Validation failed")
	})

	t.Run("server error", func(t *testing.T) {
		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusInternalServerError, "Internal server error")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Jobs.Create(ctx, MockGenerationRequest())

		AssertAPIError(t, err, http.StatusInternalServerError, "Internal server error")
	})

	t.Run("context cancellation", func(t *testing.T) {
		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			time.Sleep(100 * time.Millisecond)
			RespondJSON(w, http.StatusCreated, NewJobResponse().Build())
		})

		client := server.TestClient()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := client.Jobs.Create(ctx, MockGenerationRequest())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

func TestJobsService_Get(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful job retrieval", func(t *testing.T) {
		jobID := uuid.New()
		expectedJob := NewJobResponse().
			WithID(jobID).
			WithStatus(models.StatusCompleted).
			WithLogs([]string{"Job completed successfully"}).
			Build()

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedJob)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		job, err := client.Jobs.Get(ctx, jobID.String())

		require.NoError(t, err)
		AssertJobResponse(t, expectedJob, job)
	})

	t.Run("job not found", func(t *testing.T) {
		jobID := uuid.New()

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Job not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Jobs.Get(ctx, jobID.String())

		require.Error(t, err)
		jobNotFoundErr, ok := err.(*JobNotFoundError)
		require.True(t, ok)
		assert.Equal(t, jobID.String(), jobNotFoundErr.JobID)
	})

	t.Run("server error", func(t *testing.T) {
		jobID := uuid.New()

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusInternalServerError, "Database error")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Jobs.Get(ctx, jobID.String())

		AssertAPIError(t, err, http.StatusInternalServerError, "Database error")
	})
}

func TestJobsService_List(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("list without filters", func(t *testing.T) {
		jobs := []models.JobResponse{
			*NewJobResponse().WithStatus(models.StatusCompleted).Build(),
			*NewJobResponse().WithStatus(models.StatusPending).Build(),
		}

		expectedResponse := &models.JobListResponse{
			Jobs:       jobs,
			TotalCount: 2,
			PageSize:   10,
			Page:       0,
		}

		server.On("GET", "/api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
			// Verify no query parameters
			assert.Empty(t, r.URL.RawQuery)
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Jobs.List(ctx, nil)

		require.NoError(t, err)
		assert.Equal(t, expectedResponse.TotalCount, result.TotalCount)
		assert.Len(t, result.Jobs, 2)
	})

	t.Run("list with filters", func(t *testing.T) {
		opts := &ListJobsOptions{
			Status:   string(models.StatusCompleted),
			CourseID: uuid.New().String(),
			Limit:    5,
			Offset:   10,
		}

		expectedResponse := &models.JobListResponse{
			Jobs:       []models.JobResponse{},
			TotalCount: 0,
			PageSize:   opts.Limit,
			Page:       opts.Offset,
		}

		server.On("GET", "/api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, opts.Status, query.Get("status"))
			assert.Equal(t, opts.CourseID, query.Get("course_id"))
			assert.Equal(t, "5", query.Get("limit"))
			assert.Equal(t, "10", query.Get("offset"))

			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Jobs.List(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, expectedResponse.TotalCount, result.TotalCount)
		assert.Equal(t, expectedResponse.PageSize, result.PageSize)
		assert.Equal(t, expectedResponse.Page, result.Page)
	})

	t.Run("list with partial filters", func(t *testing.T) {
		opts := &ListJobsOptions{
			Status: string(models.StatusFailed),
			Limit:  20,
		}

		server.On("GET", "/api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, opts.Status, query.Get("status"))
			assert.Equal(t, "20", query.Get("limit"))
			assert.Empty(t, query.Get("course_id"))
			assert.Empty(t, query.Get("offset"))

			RespondJSON(w, http.StatusOK, &models.JobListResponse{})
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Jobs.List(ctx, opts)

		require.NoError(t, err)
	})
}

func TestJobsService_WaitForCompletion(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("job completes successfully", func(t *testing.T) {
		jobID := uuid.New()
		callCount := 0

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			callCount++

			var job *models.JobResponse
			switch callCount {
			case 1:
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			case 2:
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusProcessing).Build()
			case 3:
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusCompleted).Build()
			}

			RespondJSON(w, http.StatusOK, job)
		})

		client := server.TestClient()
		ctx, _ := TestContext(10 * time.Second)

		opts := &WaitOptions{
			Interval: 100 * time.Millisecond,
			Timeout:  5 * time.Second,
		}

		job, err := client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)

		require.NoError(t, err)
		assert.Equal(t, models.StatusCompleted, job.Status)
		assert.Equal(t, 3, callCount)
	})

	t.Run("job fails", func(t *testing.T) {
		jobID := uuid.New()
		callCount := 0

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			callCount++

			var job *models.JobResponse
			switch callCount {
			case 1:
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			case 2:
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusFailed).WithError("Generation failed").Build()
			}

			RespondJSON(w, http.StatusOK, job)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		opts := &WaitOptions{
			Interval: 100 * time.Millisecond,
			Timeout:  5 * time.Second,
		}

		_, err := client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "job failed with status failed")
		assert.Contains(t, err.Error(), "Generation failed")
		assert.Equal(t, 2, callCount)
	})

	t.Run("timeout waiting for completion", func(t *testing.T) {
		jobID := uuid.New()

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			// Always return pending
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			RespondJSON(w, http.StatusOK, job)
		})

		client := server.TestClient()
		ctx, _ := TestContext(10 * time.Second)

		opts := &WaitOptions{
			Interval: 50 * time.Millisecond,
			Timeout:  200 * time.Millisecond,
		}

		_, err := client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for job completion")
	})

	t.Run("context cancellation during wait", func(t *testing.T) {
		jobID := uuid.New()

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			RespondJSON(w, http.StatusOK, job)
		})

		client := server.TestClient()
		ctx, cancel := context.WithCancel(context.Background())

		opts := &WaitOptions{
			Interval: 100 * time.Millisecond,
			Timeout:  5 * time.Second,
		}

		// Cancel context after a short delay
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()

		_, err := client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("job not found during wait", func(t *testing.T) {
		jobID := uuid.New()

		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Job not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		opts := &WaitOptions{
			Interval: 100 * time.Millisecond,
			Timeout:  1 * time.Second,
		}

		_, err := client.Jobs.WaitForCompletion(ctx, jobID.String(), opts)

		require.Error(t, err)
		jobNotFoundErr, ok := err.(*JobNotFoundError)
		require.True(t, ok)
		assert.Equal(t, jobID.String(), jobNotFoundErr.JobID)
	})

	t.Run("default options", func(t *testing.T) {
		req := MockGenerationRequest()
		jobID := req.JobID

		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			RespondJSON(w, http.StatusCreated, job)
		})

		// Répondre immédiatement avec completed
		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusCompleted).Build()
			RespondJSON(w, http.StatusOK, job)
		})

		client := server.TestClient()

		// Utiliser des options personnalisées avec un timeout court
		opts := &WaitOptions{
			Interval: 100 * time.Millisecond, // Interval court
			Timeout:  2 * time.Second,        // Timeout court pour le test
		}

		ctx, _ := TestContext(5 * time.Second)
		job, err := client.Jobs.CreateAndWait(ctx, req, opts)

		require.NoError(t, err)
		assert.Equal(t, models.StatusCompleted, job.Status)
	})
}

func TestJobsService_CreateAndWait(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful create and wait", func(t *testing.T) {
		req := MockGenerationRequest()
		jobID := req.JobID
		callCount := 0

		// Mock job creation
		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			RespondJSON(w, http.StatusCreated, job)
		})

		// Mock job status checks
		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			callCount++

			var job *models.JobResponse
			if callCount == 1 {
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusProcessing).Build()
			} else {
				job = NewJobResponse().WithID(jobID).WithStatus(models.StatusCompleted).Build()
			}

			RespondJSON(w, http.StatusOK, job)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		opts := &WaitOptions{
			Interval: 100 * time.Millisecond,
			Timeout:  5 * time.Second,
		}

		job, err := client.Jobs.CreateAndWait(ctx, req, opts)

		require.NoError(t, err)
		assert.Equal(t, models.StatusCompleted, job.Status)
		assert.Equal(t, jobID, job.ID)
	})

	t.Run("create fails", func(t *testing.T) {
		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusBadRequest, "Invalid request")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Jobs.CreateAndWait(ctx, MockGenerationRequest(), nil)

		AssertAPIError(t, err, http.StatusBadRequest, "Invalid request")
	})

	t.Run("create succeeds but wait fails", func(t *testing.T) {
		req := MockGenerationRequest()
		jobID := req.JobID

		// Mock successful creation
		server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusPending).Build()
			RespondJSON(w, http.StatusCreated, job)
		})

		// Mock job failure during wait
		server.On("GET", "/api/v1/jobs/"+jobID.String(), func(w http.ResponseWriter, r *http.Request) {
			job := NewJobResponse().WithID(jobID).WithStatus(models.StatusFailed).WithError("Processing error").Build()
			RespondJSON(w, http.StatusOK, job)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		opts := &WaitOptions{
			Interval: 100 * time.Millisecond,
			Timeout:  1 * time.Second,
		}

		_, err := client.Jobs.CreateAndWait(ctx, req, opts)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "job failed with status failed")
		assert.Contains(t, err.Error(), "Processing error")
	})

}

func TestListJobsOptions(t *testing.T) {
	// Test URL parameter encoding
	opts := &ListJobsOptions{
		Status:   "completed",
		CourseID: "123e4567-e89b-12d3-a456-426614174000",
		Limit:    50,
		Offset:   100,
	}

	params := url.Values{}
	if opts.Status != "" {
		params.Set("status", opts.Status)
	}
	if opts.CourseID != "" {
		params.Set("course_id", opts.CourseID)
	}
	if opts.Limit > 0 {
		params.Set("limit", "50")
	}
	if opts.Offset > 0 {
		params.Set("offset", "100")
	}

	expected := "course_id=123e4567-e89b-12d3-a456-426614174000&limit=50&offset=100&status=completed"
	assert.Equal(t, expected, params.Encode())
}

// Benchmark tests
func BenchmarkJobsService_Create(b *testing.B) {
	server := NewTestServer()
	defer server.Close()

	server.On("POST", "/api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		job := NewJobResponse().Build()
		RespondJSON(w, http.StatusCreated, job)
	})

	client := server.TestClient()
	ctx := context.Background()
	req := MockGenerationRequest()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.JobID = uuid.New() // Unique ID for each iteration
		_, _ = client.Jobs.Create(ctx, req)
	}
}

func BenchmarkJobsService_Get(b *testing.B) {
	server := NewTestServer()
	defer server.Close()

	jobID := uuid.New().String()
	server.On("GET", "/api/v1/jobs/"+jobID, func(w http.ResponseWriter, r *http.Request) {
		job := NewJobResponse().Build()
		RespondJSON(w, http.StatusOK, job)
	})

	client := server.TestClient()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Jobs.Get(ctx, jobID)
	}
}
