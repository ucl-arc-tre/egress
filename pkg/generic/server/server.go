// Package server provides a minimal implementation of a generic storage server
package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:generate go tool oapi-codegen -generate gin -package server -o server.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate spec -package server -o spec.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate types -package server -o types.gen.go ../../../api/storage.yaml

// Server is a minimal implementation of ServerInterface.
// It provides access to files from a local directory.
type Server struct {
	// Path to the directory containing the files
	path string
}

// New returns a Server pre-populated with the given files.
func New(path string) *Server {
	return &Server{path: path}
}

// GetFiles implements GET /files.
func (s *Server) GetFiles(ctx *gin.Context, params GetFilesParams) {
	var matches []FileMetadata

	// Glob files, computing metadata for the ones that match the prefix
	files, err := filepath.Glob(s.path + "/*")
	if err != nil {
		setInternalServerError(ctx, err, "failed to glob files")
		return
	}

	for _, f := range files {
		relPath, err := filepath.Rel(s.path, f)
		if err != nil {
			setInternalServerError(ctx, err, "failed to compute relative path")
			return
		}
		if params.Prefix != nil && !strings.HasPrefix(relPath, *params.Prefix) {
			continue
		}

		fileMeta, err := s.getFileMetadata(f)
		if err != nil {
			setInternalServerError(ctx, err, "failed to compute file metadata")
			return
		}

		matches = append(matches, fileMeta)
	}
	if matches == nil {
		matches = []FileMetadata{}
	}
	count := len(matches)
	ctx.JSON(http.StatusOK, ListFilesResponse{
		Files:     matches,
		FileCount: &count,
		Prefix:    params.Prefix,
	})
}

// GetFilesKey implements GET /files/{key}.
func (s *Server) GetFilesKey(ctx *gin.Context, key KeyParam, params GetFilesKeyParams) {
	path := filepath.Join(s.path, key)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, ErrorResponse{Message: "file not found"})
		return
	} else if err != nil {
		setInternalServerError(ctx, err, "failed to stat file")
		return
	}

	eTag := computeETag(info)
	if eTag != params.IfMatch {
		ctx.JSON(http.StatusPreconditionFailed, ErrorResponse{Message: "ETag mismatch"})
		return
	}

	file, err := os.Open(path)
	if err != nil {
		setInternalServerError(ctx, err, "failed to open file")
		return
	}
	defer file.Close()

	ctx.Header("ETag", eTag)
	ctx.Header("Last-Modified", info.ModTime().Format(time.RFC3339))
	ctx.DataFromReader(
		http.StatusOK,
		info.Size(),
		"application/octet-stream",
		// FIXME: how to stream file to response??
		// io.ReadCloser(file), ??
		nil,
	)
}

func (s *Server) getFileMetadata(path string) (FileMetadata, error) {
	key, err := filepath.Rel(s.path, path)
	if err != nil {
		return FileMetadata{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return FileMetadata{}, err
	}

	return FileMetadata{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		Etag:         computeETag(info),
	}, nil
}

// Compute ETag as hash of file path + size + last-modified-at
func computeETag(info os.FileInfo) string {
	hash := sha256.New()
	hash.Write([]byte(info.Name() + fmt.Sprintf("%d", info.Size()) + info.ModTime().String()))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func setInternalServerError(ctx *gin.Context, err error, msg string) {
	ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("%s: %v", msg, err)})
}
