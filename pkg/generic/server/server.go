// Package server provides a minimal implementation of a generic storage server
package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:generate go tool oapi-codegen -generate gin -package server -o server.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate spec -package server -o spec.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate types -package server -o types.gen.go ../../../api/storage.yaml

// File holds the content and metadata of a stored file.
type File struct {
	Key          string
	Content      []byte
	ETag         string
	LastModified time.Time
}

// Server is a minimal in-memory implementation of ServerInterface.
type Server struct {
	files map[string]File
}

// New returns a Server pre-populated with the given files.
func New(files []File) *Server {
	m := make(map[string]File, len(files))
	for _, f := range files {
		m[f.Key] = f
	}
	return &Server{files: m}
}

// GetFiles implements GET /files.
func (s *Server) GetFiles(c *gin.Context, params GetFilesParams) {
	var matches []FileMetadata
	for _, f := range s.files {
		if params.Prefix != nil && !strings.HasPrefix(f.Key, *params.Prefix) {
			continue
		}
		matches = append(matches, FileMetadata{
			Key:          f.Key,
			Size:         len(f.Content),
			LastModified: f.LastModified,
			Etag:         f.ETag,
		})
	}
	if matches == nil {
		matches = []FileMetadata{}
	}
	count := len(matches)
	c.JSON(http.StatusOK, ListFilesResponse{
		Files:     matches,
		FileCount: &count,
		Prefix:    params.Prefix,
	})
}

// GetFilesKey implements GET /files/{key}.
func (s *Server) GetFilesKey(c *gin.Context, key KeyParam, params GetFilesKeyParams) {
	f, ok := s.files[key]
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{Message: "file not found"})
		return
	}
	if f.ETag != params.IfMatch {
		c.JSON(http.StatusPreconditionFailed, ErrorResponse{Message: "ETag mismatch"})
		return
	}
	c.Header("ETag", f.ETag)
	c.Header("Last-Modified", f.LastModified.UTC().Format(time.RFC3339))
	c.DataFromReader(
		http.StatusOK,
		int64(len(f.Content)),
		"application/octet-stream",
		io.NopCloser(bytes.NewReader(f.Content)),
		nil,
	)
}
