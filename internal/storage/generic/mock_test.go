package generic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ucl-arc-tre/egress/internal/types"
)

// Returns a pre-configured mock client for testing
type mockAPIClientGetter struct {
	mock ClientWithResponsesInterface
}

func (g *mockAPIClientGetter) Get(location types.LocationURI) (ClientWithResponsesInterface, error) {
	return g.mock, nil
}

func newWithMock(mockClient ClientWithResponsesInterface) *Storage {
	return &Storage{
		getter: &mockAPIClientGetter{
			mock: mockClient,
		},
	}
}

type mockFile struct {
	Key            string
	ETag           string
	LastModifiedAt time.Time
	Content        string
}

type mockClient struct {
	Files        []mockFile
	ForceListErr error
	ForceGetErr  error
}

func (c *mockClient) GetFilesWithResponse(
	_ context.Context,
	params *GetFilesParams,
	_ ...RequestEditorFn,
) (*GetFilesResponse, error) {
	if c.ForceListErr != nil {
		return nil, c.ForceListErr
	}
	var matches []FileMetadata
	for _, f := range c.Files {
		if params.Prefix != nil && !strings.HasPrefix(f.Key, *params.Prefix) {
			continue
		}
		matches = append(matches, FileMetadata{
			Key:          f.Key,
			Etag:         f.ETag,
			Size:         int64(len(f.Content)),
			LastModified: f.LastModifiedAt,
		})
	}
	body := ListFilesResponse{
		Files:     matches,
		FileCount: len(matches),
		Prefix:    params.Prefix,
	}
	return &GetFilesResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      &body,
	}, nil
}

func (c *mockClient) GetFileWithResponse(
	_ context.Context,
	params *GetFileParams,
	_ ...RequestEditorFn,
) (*GetFileResponse, error) {
	if c.ForceGetErr != nil {
		return nil, c.ForceGetErr
	}
	for _, f := range c.Files {
		if f.Key != params.Key {
			continue
		}
		// Enforce the If-Match precondition
		if f.ETag != params.IfMatch {
			return &GetFileResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusPreconditionFailed},
				JSON412: &PreconditionFailed{
					Message: fmt.Sprintf("ETag mismatch: have %s, want %s", f.ETag, params.IfMatch),
				},
			}, nil
		}
		content := []byte(f.Content)
		return &GetFileResponse{
			Body: content,
			HTTPResponse: &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: int64(len(content)),
				Body:          io.NopCloser(bytes.NewReader(content)),
			},
		}, nil
	}
	return &GetFileResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
		JSON404:      &NotFound{Message: fmt.Sprintf("file not found: %s", params.Key)},
	}, nil
}
