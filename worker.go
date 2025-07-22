package ocfworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
)

type WorkerService struct {
	client *Client
}

// Health vérifie la santé du système de workers
func (s *WorkerService) Health(ctx context.Context) (*models.WorkerHealthResponse, error) {
	resp, err := s.client.get(ctx, "/worker/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Lire le body une seule fois
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var health models.WorkerHealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Le code de retour peut être 200 ou 503 selon l'état
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		// Créer un nouveau reader pour parseAPIError
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return &health, parseAPIError(resp)
	}

	return &health, nil
}

// Stats retourne les statistiques du pool de workers
func (s *WorkerService) Stats(ctx context.Context) (*models.WorkerStatsResponse, error) {
	resp, err := s.client.get(ctx, "/worker/stats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var stats models.WorkerStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// ListWorkspaces liste les workspaces actifs
func (s *WorkerService) ListWorkspaces(ctx context.Context, opts *ListWorkspacesOptions) (*models.WorkspaceListResponse, error) {
	params := url.Values{}

	if opts != nil {
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			params.Set("offset", strconv.Itoa(opts.Offset))
		}
	}

	path := "/worker/workspaces"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := s.client.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var workspaces models.WorkspaceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &workspaces, nil
}

// GetWorkspace retourne les informations d'un workspace spécifique
func (s *WorkerService) GetWorkspace(ctx context.Context, jobID string) (*models.WorkspaceInfoResponse, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/worker/workspaces/%s", jobID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workspace not found for job %s", jobID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var workspace models.WorkspaceInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&workspace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &workspace, nil
}

// DeleteWorkspace supprime un workspace
func (s *WorkerService) DeleteWorkspace(ctx context.Context, jobID string) (*models.WorkspaceCleanupResponse, error) {
	s.client.logger.Warn("Deleting workspace", "job_id", jobID)

	req, err := http.NewRequestWithContext(ctx, "DELETE",
		s.client.baseURL+fmt.Sprintf("/api/v1/worker/workspaces/%s", jobID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workspace not found for job %s", jobID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var cleanup models.WorkspaceCleanupResponse
	if err := json.NewDecoder(resp.Body).Decode(&cleanup); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &cleanup, nil
}

// CleanupOldWorkspaces nettoie les anciens workspaces
func (s *WorkerService) CleanupOldWorkspaces(ctx context.Context, maxAgeHours int) (*models.WorkspaceCleanupBatchResponse, error) {
	params := url.Values{}
	if maxAgeHours > 0 {
		params.Set("max_age_hours", strconv.Itoa(maxAgeHours))
	}

	path := "/worker/workspaces/cleanup"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	s.client.logger.Info("Cleaning up old workspaces", "max_age_hours", maxAgeHours)

	resp, err := s.client.post(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var cleanup models.WorkspaceCleanupBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&cleanup); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &cleanup, nil
}

type ListWorkspacesOptions struct {
	Status string // "active", "idle", "completed"
	Limit  int
	Offset int
}
