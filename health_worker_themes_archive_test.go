package ocfworker

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HealthService tests
func TestHealthService_Check(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("healthy service", func(t *testing.T) {
		expectedHealth := MockHealthResponse("healthy")

		server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedHealth)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		health, err := client.Health.Check(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedHealth.Status, health.Status)
	})

	t.Run("unhealthy service", func(t *testing.T) {
		expectedHealth := MockHealthResponse("unhealthy")

		server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusServiceUnavailable, expectedHealth)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		health, err := client.Health.Check(ctx)

		require.NoError(t, err) // Service returns response even for 503
		assert.Equal(t, expectedHealth.Status, health.Status)
	})

	t.Run("invalid response", func(t *testing.T) {
		server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid json"))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Health.Check(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")
	})

	t.Run("unexpected status code", func(t *testing.T) {
		server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusBadRequest, "Bad request")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		health, err := client.Health.Check(ctx)

		require.Error(t, err)
		assert.NotNil(t, health) // Health object is still returned
		AssertAPIError(t, err, http.StatusBadRequest, "Bad request")
	})
}

// ThemesService tests
func TestThemesService_ListAvailable(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful theme list", func(t *testing.T) {
		expectedThemes := &models.ThemeListResponse{
			Themes: []models.ThemeInfo{
				{Name: "default", Version: "1.0.0", Description: "Default theme"},
				{Name: "academic", Version: "2.1.0", Description: "Academic presentation theme"},
				{Name: "corporate", Version: "1.5.0", Description: "Corporate theme"},
			},
			Count: 3,
		}

		server.On("GET", "/api/v1/themes/available", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedThemes)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		themes, err := client.Themes.ListAvailable(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedThemes.Count, themes.Count)
		assert.Len(t, themes.Themes, 3)
		assert.Equal(t, "default", themes.Themes[0].Name)
		assert.Equal(t, "academic", themes.Themes[1].Name)
		assert.Equal(t, "corporate", themes.Themes[2].Name)
	})

	t.Run("empty theme list", func(t *testing.T) {
		expectedThemes := &models.ThemeListResponse{
			Themes: []models.ThemeInfo{},
			Count:  0,
		}

		server.On("GET", "/api/v1/themes/available", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedThemes)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		themes, err := client.Themes.ListAvailable(ctx)

		require.NoError(t, err)
		assert.Equal(t, 0, themes.Count)
		assert.Empty(t, themes.Themes)
	})

	t.Run("service error", func(t *testing.T) {
		server.On("GET", "/api/v1/themes/available", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusServiceUnavailable, "Theme service unavailable")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Themes.ListAvailable(ctx)

		AssertAPIError(t, err, http.StatusServiceUnavailable, "Theme service unavailable")
	})
}

func TestThemesService_Install(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful theme installation", func(t *testing.T) {
		themeName := "academic"
		expectedResponse := &models.ThemeInstallResponse{
			Theme:   themeName,
			Success: true,
			Message: "Theme installed successfully",
		}

		server.On("POST", "/api/v1/themes/install", func(w http.ResponseWriter, r *http.Request) {
			AssertContentType(t, r, "application/json")

			var req models.ThemeInstallRequest
			ReadJSONBody(t, r, &req)
			assert.Equal(t, themeName, req.Theme)

			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Themes.Install(ctx, themeName)

		require.NoError(t, err)
		assert.Equal(t, themeName, result.Theme)
		assert.True(t, result.Success)
		assert.Equal(t, "Theme installed successfully", result.Message)
	})

	t.Run("theme installation failure", func(t *testing.T) {
		themeName := "nonexistent"
		expectedResponse := &models.ThemeInstallResponse{
			Theme:   themeName,
			Success: false,
			Message: "Theme not found",
		}

		server.On("POST", "/api/v1/themes/install", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Themes.Install(ctx, themeName)

		require.NoError(t, err)
		assert.Equal(t, themeName, result.Theme)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "not found")
	})

	t.Run("invalid theme name", func(t *testing.T) {
		server.On("POST", "/api/v1/themes/install", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusBadRequest, "Invalid theme name")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Themes.Install(ctx, "")

		AssertAPIError(t, err, http.StatusBadRequest, "Invalid theme name")
	})
}

