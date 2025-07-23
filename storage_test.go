package ocfworker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageService_UploadSources(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful upload", func(t *testing.T) {
		jobID := uuid.New().String()
		files := []FileUpload{
			MockFileUpload("test1.md", "# Test 1\nContent 1"),
			MockFileUpload("test2.md", "# Test 2\nContent 2"),
		}

		expectedResponse := &models.FileUploadResponse{
			Count:   2,
			Message: "Files uploaded successfully",
		}

		server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			// Verify content type
			contentType := r.Header.Get("Content-Type")
			assert.Contains(t, contentType, "multipart/form-data")

			// Parse multipart form
			err := r.ParseMultipartForm(32 << 20) // 32MB
			require.NoError(t, err)

			assert.Len(t, r.MultipartForm.File["files"], 2)

			// Verify first file
			fileHeader1 := r.MultipartForm.File["files"][0]
			assert.Equal(t, "test1.md", fileHeader1.Filename)

			file1, err := fileHeader1.Open()
			require.NoError(t, err)
			defer file1.Close()

			content1, err := io.ReadAll(file1)
			require.NoError(t, err)
			assert.Equal(t, "# Test 1\nContent 1", string(content1))

			// Verify second file
			fileHeader2 := r.MultipartForm.File["files"][1]
			assert.Equal(t, "test2.md", fileHeader2.Filename)

			file2, err := fileHeader2.Open()
			require.NoError(t, err)
			defer file2.Close()

			content2, err := io.ReadAll(file2)
			require.NoError(t, err)
			assert.Equal(t, "# Test 2\nContent 2", string(content2))

			RespondJSON(w, http.StatusCreated, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.UploadSources(ctx, jobID, files)

		require.NoError(t, err)
		assert.Equal(t, expectedResponse.Count, result.Count)
		assert.Equal(t, expectedResponse.Message, result.Message)
	})

	t.Run("empty file list", func(t *testing.T) {
		jobID := uuid.New().String()
		files := []FileUpload{}

		expectedResponse := &models.FileUploadResponse{
			Count:   0,
			Message: "No files uploaded",
		}

		server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(32 << 20)
			require.NoError(t, err)

			assert.Empty(t, r.MultipartForm.File["files"])
			RespondJSON(w, http.StatusCreated, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.UploadSources(ctx, jobID, files)

		require.NoError(t, err)
		assert.Equal(t, 0, result.Count)
	})

	t.Run("upload error", func(t *testing.T) {
		jobID := uuid.New().String()
		files := []FileUpload{MockFileUpload("test.md", "content")}

		server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusInsufficientStorage, "Storage quota exceeded")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Storage.UploadSources(ctx, jobID, files)

		AssertAPIError(t, err, http.StatusInsufficientStorage, "Storage quota exceeded")
	})

	t.Run("invalid job ID", func(t *testing.T) {
		jobID := "invalid-uuid"
		files := []FileUpload{MockFileUpload("test.md", "content")}

		server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusBadRequest, "Invalid job ID format")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Storage.UploadSources(ctx, jobID, files)

		AssertAPIError(t, err, http.StatusBadRequest, "Invalid job ID format")
	})
}

func TestStorageService_UploadSourcesStream(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful streaming upload", func(t *testing.T) {
		jobID := uuid.New().String()
		uploads := []StreamUpload{
			{
				Name:        "stream1.md",
				Reader:      strings.NewReader("# Stream 1\nContent from reader"),
				Size:        int64(len("# Stream 1\nContent from reader")),
				ContentType: "text/markdown",
			},
			{
				Name:        "stream2.txt",
				Reader:      strings.NewReader("Plain text content"),
				Size:        int64(len("Plain text content")),
				ContentType: "text/plain",
			},
		}

		expectedResponse := &models.FileUploadResponse{
			Count:   2,
			Message: "Streaming upload completed",
		}

		server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			assert.Contains(t, contentType, "multipart/form-data")

			// Parse the multipart form
			err := r.ParseMultipartForm(32 << 20)
			require.NoError(t, err)

			files := r.MultipartForm.File["files"]
			assert.Len(t, files, 2)

			// Verify files were uploaded correctly
			for i, fileHeader := range files {
				file, err := fileHeader.Open()
				require.NoError(t, err)
				defer file.Close()

				content, err := io.ReadAll(file)
				require.NoError(t, err)

				if i == 0 {
					assert.Equal(t, "stream1.md", fileHeader.Filename)
					assert.Equal(t, "# Stream 1\nContent from reader", string(content))
				} else {
					assert.Equal(t, "stream2.txt", fileHeader.Filename)
					assert.Equal(t, "Plain text content", string(content))
				}
			}

			RespondJSON(w, http.StatusCreated, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.UploadSourcesStream(ctx, jobID, uploads)

		require.NoError(t, err)
		assert.Equal(t, 2, result.Count)
	})

	t.Run("streaming error", func(t *testing.T) {
		jobID := uuid.New().String()
		uploads := []StreamUpload{
			{
				Name:        "test.md",
				Reader:      &errorReader{},
				Size:        100,
				ContentType: "text/markdown",
			},
		}

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Storage.UploadSourcesStream(ctx, jobID, uploads)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "simulated read error")
	})
}

