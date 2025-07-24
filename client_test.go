package ocfworker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/require"
)

// Mock implementations pour tester l'isolation des services

// MockJobsService implémente JobsServiceInterface pour les tests
type MockJobsService struct {
	mock.Mock
}

func (m *MockJobsService) Create(ctx context.Context, req *models.GenerationRequest) (*models.JobResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobsService) Get(ctx context.Context, jobID string) (*models.JobResponse, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobsService) List(ctx context.Context, opts *ListJobsOptions) (*models.JobListResponse, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobListResponse), args.Error(1)
}

func (m *MockJobsService) CreateAndWait(ctx context.Context, req *models.GenerationRequest, opts *WaitOptions) (*models.JobResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

func (m *MockJobsService) WaitForCompletion(ctx context.Context, jobID string, opts *WaitOptions) (*models.JobResponse, error) {
	args := m.Called(ctx, jobID, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobResponse), args.Error(1)
}

// MockStorageService implémente StorageServiceInterface pour les tests
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) UploadSources(ctx context.Context, jobID string, files []FileUpload) (*models.FileUploadResponse, error) {
	args := m.Called(ctx, jobID, files)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FileUploadResponse), args.Error(1)
}

func (m *MockStorageService) UploadSourcesStream(ctx context.Context, jobID string, uploads []StreamUpload) (*models.FileUploadResponse, error) {
	args := m.Called(ctx, jobID, uploads)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FileUploadResponse), args.Error(1)
}

func (m *MockStorageService) UploadSourceFiles(ctx context.Context, jobID string, filePaths []string) (*models.FileUploadResponse, error) {
	args := m.Called(ctx, jobID, filePaths)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FileUploadResponse), args.Error(1)
}

func (m *MockStorageService) ListSources(ctx context.Context, jobID string) (*models.FileListResponse, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FileListResponse), args.Error(1)
}

func (m *MockStorageService) DownloadSource(ctx context.Context, jobID, filename string) (io.ReadCloser, error) {
	args := m.Called(ctx, jobID, filename)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorageService) ListResults(ctx context.Context, courseID string) (*models.FileListResponse, error) {
	args := m.Called(ctx, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FileListResponse), args.Error(1)
}

func (m *MockStorageService) DownloadResult(ctx context.Context, courseID, filename string) (io.ReadCloser, error) {
	args := m.Called(ctx, courseID, filename)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorageService) GetLogs(ctx context.Context, jobID string) (string, error) {
	args := m.Called(ctx, jobID)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) GetStorageInfo(ctx context.Context) (*models.StorageInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StorageInfo), args.Error(1)
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "valid URL without options",
			baseURL: "http://localhost:8081",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "valid URL with timeout option",
			baseURL: "http://localhost:8081",
			opts:    []Option{WithTimeout(30 * time.Second)},
			wantErr: false,
		},
		{
			name:    "valid URL with auth option",
			baseURL: "http://localhost:8081",
			opts:    []Option{WithAuth("test-token")},
			wantErr: false,
		},
		{
			name:    "valid URL with multiple options",
			baseURL: "http://localhost:8081",
			opts: []Option{
				WithTimeout(60 * time.Second),
				WithAuth("test-token"),
				WithLogger(&simpleLogger{}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, tt.opts...)

			// Basic assertions
			require.NotNil(t, client)
			assert.Equal(t, tt.baseURL, client.baseURL)
			assert.NotNil(t, client.httpClient)
			assert.NotNil(t, client.logger)

			// Service initialization - Les interfaces sont maintenant assignées
			assert.NotNil(t, client.Jobs)
			assert.NotNil(t, client.Storage)
			assert.NotNil(t, client.Worker)
			assert.NotNil(t, client.Themes)
			assert.NotNil(t, client.Health)
			assert.NotNil(t, client.Archive)

			// Vérifier que ce sont bien les bonnes implémentations
			_, ok := client.Jobs.(*JobsService)
			assert.True(t, ok, "Jobs should be *JobsService implementation")

			_, ok = client.Storage.(*StorageService)
			assert.True(t, ok, "Storage should be *StorageService implementation")
		})
	}
}

