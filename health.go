package ocfworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	// Lire le body une seule fois
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var health models.HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Le service peut retourner 200 ou 503 selon l'état
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		// Pour les autres codes d'erreur, créer un nouveau reader pour parseAPIError
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return nil, parseAPIError(resp) // Retourner nil, error pour les vraies erreurs
	}

	return &health, nil
}
