package ocfworker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models" // Réutilisation des types
)

type JobsService struct {
	client *Client
}

// WaitOptions options pour le polling automatique
type WaitOptions struct {
	Interval time.Duration // Intervalle entre les checks
	Timeout  time.Duration // Timeout total
}

// Create crée un nouveau job
func (s *JobsService) Create(ctx context.Context, req *models.GenerationRequest) (*models.JobResponse, error) {
	s.client.logger.Info("Creating job", "job_id", req.JobID, "course_id", req.CourseID)

	resp, err := s.client.post(ctx, "/generate", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var job models.JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &job, nil
}

// Get récupère le statut d'un job
func (s *JobsService) Get(ctx context.Context, jobID string) (*models.JobResponse, error) {
	resp, err := s.client.get(ctx, fmt.Sprintf("/jobs/%s", jobID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &JobNotFoundError{JobID: jobID}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var job models.JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &job, nil
}

// List liste les jobs avec pagination et filtres
func (s *JobsService) List(ctx context.Context, opts *ListJobsOptions) (*models.JobListResponse, error) {
	params := url.Values{}

	if opts != nil {
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.CourseID != "" {
			params.Set("course_id", opts.CourseID)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			params.Set("offset", strconv.Itoa(opts.Offset))
		}
	}

	path := "/jobs"
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

	var jobList models.JobListResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &jobList, nil
}

// CreateAndWait crée un job et attend sa completion (polling automatique)
func (s *JobsService) CreateAndWait(ctx context.Context, req *models.GenerationRequest, opts *WaitOptions) (*models.JobResponse, error) {
	if opts == nil {
		opts = &WaitOptions{
			Interval: 5 * time.Second,
			Timeout:  10 * time.Minute,
		}
	}

	// Créer le job
	job, err := s.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Attendre la completion
	return s.WaitForCompletion(ctx, job.ID.String(), opts)
}

// WaitForCompletion attend qu'un job soit terminé (polling automatique)
func (s *JobsService) WaitForCompletion(ctx context.Context, jobID string, opts *WaitOptions) (*models.JobResponse, error) {
	if opts == nil {
		opts = &WaitOptions{
			Interval: 5 * time.Second,
			Timeout:  10 * time.Minute,
		}
	}

	// Créer un contexte avec timeout pour le polling global
	waitCtx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	s.client.logger.Info("Waiting for job completion", "job_id", jobID)

	for {
		select {
		case <-waitCtx.Done():
			return nil, fmt.Errorf("timeout waiting for job completion: %w", waitCtx.Err())
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled while waiting for job completion: %w", ctx.Err())
		case <-ticker.C:
			job, err := s.Get(ctx, jobID)
			if err != nil {
				if isTimeoutError(err) {
					s.client.logger.Debug("Request timeout, retrying", "job_id", jobID)
					continue
				}
				return nil, err
			}

			s.client.logger.Debug("Job status check", "job_id", jobID, "status", job.Status)

			switch job.Status {
			case models.StatusCompleted:
				s.client.logger.Info("Job completed successfully", "job_id", jobID)
				return job, nil
			case models.StatusFailed, models.StatusTimeout:
				s.client.logger.Error("Job failed", "job_id", jobID, "status", job.Status, "error", job.Error)
				return job, fmt.Errorf("job failed with status %s: %s", job.Status, job.Error)
			}
		}
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "timeout")
}

type ListJobsOptions struct {
	Status   string
	CourseID string
	Limit    int
	Offset   int
}