func TestClient_WithMockedServices(t *testing.T) {
	t.Run("test avec Jobs service mocké individuellement", func(t *testing.T) {
		// Créer un client avec des services réels
		client := NewClient("http://localhost:8081")

		// Remplacer uniquement le service Jobs par un mock
		mockJobs := &MockJobsService{}
		client.Jobs = mockJobs

		// Les autres services restent réels
		assert.IsType(t, &StorageService{}, client.Storage)
		assert.IsType(t, &WorkerService{}, client.Worker)

		// Configurer le mock
		expectedJob := &models.JobResponse{
			ID:     uuid.New(),
			Status: models.StatusCompleted,
		}

		mockJobs.On("Create", mock.Anything, mock.Anything).Return(expectedJob, nil)

		// Test
		ctx := context.Background()
		req := &models.GenerationRequest{
			JobID:    uuid.New(),
			CourseID: uuid.New(),
		}

		job, err := client.Jobs.Create(ctx, req)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, expectedJob.ID, job.ID)
		assert.Equal(t, expectedJob.Status, job.Status)

		// Vérifier que le mock a été appelé
		mockJobs.AssertExpectations(t)
	})

	t.Run("test de workflow avec services partiellement mockés", func(t *testing.T) {
		// Scénario : Mocker Jobs (succès) et Storage (échec) pour tester la gestion d'erreur
		client := NewClient("http://localhost:8081")

		mockJobs := &MockJobsService{}
		mockStorage := &MockStorageService{}

		client.Jobs = mockJobs
		client.Storage = mockStorage

		// Jobs réussit
		job := &models.JobResponse{
			ID:     uuid.New(),
			Status: models.StatusPending,
		}
		mockJobs.On("Create", mock.Anything, mock.Anything).Return(job, nil)

		// Storage échoue
		storageError := errors.New("storage quota exceeded")
		mockStorage.On("UploadSources", mock.Anything, mock.Anything, mock.Anything).Return(nil, storageError)

		// Test du workflow
		ctx := context.Background()
		req := &models.GenerationRequest{
			JobID:    job.ID,
			CourseID: uuid.New(),
		}

		// Étape 1 : Créer le job (doit réussir)
		createdJob, err := client.Jobs.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, job.ID, createdJob.ID)

		// Étape 2 : Upload des sources (doit échouer)
		files := []FileUpload{{Name: "test.md", Content: []byte("content")}}
		_, err = client.Storage.UploadSources(ctx, job.ID.String(), files)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "storage quota exceeded")

		// Vérifier les appels
		mockJobs.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})
}

func TestClient_InterfaceCompliance(t *testing.T) {
	t.Run("vérifier que les implémentations respectent les interfaces", func(t *testing.T) {
		client := NewClient("http://localhost:8081")

		// Ces assertions passent à la compilation grâce aux interfaces
		var _ JobsServiceInterface = client.Jobs
		var _ StorageServiceInterface = client.Storage
		var _ WorkerServiceInterface = client.Worker
		var _ ThemesServiceInterface = client.Themes
		var _ HealthServiceInterface = client.Health
		var _ ArchiveServiceInterface = client.Archive
	})
}

// Test d'injection de dépendance personnalisée
func TestClient_CustomServiceInjection(t *testing.T) {
	t.Run("injection d'une implémentation personnalisée", func(t *testing.T) {
		client := NewClient("http://localhost:8081")

		// Créer une implémentation personnalisée (exemple : avec cache)
		cachedJobs := &CachedJobsService{
			underlying: client.Jobs,
			cache:      make(map[string]*models.JobResponse),
		}

		// Injecter l'implémentation personnalisée
		client.Jobs = cachedJobs

		// Test que l'interface fonctionne toujours
		ctx := context.Background()
		_, err := client.Jobs.Get(ctx, "test-job-id")

		// L'erreur est attendue car c'est un test, l'important est que ça compile
		assert.Error(t, err) // Erreur réseau attendue
	})
}

// Exemple d'implémentation avec cache pour démontrer l'extensibilité
type CachedJobsService struct {
	underlying JobsServiceInterface
	cache      map[string]*models.JobResponse
}

func (c *CachedJobsService) Create(ctx context.Context, req *models.GenerationRequest) (*models.JobResponse, error) {
	return c.underlying.Create(ctx, req)
}

func (c *CachedJobsService) Get(ctx context.Context, jobID string) (*models.JobResponse, error) {
	// Vérifier le cache d'abord
	if cached, exists := c.cache[jobID]; exists {
		return cached, nil
	}

	// Sinon, déléguer à l'implémentation sous-jacente
	job, err := c.underlying.Get(ctx, jobID)
	if err == nil {
		c.cache[jobID] = job
	}
	return job, err
}

func (c *CachedJobsService) List(ctx context.Context, opts *ListJobsOptions) (*models.JobListResponse, error) {
	return c.underlying.List(ctx, opts)
}

func (c *CachedJobsService) CreateAndWait(ctx context.Context, req *models.GenerationRequest, opts *WaitOptions) (*models.JobResponse, error) {
	return c.underlying.CreateAndWait(ctx, req, opts)
}

func (c *CachedJobsService) WaitForCompletion(ctx context.Context, jobID string, opts *WaitOptions) (*models.JobResponse, error) {
	return c.underlying.WaitForCompletion(ctx, jobID, opts)
}

// Tests de compatibilité backward - Les anciens tests continuent de fonctionner
func TestClientBackwardCompatibility(t *testing.T) {
	server := NewTestServer()
	defer server.Close()

	client := server.TestClient()
	ctx := context.Background()

	t.Run("GET request", func(t *testing.T) {
		server.On("GET", "/api/v1/test", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			RespondJSON(w, http.StatusOK, map[string]string{"method": "GET"})
		})

		resp, err := client.get(ctx, "/test")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
