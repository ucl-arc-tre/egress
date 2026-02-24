// Package server provides a minimal implementation of a generic storage server
package server

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
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

// New returns a Server that serves files from the given directory.
func New(path string) *Server {
	return &Server{path: path}
}

// GetFiles implements GET /files.
func (s *Server) GetFiles(ctx *gin.Context, params GetFilesParams) {
	var matches []FileMetadata

	err := fs.WalkDir(os.DirFS(s.path), ".", func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if params.Prefix != nil && !strings.HasPrefix(relPath, *params.Prefix) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		matches = append(matches, fileMetadata(relPath, info))
		return nil
	})
	if err != nil {
		setInternalServerError(ctx, err, "failed to walk directory")
		return
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

	// Prevent path traversal outside the server directory.
	if !strings.HasPrefix(path, filepath.Clean(s.path)+string(filepath.Separator)) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid key"})
		return
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, ErrorResponse{Message: "file not found"})
		return
	} else if err != nil {
		setInternalServerError(ctx, err, "failed to stat file")
		return
	}

	eTag := computeETag(key, info)
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
	ctx.Header("Last-Modified", info.ModTime().UTC().Format(time.RFC3339))
	ctx.DataFromReader(http.StatusOK, info.Size(), "application/octet-stream", file, nil)
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
		Etag:         computeETag(key, info),
	}
}

// computeETag returns a quoted ETag derived from the file's key, size, and mtime.
func computeETag(key string, info fs.FileInfo) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "%s:%d:%s", key, info.Size(), info.ModTime().String())
	return fmt.Sprintf("%q", fmt.Sprintf("%x", hash.Sum(nil)))
}

func setInternalServerError(ctx *gin.Context, err error, msg string) {
	ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: fmt.Sprintf("%s: %v", msg, err)})
}