func TestThemesService_DetectForJob(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful theme detection", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedResponse := &models.ThemeDetectionResponse{
			MissingThemes:   []string{"academic", "custom-theme"},
			InstalledThemes: []models.ThemeInfo{},
			DetectedCount:   2,
		}

		server.On("GET", "/api/v1/themes/jobs/"+jobID+"/detect", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Themes.DetectForJob(ctx, jobID)

		require.NoError(t, err)
		assert.Len(t, result.DetectedCount, 2)
		assert.Contains(t, result.MissingThemes, "academic")
		assert.Contains(t, result.MissingThemes, "custom-theme")
	})

	t.Run("job not found", func(t *testing.T) {
		jobID := uuid.New().String()

		server.On("GET", "/api/v1/themes/jobs/"+jobID+"/detect", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Job not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Themes.DetectForJob(ctx, jobID)

		AssertAPIError(t, err, http.StatusNotFound, "Job not found")
	})
}

func TestThemesService_AutoInstallForJob(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful auto-installation", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedResponse := &models.ThemeAutoInstallResponse{
			Successful: 2,
			Results: []models.ThemeInstallResult{
				{Theme: "academic", Success: true, Logs: []string{"Installed successfully"}},
				{Theme: "custom", Success: true, Logs: []string{"Installed successfully"}},
			},
		}

		server.On("POST", "/api/v1/themes/jobs/"+jobID+"/install", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Themes.AutoInstallForJob(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, 2, result.Successful)
		assert.Len(t, result.Results, 2)
		assert.True(t, result.Results[0].Success)
		assert.True(t, result.Results[1].Success)
	})

	t.Run("partial installation success", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedResponse := &models.ThemeAutoInstallResponse{
			Successful: 0, // Will be calculated from Results
			Results: []models.ThemeInstallResult{
				{Theme: "academic", Success: true, Logs: []string{"Installed successfully"}},
				{Theme: "broken-theme", Success: false, Logs: []string{"Installation failed"}},
			},
		}

		server.On("POST", "/api/v1/themes/jobs/"+jobID+"/install", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Themes.AutoInstallForJob(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, 2, result.Successful) // Calculated from len(Results)
		assert.Len(t, result.Results, 2)
		assert.True(t, result.Results[0].Success)
		assert.False(t, result.Results[1].Success)
	})

	t.Run("no themes to install", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedResponse := &models.ThemeAutoInstallResponse{
			Successful: 0,
			Results:    []models.ThemeInstallResult{},
		}

		server.On("POST", "/api/v1/themes/jobs/"+jobID+"/install", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Themes.AutoInstallForJob(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, 0, result.Successful)
		assert.Empty(t, result.Results)
	})
}

// WorkerService tests
func TestWorkerService_Health(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("healthy workers", func(t *testing.T) {
		expectedHealth := &models.WorkerHealthResponse{
			WorkerPool: models.WorkerPoolHealth{
				ActiveWorkers: 3,
				WorkerCount:   5,
				QueueSize:     2,
			},
			Status: "healthy",
		}

		server.On("GET", "/api/v1/worker/health", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedHealth)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		health, err := client.Worker.Health(ctx)

		require.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, 3, health.WorkerPool.ActiveWorkers)
		assert.Equal(t, 5, health.WorkerPool.WorkerCount)
		assert.Equal(t, 2, health.WorkerPool.QueueSize)
	})

	t.Run("degraded workers", func(t *testing.T) {
		expectedHealth := &models.WorkerHealthResponse{
			WorkerPool: models.WorkerPoolHealth{
				ActiveWorkers: 1,
				WorkerCount:   5,
				QueueSize:     10,
			},
			Status: "degraded",
		}

		server.On("GET", "/api/v1/worker/health", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusServiceUnavailable, expectedHealth)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		health, err := client.Worker.Health(ctx)

		require.NoError(t, err) // Don't error on 503
		assert.Equal(t, "degraded", health.Status)
		assert.Equal(t, 1, health.WorkerPool.ActiveWorkers)
	})
}

