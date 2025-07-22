package ocfworker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
)

type ThemesService struct {
	client *Client
}

// ListAvailable liste tous les thèmes Slidev disponibles
func (s *ThemesService) ListAvailable(ctx context.Context) (*models.ThemeListResponse, error) {
	resp, err := s.client.get(ctx, "/themes/available")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var themes models.ThemeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&themes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &themes, nil
}

// Install installe un thème Slidev
func (s *ThemesService) Install(ctx context.Context, themeName string) (*models.ThemeInstallResponse, error) {
	req := &models.ThemeInstallRequest{Theme: themeName}

	s.client.logger.Info("Installing theme", "theme", themeName)

	resp, err := s.client.post(ctx, "/themes/install", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result models.ThemeInstallResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DetectForJob détecte les thèmes requis pour un job
func (s *ThemesService) DetectForJob(ctx context.Context, jobID string) (*models.ThemeDetectionResponse, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/themes/jobs/%s/detect", jobID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var detection models.ThemeDetectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&detection); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &detection, nil
}

// AutoInstallForJob détecte et installe automatiquement les thèmes manquants pour un job
func (s *ThemesService) AutoInstallForJob(ctx context.Context, jobID string) (*models.ThemeAutoInstallResponse, error) {
	s.client.logger.Info("Auto-installing themes for job", "job_id", jobID)

	resp, err := s.client.post(ctx, fmt.Sprintf("/themes/jobs/%s/install", jobID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result models.ThemeAutoInstallResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.Successful = len(result.Results)

	return &result, nil
}
