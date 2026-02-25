// Package server provides a minimal implementation of a generic storage server
package server

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

//go:generate go tool oapi-codegen -generate gin -package server -o main.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate spec -package server -o spec.gen.go ../../../api/storage.yaml
//go:generate go tool oapi-codegen -generate types -package server -o types.gen.go ../../../api/storage.yaml

const (
	maxKeyLen    = 1024 // bytes, same limit that S3 uses
	maxFileCount = 1000 // limit number of files returned. Should match maximum in ../../../api/storage.yaml
)

// Handler is a minimal implementation of the storage OAPI spec.
// It serves files from a local directory.
type Handler struct {
	// Path to the directory containing the files
	rootDirPath string
}

// New returns a Handler that serves files from the given directory.
func New(rootDirPath string) *Handler {
	return &Handler{rootDirPath: filepath.Clean(rootDirPath)}
}

// GetFiles implements GET /files.
func (h *Handler) GetFiles(ctx *gin.Context, params GetFilesParams) {
	matches := []FileMetadata{}

	// Prefix should be a relative rootDirPath local to the server's root
	if params.Prefix != nil && !isValidPrefix(*params.Prefix) {
		badRequest(ctx, "invalid prefix")
		return
	}

	root, err := os.OpenRoot(h.rootDirPath)
	if err != nil {
		internalServerError(ctx, err, "failed to open server root")
		return
	}
	defer root.Close()

	count := 0
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
		meta, err := fileMetadata(relPath, info)
		if err != nil {
			return err
		}
		matches = append(matches, meta)
		count += 1
		if count > maxFileCount {
			log.Info().Msg("Maximum file count reached, truncating output.")
			return nil
		}
		return nil
	})
	if err != nil {
		internalServerError(ctx, err, "failed to walk directory")
		return
	}

	ctx.JSON(http.StatusOK, ListFilesResponse{
		Files:     matches,
		FileCount: count,
		Prefix:    params.Prefix,
	})
}

// GetFile implements GET /file.
func (h *Handler) GetFile(ctx *gin.Context, params GetFileParams) {
	if !isValidKey(params.Key) {
		badRequest(ctx, "invalid key")
		return
	}

	requestedETag := params.IfMatch
	if !isValidETag(requestedETag) {
		badRequest(ctx, `invalid If-Match header. ETag must be a quoted string, e.g. "abc123"`)
		return
	}

	// Prevent rootDirPath traversal by opening in root
	file, err := os.OpenInRoot(h.rootDirPath, params.Key)
	if errors.Is(err, fs.ErrNotExist) {
		ctx.JSON(http.StatusNotFound, ErrorResponse{Message: "file not found"})
		return
	} else if err != nil {
		internalServerError(ctx, err, "failed to open file")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		internalServerError(ctx, err, "failed to stat file")
		return
	}
	if info.IsDir() {
		badRequest(ctx, "key must refer to a file, not a directory")
	}

	eTag, err := makeETag(params.Key, info)
	if err != nil {
		internalServerError(ctx, err, "failed to compute ETag")
		return
	}
	if eTag != requestedETag {
		ctx.JSON(http.StatusPreconditionFailed, ErrorResponse{Message: "ETag mismatch"})
		return
	}

	ctx.Header("ETag", eTag)
	ctx.Header("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	ctx.DataFromReader(http.StatusOK, info.Size(), "application/octet-stream", file, nil)
}

func isValidPrefix(prefix string) bool {
	// Treat an empty prefix as valid (equivalent to "no prefix" per API spec),
	// while still enforcing locality for any non-empty prefix.
	return prefix == "" || filepath.IsLocal(prefix)
}

func isValidKey(key string) bool {
	return key != "" && len(key) <= maxKeyLen && filepath.IsLocal(key) && !strings.HasSuffix(key, "/")
}

// isValidETag reports whether s is a quoted ETag string as per RFC 7232.
func isValidETag(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

func fileMetadata(key string, info fs.FileInfo) (FileMetadata, error) {
	etag, err := makeETag(key, info)
	if err != nil {
		return FileMetadata{}, err
	}
	return FileMetadata{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		Etag:         etag,
	}, nil
}

func makeETag(key string, info fs.FileInfo) (string, error) {
	hash := sha256.New()
	hash.Write([]byte(key))
	if err := binary.Write(hash, binary.LittleEndian, info.Size()); err != nil {
		return "", err
	}
	if err := binary.Write(hash, binary.LittleEndian, info.ModTime().Unix()); err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%x"`, hash.Sum(nil)), nil
}

func internalServerError(ctx *gin.Context, err error, msg string) {
	log.Error().Err(err).Msg(msg)
	ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: msg})
}

func badRequest(ctx *gin.Context, msg string) {
	ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: msg})
}