func TestWorkerService_ListWorkspaces(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("list workspaces without filters", func(t *testing.T) {
		expectedWorkspaces := &models.WorkspaceListResponse{
			Workspaces: []models.WorkspaceInfo{
				{JobID: uuid.New().String()},
				{JobID: uuid.New().String()},
			},
			TotalCount: 2,
			PageSize:   10,
			Page:       0,
		}

		server.On("GET", "/api/v1/worker/workspaces", func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.URL.RawQuery)
			RespondJSON(w, http.StatusOK, expectedWorkspaces)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		workspaces, err := client.Worker.ListWorkspaces(ctx, nil)

		require.NoError(t, err)
		assert.Equal(t, 2, workspaces.TotalCount)
		assert.Len(t, workspaces.Workspaces, 2)
	})

	t.Run("list workspaces with filters", func(t *testing.T) {
		opts := &ListWorkspacesOptions{
			Status: "active",
			Limit:  5,
			Offset: 10,
		}

		server.On("GET", "/api/v1/worker/workspaces", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, "active", query.Get("status"))
			assert.Equal(t, "5", query.Get("limit"))
			assert.Equal(t, "10", query.Get("offset"))

			RespondJSON(w, http.StatusOK, &models.WorkspaceListResponse{})
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Worker.ListWorkspaces(ctx, opts)

		require.NoError(t, err)
	})
}

