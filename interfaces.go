package ocfworker

import (
	"context"
	"io"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
)

// JobsServiceInterface définit l'interface pour la gestion des jobs
type JobsServiceInterface interface {
	// Create crée un nouveau job
	Create(ctx context.Context, req *models.GenerationRequest) (*models.JobResponse, error)

	// Get récupère le statut d'un job
	Get(ctx context.Context, jobID string) (*models.JobResponse, error)

	// List liste les jobs avec pagination et filtres
	List(ctx context.Context, opts *ListJobsOptions) (*models.JobListResponse, error)

	// CreateAndWait crée un job et attend sa completion (polling automatique)
	CreateAndWait(ctx context.Context, req *models.GenerationRequest, opts *WaitOptions) (*models.JobResponse, error)

	// WaitForCompletion attend qu'un job soit terminé (polling automatique)
	WaitForCompletion(ctx context.Context, jobID string, opts *WaitOptions) (*models.JobResponse, error)
}

// StorageServiceInterface définit l'interface pour la gestion du stockage
type StorageServiceInterface interface {
	// UploadSources upload des fichiers sources pour un job (mode mémoire)
	UploadSources(ctx context.Context, jobID string, files []FileUpload) (*models.FileUploadResponse, error)

	// UploadSourcesStream upload des fichiers sources en streaming
	UploadSourcesStream(ctx context.Context, jobID string, uploads []StreamUpload) (*models.FileUploadResponse, error)

	// UploadSourceFiles helper pour uploader des fichiers depuis le système de fichiers
	UploadSourceFiles(ctx context.Context, jobID string, filePaths []string) (*models.FileUploadResponse, error)

	// ListSources liste les fichiers sources d'un job
	ListSources(ctx context.Context, jobID string) (*models.FileListResponse, error)

	// DownloadSource télécharge un fichier source spécifique
	DownloadSource(ctx context.Context, jobID, filename string) (io.ReadCloser, error)

	// ListResults liste les fichiers de résultats d'un cours
	ListResults(ctx context.Context, courseID string) (*models.FileListResponse, error)

	// DownloadResult télécharge un fichier de résultat spécifique
	DownloadResult(ctx context.Context, courseID, filename string) (io.ReadCloser, error)

	// GetLogs récupère les logs d'un job
	GetLogs(ctx context.Context, jobID string) (string, error)

	// GetStorageInfo récupère les informations sur le stockage
	GetStorageInfo(ctx context.Context) (*models.StorageInfo, error)
}

// WorkerServiceInterface définit l'interface pour la gestion des workers
type WorkerServiceInterface interface {
	// Health vérifie la santé du système de workers
	Health(ctx context.Context) (*models.WorkerHealthResponse, error)

	// Stats retourne les statistiques du pool de workers
	Stats(ctx context.Context) (*models.WorkerStatsResponse, error)

	// ListWorkspaces liste les workspaces actifs
	ListWorkspaces(ctx context.Context, opts *ListWorkspacesOptions) (*models.WorkspaceListResponse, error)

	// GetWorkspace retourne les informations d'un workspace spécifique
	GetWorkspace(ctx context.Context, jobID string) (*models.WorkspaceInfoResponse, error)

	// DeleteWorkspace supprime un workspace
	DeleteWorkspace(ctx context.Context, jobID string) (*models.WorkspaceCleanupResponse, error)

	// CleanupOldWorkspaces nettoie les anciens workspaces
	CleanupOldWorkspaces(ctx context.Context, maxAgeHours int) (*models.WorkspaceCleanupBatchResponse, error)
}

// ThemesServiceInterface définit l'interface pour la gestion des thèmes
type ThemesServiceInterface interface {
	// ListAvailable liste tous les thèmes Slidev disponibles
	ListAvailable(ctx context.Context) (*models.ThemeListResponse, error)

	// Install installe un thème Slidev
	Install(ctx context.Context, themeName string) (*models.ThemeInstallResponse, error)

	// DetectForJob détecte les thèmes requis pour un job
	DetectForJob(ctx context.Context, jobID string) (*models.ThemeDetectionResponse, error)

	// AutoInstallForJob détecte et installe automatiquement les thèmes manquants pour un job
	AutoInstallForJob(ctx context.Context, jobID string) (*models.ThemeAutoInstallResponse, error)
}

// HealthServiceInterface définit l'interface pour les health checks
type HealthServiceInterface interface {
	// Check effectue un health check général du service
	Check(ctx context.Context) (*models.HealthResponse, error)
}

// ArchiveServiceInterface définit l'interface pour la gestion des archives
type ArchiveServiceInterface interface {
	// DownloadArchive télécharge l'archive d'un cours
	DownloadArchive(ctx context.Context, courseID string, opts *DownloadArchiveOptions) (io.ReadCloser, error)
}

// Vérification à la compilation que les implémentations respectent les interfaces
var (
	_ JobsServiceInterface    = (*JobsService)(nil)
	_ StorageServiceInterface = (*StorageService)(nil)
	_ WorkerServiceInterface  = (*WorkerService)(nil)
	_ ThemesServiceInterface  = (*ThemesService)(nil)
	_ HealthServiceInterface  = (*HealthService)(nil)
	_ ArchiveServiceInterface = (*ArchiveService)(nil)
)