func TestStorageService_UploadSourceFiles(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "test1.md")
	file2 := filepath.Join(tempDir, "test2.txt")

	err := os.WriteFile(file1, []byte("# Test File 1\nMarkdown content"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(file2, []byte("Plain text file content"), 0644)
	require.NoError(t, err)

	server := NewTestServer()
	defer server.Close()

	t.Run("successful file upload", func(t *testing.T) {
		jobID := uuid.New().String()
		filePaths := []string{file1, file2}

		expectedResponse := &models.FileUploadResponse{
			Count:   2,
			Message: "Files uploaded from filesystem",
		}

		server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(32 << 20)
			require.NoError(t, err)

			files := r.MultipartForm.File["files"]
			assert.Len(t, files, 2)

			// Check file names and content
			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				require.NoError(t, err)
				defer file.Close()

				content, err := io.ReadAll(file)
				require.NoError(t, err)

				if fileHeader.Filename == "test1.md" {
					assert.Equal(t, "# Test File 1\nMarkdown content", string(content))
				} else if fileHeader.Filename == "test2.txt" {
					assert.Equal(t, "Plain text file content", string(content))
				}
			}

			RespondJSON(w, http.StatusCreated, expectedResponse)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.UploadSourceFiles(ctx, jobID, filePaths)

		require.NoError(t, err)
		assert.Equal(t, 2, result.Count)
	})

}

func TestStorageService_ListSources(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful list", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedFiles := MockFileList("source1.md", "source2.txt", "slides.md")

		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedFiles)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.ListSources(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, expectedFiles.Count, result.Count)
		assert.Equal(t, expectedFiles.Files, result.Files)
	})

	t.Run("job not found", func(t *testing.T) {
		jobID := uuid.New().String()

		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Job not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Storage.ListSources(ctx, jobID)

		AssertAPIError(t, err, http.StatusNotFound, "Job not found")
	})

	t.Run("empty source list", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedFiles := MockFileList() // Empty list

		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedFiles)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.ListSources(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, 0, result.Count)
		assert.Empty(t, result.Files)
	})
}

func TestStorageService_DownloadSource(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful download", func(t *testing.T) {
		jobID := uuid.New().String()
		filename := "test.md"
		content := "# Test Content\nThis is a test file"

		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/sources/"+filename, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/markdown")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		reader, err := client.Storage.DownloadSource(ctx, jobID, filename)

		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		downloaded, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, string(downloaded))
	})
}

func TestStorageService_ListResults(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful list results", func(t *testing.T) {
		courseID := uuid.New().String()
		expectedFiles := &models.FileListResponse{
			Files: []string{"presentation.pdf", "slides.html", "notes.txt"},
			Count: 0, // Will be calculated
		}

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/results", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedFiles)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.ListResults(ctx, courseID)

		require.NoError(t, err)
		assert.Equal(t, 3, result.Count) // Should be calculated from len(Files)
		assert.Len(t, result.Files, 3)
		assert.Contains(t, result.Files, "presentation.pdf")
		assert.Contains(t, result.Files, "slides.html")
		assert.Contains(t, result.Files, "notes.txt")
	})

	t.Run("course not found", func(t *testing.T) {
		courseID := uuid.New().String()

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/results", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Course not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Storage.ListResults(ctx, courseID)

		AssertAPIError(t, err, http.StatusNotFound, "Course not found")
	})

	t.Run("empty results", func(t *testing.T) {
		courseID := uuid.New().String()
		expectedFiles := &models.FileListResponse{
			Files: []string{},
			Count: 0,
		}

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/results", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, expectedFiles)
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		result, err := client.Storage.ListResults(ctx, courseID)

		require.NoError(t, err)
		assert.Equal(t, 0, result.Count)
		assert.Empty(t, result.Files)
	})
}

func TestStorageService_DownloadResult(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful download", func(t *testing.T) {
		courseID := uuid.New().String()
		filename := "presentation.pdf"
		content := "PDF content here"

		server.On("GET", "/api/v1/storage/courses/"+courseID+"/results/"+filename, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		reader, err := client.Storage.DownloadResult(ctx, courseID, filename)

		require.NoError(t, err)
		require.NotNil(t, reader)
		defer reader.Close()

		downloaded, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, string(downloaded))
	})

}

