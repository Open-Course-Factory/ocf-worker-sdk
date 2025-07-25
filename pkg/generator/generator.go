package generator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	ocfworker "ocf-worker-sdk"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
)

// Generator gère la génération de présentations Slidev
type Generator struct {
	client     *ocfworker.Client
	downloader *GitHubDownloader
	config     *Config
	logger     *log.Logger
}

// New crée un nouveau générateur
func New(config *Config) *Generator {
	// Options du client
	clientOpts := []ocfworker.Option{
		ocfworker.WithTimeout(config.Timeout),
	}

	if config.AuthToken != "" {
		clientOpts = append(clientOpts, ocfworker.WithAuth(config.AuthToken))
	}

	if config.Verbose {
		clientOpts = append(clientOpts, ocfworker.WithLogger(NewVerboseLogger()))
	}

	client := ocfworker.NewClient(config.APIBaseURL, clientOpts...)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	if !config.Verbose {
		logger.SetOutput(os.Stderr)
	}

	return &Generator{
		client:     client,
		downloader: NewGitHubDownloader(),
		config:     config,
		logger:     logger,
	}
}

// Generate génère une présentation Slidev
func (g *Generator) Generate(ctx context.Context) (*Result, error) {
	g.logger.Printf("🚀 Début de la génération depuis: %s", g.config.GitHubURL)

	// 1. Vérifier la santé du service
	if err := g.checkHealth(ctx); err != nil {
		return nil, fmt.Errorf("service indisponible: %w", err)
	}

	// 2. Générer les IDs
	jobID := uuid.New()
	courseID := uuid.New()

	g.logger.Printf("🆔 Job ID: %s", jobID)
	g.logger.Printf("🆔 Course ID: %s", courseID)

	// 3. Télécharger le dépôt
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("ocf-slidev-%s", jobID.String()[:8]))
	defer os.RemoveAll(tempDir)

	files, err := g.downloader.DownloadRepo(ctx, g.config.GitHubURL, tempDir, g.config.Subfolder)
	if err != nil {
		return nil, fmt.Errorf("téléchargement échoué: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("aucun fichier Slidev trouvé")
	}

	// 4. Préparer les uploads
	uploads, err := g.prepareUploads(files)
	if err != nil {
		return nil, fmt.Errorf("préparation uploads échouée: %w", err)
	}

	// 5. Upload des sources
	if err := g.uploadSources(ctx, jobID.String(), uploads); err != nil {
		return nil, fmt.Errorf("upload échoué: %w", err)
	}

	// 6. Installation automatique des thèmes
	if err := g.installThemes(ctx, jobID.String()); err != nil {
		g.logger.Printf("⚠️ Avertissement thèmes: %v", err)
	}

	// 7. Génération
	_, err = g.createAndWaitJob(ctx, jobID, courseID)
	if err != nil {
		logs, errLogs := g.client.Storage.GetLogs(ctx, jobID.String())
		if errLogs != nil {
			return nil, fmt.Errorf("génération échouée: %w", err)
		}
		g.logger.Printf("log slidev: %s", logs)
		return nil, fmt.Errorf("génération échouée: %w", err)
	}

	// 8. Téléchargement des résultats
	result, err := g.downloadResults(ctx, courseID.String())
	if err != nil {
		return nil, fmt.Errorf("téléchargement résultats échoué: %w", err)
	}

	result.JobID = jobID.String()
	result.CourseID = courseID.String()

	g.logger.Printf("🎉 Génération terminée avec succès!")
	return result, nil
}

func (g *Generator) checkHealth(ctx context.Context) error {
	g.logger.Printf("🏥 Vérification de la santé du service...")

	health, err := g.client.Health.Check(ctx)
	if err != nil {
		return err
	}

	g.logger.Printf("✅ Service: %s (%s)", health.Service, health.Status)
	return nil
}

