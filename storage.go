package ocfworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
)

type StorageService struct {
	client *Client
}

// FileUpload représente un fichier à uploader en mémoire
type FileUpload struct {
	Name        string
	Content     []byte
	ContentType string
}

// StreamUpload représente un fichier à uploader en streaming
type StreamUpload struct {
	Name        string
	Reader      io.Reader
	Size        int64
	ContentType string
}

// UploadSources upload des fichiers sources pour un job (mode mémoire)
func (s *StorageService) UploadSources(ctx context.Context, jobID string, files []FileUpload) (*models.FileUploadResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, file := range files {
		part, err := writer.CreateFormFile("files", file.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err := part.Write(file.Content); err != nil {
			return nil, fmt.Errorf("failed to write file content: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		s.client.baseURL+fmt.Sprintf("/api/v1/storage/jobs/%s/sources", jobID), &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var uploadResp models.FileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &uploadResp, nil
}

// UploadSourcesStream upload des fichiers sources en streaming
func (s *StorageService) UploadSourcesStream(ctx context.Context, jobID string, uploads []StreamUpload) (*models.FileUploadResponse, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		for _, upload := range uploads {
			part, err := writer.CreateFormFile("files", upload.Name)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			if _, err := io.Copy(part, upload.Reader); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	req, err := http.NewRequestWithContext(ctx, "POST",
		s.client.baseURL+fmt.Sprintf("/api/v1/storage/jobs/%s/sources", jobID), pr)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var uploadResp models.FileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &uploadResp, nil
}

// UploadSourceFiles helper pour uploader des fichiers depuis le système de fichiers
func (s *StorageService) UploadSourceFiles(ctx context.Context, jobID string, filePaths []string) (*models.FileUploadResponse, error) {
	var uploads []StreamUpload

	for _, path := range filePaths {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
		}

		uploads = append(uploads, StreamUpload{
			Name:        stat.Name(),
			Reader:      file,
			Size:        stat.Size(),
			ContentType: detectContentType(path),
		})
	}

	return s.UploadSourcesStream(ctx, jobID, uploads)
}

// ListSources liste les fichiers sources d'un job
func (s *StorageService) ListSources(ctx context.Context, jobID string) (*models.FileListResponse, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/storage/jobs/%s/sources", jobID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var sources models.FileListResponse
	if err := json.NewDecoder(resp.Body).Decode(&sources); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &sources, nil
}

// DownloadSource télécharge un fichier source spécifique
func (s *StorageService) DownloadSource(ctx context.Context, jobID, filename string) (io.ReadCloser, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/storage/jobs/%s/sources/%s", jobID, filename))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, parseAPIError(resp)
	}

	return resp.Body, nil
}

// ListResults liste les fichiers de résultats d'un cours
func (s *StorageService) ListResults(ctx context.Context, courseID string) (*models.FileListResponse, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/storage/courses/%s/results", courseID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var results models.FileListResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results.Count = len(results.Files)

	return &results, nil
}

// DownloadResult télécharge un fichier de résultat spécifique
func (s *StorageService) DownloadResult(ctx context.Context, courseID, filename string) (io.ReadCloser, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/storage/courses/%s/results/%s", courseID, filename))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, parseAPIError(resp)
	}

	return resp.Body, nil
}

// GetLogs récupère les logs d'un job
func (s *StorageService) GetLogs(ctx context.Context, jobID string) (string, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/storage/jobs/%s/logs", jobID))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", parseAPIError(resp)
	}

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return string(logs), nil
}

// GetStorageInfo récupère les informations sur le stockage
func (s *StorageService) GetStorageInfo(ctx context.Context) (*models.StorageInfo, error) {
	resp, err := s.client.get(ctx, "/storage/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var info models.StorageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// detectContentType détecte le type MIME basé sur l'extension
func detectContentType(filename string) string {
	// Implémentation simple basée sur l'extension
	switch {
	case strings.HasSuffix(filename, ".md"):
		return "text/markdown"
	case strings.HasSuffix(filename, ".css"):
		return "text/css"
	case strings.HasSuffix(filename, ".js"):
		return "application/javascript"
	case strings.HasSuffix(filename, ".json"):
		return "application/json"
	case strings.HasSuffix(filename, ".png"):
		return "image/png"
	case strings.HasSuffix(filename, ".jpg"), strings.HasSuffix(filename, ".jpeg"):
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
