package generic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ucl-arc-tre/egress/internal/types"
)

type Storage struct {
	client clientGetter
}

// Uses a single reusable http.Client for all requests
// mTLS auth is handled at the transport layer
func New() *Storage {
	return &Storage{
		client: &httpClientGetter{
			httpClient: &http.Client{
				Transport: http.DefaultTransport,
			},
		},
	}
}

func (s *Storage) List(ctx context.Context, location types.LocationURI) ([]types.FileMetadata, error) {
	client, err := s.client.Get(location)
	if err != nil {
		return nil, err
	}

	errmsg := "[generic] failed to list files"
	resp, err := client.GetFilesWithResponse(ctx, &GetFilesParams{})
	if err != nil {
		return nil, types.NewErrServerF("%s: %w", errmsg, err)
	}
	switch resp.StatusCode() {
	case http.StatusOK: // Handled after switch
	case http.StatusBadRequest:
		m := responseMessageOrDefault("bad request", resp.JSON400)
		return nil, types.NewErrInvalidObjectF("%s: %s", errmsg, m)
	default:
		m := responseMessageOrDefault("unexpected error", resp.JSON500)
		return nil, types.NewErrServerF("%s: %s (status %d)", errmsg, m, resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, types.NewErrServerF("[generic] list files returned empty response")
	}
	result := make([]types.FileMetadata, 0, len(resp.JSON200.Files))
	for _, f := range resp.JSON200.Files {
		result = append(result, types.FileMetadata{
			Name:           f.Key,
			Id:             types.FileId(stripQuotes(f.Etag)),
			Size:           f.Size,
			LastModifiedAt: f.LastModified,
		})
	}
	return result, nil
}

func (s *Storage) Get(ctx context.Context, location types.LocationURI, fileId types.FileId) (*types.File, error) {
	client, err := s.client.Get(location)
	if err != nil {
		return nil, err
	}
	key, err := s.keyForFileId(ctx, client, fileId)
	if err != nil {
		return nil, err
	}

	errmsg := "[generic] failed to get file"
	resp, err := client.GetFileWithResponse(ctx, &GetFileParams{
		Key:     key,
		IfMatch: fmt.Sprintf(`"%s"`, fileId),
	})
	if err != nil {
		return nil, types.NewErrServerF("%s: %w", errmsg, err)
	}
	switch resp.StatusCode() {
	case http.StatusOK: // Handled after switch
	case http.StatusNotFound:
		m := responseMessageOrDefault("file not found", resp.JSON404)
		return nil, types.NewErrNotFoundF("%s: %s", errmsg, m)
	case http.StatusBadRequest:
		m := responseMessageOrDefault("bad request", resp.JSON400)
		return nil, types.NewErrInvalidObjectF("%s: %s", errmsg, m)
	case http.StatusPreconditionFailed:
		return nil, types.NewErrNotFoundF("%s: ETag mismatch for fileId [%v]", errmsg, fileId)
	default:
		m := responseMessageOrDefault("unexpected error", resp.JSON500)
		return nil, types.NewErrServerF("%s: %s (status %d)", errmsg, m, resp.StatusCode())
	}

	return &types.File{
		Content: io.NopCloser(bytes.NewReader(resp.Body)),
		Size:    max(resp.HTTPResponse.ContentLength, 0),
	}, nil
}

func (s *Storage) keyForFileId(ctx context.Context, client ClientWithResponsesInterface, fileId types.FileId) (string, error) {
	resp, err := client.GetFilesWithResponse(ctx, &GetFilesParams{})
	if err != nil {
		return "", types.NewErrServerF("[generic] failed to list: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return "", types.NewErrServerF("[generic] unexpected list status [%d]", resp.StatusCode())
	}
	for _, f := range resp.JSON200.Files {
		if types.FileId(stripQuotes(f.Etag)) == fileId {
			return f.Key, nil
		}
	}
	return "", types.NewErrNotFoundF("[generic] no file with fileId [%v]", fileId)
}

func responseMessageOrDefault(fallback string, body *ErrorResponse) string {
	if body != nil {
		return body.Message
	}
	return fallback
}

func stripQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, "")
}
