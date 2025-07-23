package ocfworker

import (
	"encoding/json"
	"fmt"
	"io"
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
		}
	}

	// D'abord essayer de décoder comme APIError complet
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	// Ensuite essayer de décoder comme simple message d'erreur
	var simpleErr struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &simpleErr); err == nil {
		message := simpleErr.Error
		if message == "" {
			message = simpleErr.Message
		}
		if message != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    message,
			}
		}
	}

	// Si rien ne fonctionne, utiliser le status HTTP
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    resp.Status,
	}
}
