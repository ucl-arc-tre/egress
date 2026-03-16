package generic

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ucl-arc-tre/egress/internal/types"
)

type Storage struct {
	getter apiClientGetter
}

// Uses a single reusable http.Client for all requests
// mTLS auth is handled at the transport layer
func New(tlsCertDir string) (*Storage, error) {
	transport, err := newMTLSTransport(tlsCertDir)
	if err != nil {
		return nil, fmt.Errorf("[generic] failed to configure TLS transport: %w", err)
	}
	return &Storage{
		getter: &httpAPIClientGetter{
			http: &http.Client{
				Transport: transport,
			},
		},
	}, nil
}

func (s *Storage) List(ctx context.Context, location types.LocationURI) ([]types.FileMetadata, error) {
	client, err := s.getter.Get(location)
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
		m := extractResponseMessageOrDefault(resp.JSON400, "bad request")
		return nil, types.NewErrInvalidObjectF("%s: %s", errmsg, m)
	default:
		m := extractResponseMessageOrDefault(resp.JSON500, "unexpected error")
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
	client, err := s.getter.Get(location)
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
		m := extractResponseMessageOrDefault(resp.JSON404, "file not found")
		return nil, types.NewErrNotFoundF("%s: %s", errmsg, m)
	case http.StatusBadRequest:
		m := extractResponseMessageOrDefault(resp.JSON400, "bad request")
		return nil, types.NewErrInvalidObjectF("%s: %s", errmsg, m)
	case http.StatusPreconditionFailed:
		return nil, types.NewErrNotFoundF("%s: ETag mismatch for fileId [%v]", errmsg, fileId)
	default:
		m := extractResponseMessageOrDefault(resp.JSON500, "unexpected error")
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

func extractResponseMessageOrDefault(body *ErrorResponse, fallback string) string {
	if body != nil {
		return body.Message
	}
	return fallback
}

func stripQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, "")
}

// Builds an http.Transport configured for mTLS using the
// CA cert, client cert and client key found in given directory
// Expected files:
//
//	ca.crt  – CA certificate
//	tls.crt – client certificate for TLS handshake
//	tls.key – private key for client certificate
func newMTLSTransport(dir string) (http.RoundTripper, error) {
	caCert, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}
	cert, err := tls.LoadX509KeyPair(
		filepath.Join(dir, "tls.crt"),
		filepath.Join(dir, "tls.key"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert/key: %w", err)
	}
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		},
	}, nil
}