func TestWorkerService_GetWorkspace(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful workspace retrieval", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedWorkspace := &models.WorkspaceInfoResponse{
			Workspace: models.WorkspaceInfo{
				JobID: jobID,
			},
			Activity: models.WorkspaceActivity{
				Status:    "active",
				CreatedAt: time.Date(int(2024), time.January, int(1), int(10), int(0), int(0), int(0), time.UTC),
			},
			Usage: models.WorkspaceUsage{
				DiskUsage: models.StorageUsage{
					TotalBytes: 1024000,
				},
			},
		}

		server.On("GET", "/api/v1/worker/workspaces/"+jobID, func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedWorkspace)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		workspace, err := client.Worker.GetWorkspace(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, jobID, workspace.Workspace.JobID)
		assert.Equal(t, "active", workspace.Activity.Status)
		assert.Equal(t, int64(1024000), workspace.Usage.DiskUsage.TotalBytes)
	})

	t.Run("workspace not found", func(t *testing.T) {
		jobID := uuid.New().String()

		server.On("GET", "/api/v1/worker/workspaces/"+jobID, func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Workspace not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Worker.GetWorkspace(ctx, jobID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found for job")
	})
}

func TestWorkerService_DeleteWorkspace(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful workspace deletion", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedResponse := &models.WorkspaceCleanupResponse{
			JobID:     jobID,
			Cleaned:   true,
			SizeFreed: 1024000,
		}

		server.On("DELETE", "/api/v1/worker/workspaces/"+jobID, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Worker.DeleteWorkspace(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, jobID, result.JobID)
		assert.True(t, result.Cleaned)
		assert.Equal(t, int64(1024000), result.SizeFreed)
	})

	t.Run("workspace not found for deletion", func(t *testing.T) {
		jobID := uuid.New().String()

		server.On("DELETE", "/api/v1/worker/workspaces/"+jobID, func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Workspace not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Worker.DeleteWorkspace(ctx, jobID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found for job")
	})
}

func TestWorkerService_CleanupOldWorkspaces(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful cleanup", func(t *testing.T) {
		maxAgeHours := 24
		expectedResponse := &models.WorkspaceCleanupBatchResponse{
			CleanedCount:   5,
			TotalSizeFreed: 5120000,
		}

		server.On("POST", "/api/v1/worker/workspaces/cleanup", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, "24", query.Get("max_age_hours"))
			RespondJSON(w, http.StatusOK, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Worker.CleanupOldWorkspaces(ctx, maxAgeHours)

		require.NoError(t, err)
		assert.Equal(t, 5, result.CleanedCount)
		assert.Equal(t, int64(5120000), result.TotalSizeFreed)
	})

	t.Run("cleanup with default age", func(t *testing.T) {
		server.On("POST", "/api/v1/worker/workspaces/cleanup", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Empty(t, query.Get("max_age_hours"))
			RespondJSON(w, http.StatusOK, &models.WorkspaceCleanupBatchResponse{})
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Worker.CleanupOldWorkspaces(ctx, 0)

		require.NoError(t, err)
	})
}

// ArchiveService tests
func TestArchiveService_DownloadArchive(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful archive download", func(t *testing.T) {
		courseID := uuid.New().String()
		archiveContent := "ZIP archive binary content"

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/archive", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, "zip", query.Get("format"))
			assert.Equal(t, "true", query.Get("compress"))

			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(archiveContent))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		opts := &DownloadArchiveOptions{
			Format:   "zip",
			Compress: &[]bool{true}[0],
		}

		reader, err := client.Archive.DownloadArchive(ctx, courseID, opts)

		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, archiveContent, string(content))
	})

	t.Run("archive download with tar format", func(t *testing.T) {
		courseID := uuid.New().String()

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/archive", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Equal(t, "tar", query.Get("format"))
			assert.Equal(t, "false", query.Get("compress"))

			w.Header().Set("Content-Type", "application/x-tar")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("TAR archive content"))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		opts := &DownloadArchiveOptions{
			Format:   "tar",
			Compress: &[]bool{false}[0],
		}

		reader, err := client.Archive.DownloadArchive(ctx, courseID, opts)

		require.NoError(t, err)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "TAR archive content", string(content))
	})

	t.Run("archive download without options", func(t *testing.T) {
		courseID := uuid.New().String()

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/archive", func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.URL.RawQuery)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("default archive"))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		reader, err := client.Archive.DownloadArchive(ctx, courseID, nil)

		require.NoError(t, err)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "default archive", string(content))
	})

	t.Run("course not found", func(t *testing.T) {
		courseID := uuid.New().String()

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/archive", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Course not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Archive.DownloadArchive(ctx, courseID, nil)

		AssertAPIError(t, err, http.StatusNotFound, "Course not found")
	})

	t.Run("archive generation error", func(t *testing.T) {
		courseID := uuid.New().String()

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/archive", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusInternalServerError, "Failed to generate archive")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Archive.DownloadArchive(ctx, courseID, nil)

		AssertAPIError(t, err, http.StatusInternalServerError, "Failed to generate archive")
	})
}

func TestDownloadArchiveOptions(t *testing.T) {
	// Test URL parameter encoding
	opts := &DownloadArchiveOptions{
		Format:   "zip",
		Compress: &[]bool{true}[0],
	}

	params := url.Values{}
	if opts.Format != "" {
		params.Set("format", opts.Format)
	}
	if opts.Compress != nil {
		params.Set("compress", "true")
	}

	expected := "compress=true&format=zip"
	assert.Equal(t, expected, params.Encode())
}

// Benchmark tests for remaining services
func BenchmarkHealthService_Check(b *testing.B) {
	server := NewTestServer()
	defer server.Close()

	server.On("GET", "/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, MockHealthResponse("healthy"))
	})

	client := server.TestClient()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Health.Check(ctx)
	}
}

func BenchmarkThemesService_ListAvailable(b *testing.B) {
	server := NewTestServer()
	defer server.Close()

	themes := &models.ThemeListResponse{
		Themes: []models.ThemeInfo{
			{Name: "default", Version: "1.0.0"},
			{Name: "academic", Version: "2.0.0"},
		},
		Count: 2,
	}

	server.On("GET", "/api/v1/themes/available", func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, themes)
	})

	client := server.TestClient()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Themes.ListAvailable(ctx)
	}
}
