package ocfworker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type ArchiveService struct {
	client *Client
}

// DownloadArchive télécharge l'archive d'un cours
func (s *ArchiveService) DownloadArchive(ctx context.Context, courseID string, opts *DownloadArchiveOptions) (io.ReadCloser, error) {
	params := url.Values{}

	if opts != nil {
		if opts.Format != "" {
			params.Set("format", opts.Format)
		}
		if opts.Compress != nil {
			params.Set("compress", strconv.FormatBool(*opts.Compress))
		}
	}

	path := fmt.Sprintf("/storage/courses/%s/archive", courseID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := s.client.get(ctx, path)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, parseAPIError(resp)
	}

	return resp.Body, nil
}

type DownloadArchiveOptions struct {
	Format   string // "zip", "tar"
	Compress *bool
}
