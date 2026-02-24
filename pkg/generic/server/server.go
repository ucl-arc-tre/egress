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
	"github.com/rs/zerolog/log"
)

//go:generate go tool oapi-codegen -generate gin -package server -o server.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate spec -package server -o spec.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate types -package server -o types.gen.go ../../../api/storage.yaml

// Handler is a minimal implementation of the storage OAPI spec.
// It serves files from a local directory.
type Handler struct {
	// Path to the directory containing the files
	path string
}

// New returns a Server that serves files from the given directory.
func New(path string) *Handler {
	return &Handler{path: filepath.Clean(path)}
}

// GetFiles implements GET /files.
func (h *Handler) GetFiles(ctx *gin.Context, params GetFilesParams) {
	matches := []FileMetadata{}

	// Prefix should be a relative path local to the server's root
	if params.Prefix != nil && !prefixIsValid(*params.Prefix) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid prefix"})
		return
	}

	root, err := os.OpenRoot(h.path)
	if err != nil {
		setInternalServerError(ctx, err, "failed to open server root")
		return
	}
	defer root.Close()

	err = fs.WalkDir(root.FS(), ".", func(relPath string, d fs.DirEntry, err error) error {
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

	count := len(matches)
	ctx.JSON(http.StatusOK, ListFilesResponse{
		Files:     matches,
		FileCount: &count,
		Prefix:    params.Prefix,
	})
}

// GetFile implements GET /file.
func (h *Handler) GetFile(ctx *gin.Context, params GetFileParams) {
	if !keyIsValid(params.Key) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid key"})
		return
	}
	if !eTagIsValid(params.IfMatch) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: `invalid If-Match header. ETag must be a quoted string, e.g. "abc123"`})
		return
	}

	// Prevent path traversal by opening in root
	file, err := os.OpenInRoot(h.path, params.Key)
	if os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, ErrorResponse{Message: "file not found"})
		return
	} else if err != nil {
		setInternalServerError(ctx, err, "failed to open file")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		setInternalServerError(ctx, err, "failed to stat file")
		return
	}
	if info.IsDir() {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "key must refer to a file, not a directory"})
		return
	}

	eTag := computeETag(params.Key, info)
	if eTag != params.IfMatch {
		ctx.JSON(http.StatusPreconditionFailed, ErrorResponse{Message: "ETag mismatch"})
		return
	}

	ctx.Header("ETag", eTag)
	ctx.Header("Last-Modified", info.ModTime().UTC().Format(time.RFC3339))
	ctx.DataFromReader(http.StatusOK, info.Size(), "application/octet-stream", file, nil)
}

func prefixIsValid(prefix string) bool {
	return prefix != "" && filepath.IsLocal(prefix)
}

func keyIsValid(key string) bool {
	return key != "" && filepath.IsLocal(key) && !strings.HasSuffix(key, "/")
}

// eTagIsValid reports whether s is a quoted ETag string as per RFC 7232.
func eTagIsValid(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

func fileMetadata(key string, info fs.FileInfo) FileMetadata {
	return FileMetadata{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		Etag:         computeETag(key, info),
	}
}

func computeETag(key string, info fs.FileInfo) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "%s:%d:%s", key, info.Size(), info.ModTime().String())
	return fmt.Sprintf("%q", fmt.Sprintf("%x", hash.Sum(nil)))
}

func setInternalServerError(ctx *gin.Context, err error, msg string) {
	log.Error().Err(err).Msg(msg)
	ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: msg})
}
