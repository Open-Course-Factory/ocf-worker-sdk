package ocfworker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
)

type HealthService struct {
	client *Client
}

// Check effectue un health check général du service
func (s *HealthService) Check(ctx context.Context) (*models.HealthResponse, error) {
	resp, err := s.client.get(ctx, "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Le service peut retourner 200 ou 503 selon l'état
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return &health, parseAPIError(resp)
	}

	return &health, nil
}
