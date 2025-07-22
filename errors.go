package ocfworker

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Erreurs typées basées sur l'API
type APIError struct {
	StatusCode int
	Message    string
	Path       string
	RequestID  string
	Timestamp  string
	Details    []ValidationError `json:"validation_errors,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

// JobNotFoundError erreur spécifique pour les jobs non trouvés
type JobNotFoundError struct {
	JobID string
}

func (e *JobNotFoundError) Error() string {
	return fmt.Sprintf("job not found: %s", e.JobID)
}

// parseAPIError extrait l'erreur depuis la réponse HTTP
func parseAPIError(resp *http.Response) error {
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
		}
	}

	apiErr.StatusCode = resp.StatusCode
	return &apiErr
}