func (g *Generator) prepareUploads(files []string) ([]ocfworker.FileUpload, error) {
	g.logger.Printf("📦 Préparation de %d fichiers...", len(files))

	var uploads []ocfworker.FileUpload
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("lecture fichier %s: %w", file, err)
		}

		path := strings.Split(file, string(os.PathSeparator))
		realPath := path[3:]

		uploads = append(uploads, ocfworker.FileUpload{
			Name:        filepath.Join(realPath...),
			Content:     content,
			ContentType: detectContentType(file),
		})
	}

	return uploads, nil
}

func (g *Generator) uploadSources(ctx context.Context, jobID string, uploads []ocfworker.FileUpload) error {
	g.logger.Printf("📤 Upload de %d fichiers...", len(uploads))

	result, err := g.client.Storage.UploadSources(ctx, jobID, uploads)
	if err != nil {
		return err
	}

	g.logger.Printf("✅ %d fichiers uploadés", result.Count)
	return nil
}

func (g *Generator) installThemes(ctx context.Context, jobID string) error {
	g.logger.Printf("🎨 Installation automatique des thèmes...")

	result, err := g.client.Themes.AutoInstallForJob(ctx, jobID)
	if err != nil {
		return err
	}

	g.logger.Printf("✅ %d thèmes installés", result.Successful)

	if g.config.Verbose {
		for _, theme := range result.Results {
			status := "❌"
			if theme.Success {
				status = "✅"
			}
			g.logger.Printf("  %s %s", status, theme.Theme)
		}
	}

	return nil
}

func (g *Generator) createAndWaitJob(ctx context.Context, jobID, courseID uuid.UUID) (*models.JobResponse, error) {
	g.logger.Printf("🚀 Création du job de génération...")

	req := &models.GenerationRequest{
		JobID:      jobID,
		CourseID:   courseID,
		SourcePath: "slides.md", // Fichier principal par défaut
		Metadata: map[string]interface{}{
			"generator": "ocf-cli",
			"source":    "github",
			"url":       g.config.GitHubURL,
		},
	}

	waitOpts := &ocfworker.WaitOptions{
		Interval: g.config.WaitInterval,
		Timeout:  g.config.WaitTimeout,
	}

	g.logger.Printf("⏳ Génération en cours...")
	job, err := g.client.Jobs.CreateAndWait(ctx, req, waitOpts)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (g *Generator) downloadResults(ctx context.Context, courseID string) (*Result, error) {
	g.logger.Printf("📥 Téléchargement des résultats...")

	// Créer le répertoire de sortie
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("création répertoire sortie: %w", err)
	}

	// Télécharger l'archive
	archiveOpts := &ocfworker.DownloadArchiveOptions{
		Format:   "zip",
		Compress: &[]bool{true}[0],
	}

	reader, err := g.client.Archive.DownloadArchive(ctx, courseID, archiveOpts)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Sauvegarder l'archive
	archivePath := filepath.Join(g.config.OutputDir, "presentation.zip")
	if err := saveReaderToFile(reader, archivePath); err != nil {
		return nil, fmt.Errorf("sauvegarde archive: %w", err)
	}

	g.logger.Printf("✅ Archive: %s", archivePath)

	// Extraire l'archive
	extractDir := filepath.Join(g.config.OutputDir, "presentation")
	files, err := extractZipFile(archivePath, extractDir)
	if err != nil {
		return nil, fmt.Errorf("extraction archive: %w", err)
	}

	g.logger.Printf("✅ Présentation extraite: %s", extractDir)

	// Chercher l'index.html principal
	indexPath := filepath.Join(extractDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		// Chercher dans les sous-dossiers
		matches, _ := filepath.Glob(filepath.Join(extractDir, "**/index.html"))
		if len(matches) > 0 {
			indexPath = matches[0]
		}
	}

	return &Result{
		OutputDir:   g.config.OutputDir,
		IndexPath:   indexPath,
		ArchivePath: archivePath,
		Files:       files,
	}, nil
}