func TestStorageService_GetLogs(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("successful log retrieval", func(t *testing.T) {
		jobID := uuid.New().String()
		expectedLogs := `
2024-01-01 10:00:00 [INFO] Job started
2024-01-01 10:00:01 [INFO] Processing sources
2024-01-01 10:00:05 [INFO] Generating slides
2024-01-01 10:00:10 [INFO] Job completed successfully
`

		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/logs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedLogs))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		logs, err := client.Storage.GetLogs(ctx, jobID)

		require.NoError(t, err)
		assert.Equal(t, expectedLogs, logs)
		assert.Contains(t, logs, "Job started")
		assert.Contains(t, logs, "Job completed successfully")
	})

	t.Run("job not found", func(t *testing.T) {
		jobID := uuid.New().String()

		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/logs", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusNotFound, "Job not found")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Storage.GetLogs(ctx, jobID)

		AssertAPIError(t, err, http.StatusNotFound, "Job not found")
	})

	t.Run("empty logs", func(t *testing.T) {
		jobID := uuid.New().String()

		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/logs", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(""))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		logs, err := client.Storage.GetLogs(ctx, jobID)

		require.NoError(t, err)
		assert.Empty(t, logs)
	})
}

func TestStorageService_GetStorageInfo(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	t.Run("storage info unavailable", func(t *testing.T) {
		server.On("GET", "/api/v1/storage/info", func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, http.StatusServiceUnavailable, "Storage service unavailable")
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		_, err := client.Storage.GetStorageInfo(ctx)

		AssertAPIError(t, err, http.StatusServiceUnavailable, "Storage service unavailable")
	})
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename     string
		expectedType string
	}{
		{"test.md", "text/markdown"},
		{"style.css", "text/css"},
		{"script.js", "application/javascript"},
		{"data.json", "application/json"},
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"picture.jpeg", "image/jpeg"},
		{"unknown.xyz", "application/octet-stream"},
		{"no-extension", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detectContentType(tt.filename)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

// Helper types for testing

// errorReader simulates a reader that always returns an error
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

// Test integration scenarios
func TestStorageService_IntegrationScenario(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	jobID := uuid.New().String()
	courseID := uuid.New().String()

	// Setup complete workflow
	t.Run("complete storage workflow", func(t *testing.T) {
		// 1. Upload sources
		files := []FileUpload{
			MockFileUpload("slides.md", "# My Presentation\n## Slide 1\nContent"),
			MockFileUpload("config.json", `{"theme": "default"}`),
		}

		server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusCreated, &models.FileUploadResponse{Count: 2})
		})

		// 2. List uploaded sources
		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, MockFileList("slides.md", "config.json"))
		})

		// 3. Download a source file
		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/sources/slides.md", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# My Presentation\n## Slide 1\nContent"))
		})

		// 4. Get logs
		server.On("GET", "/api/v1/storage/jobs/"+jobID+"/logs", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Processing started\nGeneration completed"))
		})

		// 5. List results
		server.On("GET", "/api/v1/storage/courses/"+courseID+"/results", func(w http.ResponseWriter, r *http.Request) {
			RespondJSON(w, http.StatusOK, MockFileList("presentation.pdf", "slides.html"))
		})

		// 6. Download result
		server.On("GET", "/api/v1/storage/courses/"+courseID+"/results/presentation.pdf", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("PDF content"))
		})

		client := server.TestClient()
		ctx, _ := TestContext()

		// Execute workflow
		uploadResult, err := client.Storage.UploadSources(ctx, jobID, files)
		require.NoError(t, err)
		assert.Equal(t, 2, uploadResult.Count)

		sources, err := client.Storage.ListSources(ctx, jobID)
		require.NoError(t, err)
		assert.Len(t, sources.Files, 2)

		sourceReader, err := client.Storage.DownloadSource(ctx, jobID, "slides.md")
		require.NoError(t, err)
		sourceContent, _ := io.ReadAll(sourceReader)
		sourceReader.Close()
		assert.Contains(t, string(sourceContent), "My Presentation")

		logs, err := client.Storage.GetLogs(ctx, jobID)
		require.NoError(t, err)
		assert.Contains(t, logs, "Processing started")

		results, err := client.Storage.ListResults(ctx, courseID)
		require.NoError(t, err)
		assert.Equal(t, 2, results.Count)

		resultReader, err := client.Storage.DownloadResult(ctx, courseID, "presentation.pdf")
		require.NoError(t, err)
		resultContent, _ := io.ReadAll(resultReader)
		resultReader.Close()
		assert.Equal(t, "PDF content", string(resultContent))
	})
}

// Benchmark tests
func BenchmarkStorageService_UploadSources(b *testing.B) {
	server := NewTestServer()
	defer server.Close()

	jobID := uuid.New().String()
	files := []FileUpload{
		MockFileUpload("test.md", "content"),
	}

	server.On("POST", "/api/v1/storage/jobs/"+jobID+"/sources", func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusCreated, &models.FileUploadResponse{Count: 1})
	})

	client := server.TestClient()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Storage.UploadSources(ctx, jobID, files)
	}
}

func BenchmarkDetectContentType(b *testing.B) {
	filenames := []string{
		"test.md", "style.css", "script.js", "data.json",
		"image.png", "photo.jpg", "unknown.xyz", "no-extension",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, filename := range filenames {
			_ = detectContentType(filename)
		}
	}
}
