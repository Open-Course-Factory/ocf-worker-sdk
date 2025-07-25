package generator

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitHubDownloader gère le téléchargement des dépôts GitHub
type GitHubDownloader struct {
	httpClient *http.Client
}

// NewGitHubDownloader crée un nouveau téléchargeur
func NewGitHubDownloader() *GitHubDownloader {
	return &GitHubDownloader{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DownloadRepo télécharge et extrait un dépôt GitHub
func (d *GitHubDownloader) DownloadRepo(ctx context.Context, repoURL, outputDir, subfolder string) ([]string, error) {
	if deadline, ok := ctx.Deadline(); ok {
		fmt.Printf("Context deadline: %v (time remaining: %v)\n", deadline, time.Until(deadline))
	}

	// Check your HTTP client timeout
	fmt.Printf("HTTP client timeout: %v\n", d.httpClient.Timeout)

	// Parser l'URL GitHub
	owner, repo, branch, subPath := parseGitHubURL(repoURL)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("URL GitHub invalide: %s", repoURL)
	}

	// Ajouter le subfolder au subPath
	if subfolder != "" {
		if subPath != "" {
			subPath = subPath + "/" + subfolder
		} else {
			subPath = subfolder
		}
	}

	// Construire l'URL ZIP
	zipURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, branch)

	// Télécharger
	req, err := http.NewRequestWithContext(ctx, "GET", zipURL, nil)
	if err != nil {
		return nil, err
	}

	originalTimeout := d.httpClient.Timeout

	// Remove timeout for this request
	d.httpClient.Timeout = 0 // 0 means no timeout

	// Restore it after
	defer func() {
		d.httpClient.Timeout = originalTimeout
	}()

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur HTTP %d", resp.StatusCode)
	}

	// Fichier temporaire
	tempFile, err := os.CreateTemp("", "repo-*.zip")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copier le contenu
	progressReader := &ProgressReader{Reader: resp.Body, lastLog: time.Now()}
	if _, err := io.Copy(tempFile, progressReader); err != nil {
		fmt.Printf("Copy failed after downloading %d bytes: %v\n", progressReader.bytesRead, err)
		return nil, err
	}

	// Extraire
	return d.extractRepo(tempFile.Name(), outputDir, fmt.Sprintf("%s-%s", repo, branch), subPath)
}

func parseGitHubURL(url string) (owner, repo, branch, subPath string) {
	// Supprimer le préfixe
	url = strings.TrimPrefix(url, "https://github.com/")
	parts := strings.Split(url, "/")

	if len(parts) < 2 {
		return
	}

	owner = parts[0]
	repo = parts[1]
	branch = "main" // défaut

	// Gérer tree/branch/path
	if len(parts) > 2 && parts[2] == "tree" {
		if len(parts) > 3 {
			branch = parts[3]
		}
		if len(parts) > 4 {
			subPath = strings.Join(parts[4:], "/")
		}
	}

	return
}

func (d *GitHubDownloader) extractRepo(zipFile, outputDir, repoPrefix, subPath string) ([]string, error) {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	var extractedFiles []string

	for _, file := range reader.File {
		// Supprimer le préfixe du dépôt
		relativePath := strings.TrimPrefix(file.Name, repoPrefix+"/")
		if relativePath == file.Name {
			continue
		}

		// Filtrer par subPath
		if subPath != "" {
			if !strings.HasPrefix(relativePath, subPath) {
				continue
			}
			relativePath = strings.TrimPrefix(relativePath, subPath)
			relativePath = strings.TrimPrefix(relativePath, "/")
		}

		if relativePath == "" || file.FileInfo().IsDir() {
			continue
		}

		// Vérifier si c'est un fichier supporté
		if !isSlidevFile(relativePath) {
			continue
		}

		outputPath := filepath.Join(outputDir, relativePath)

		// Créer les dossiers parents
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return nil, err
		}

		// Extraire le fichier
		if err := extractZipEntry(file, outputPath); err != nil {
			return nil, err
		}

		extractedFiles = append(extractedFiles, outputPath)
	}

	return extractedFiles, nil
}

func extractZipEntry(file *zip.File, outputPath string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}
