package server

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/fs"
)

// ETagGenerator computes an ETag value for a file stored at path.
// Implementations must return a quoted string as per RFC 7232 (e.g. `"abc123"`).
type ETagGenerator interface {
	GenerateETag(path string, info fs.FileInfo) (string, error)
}

// DefaultETagGenerator is the out-of-the-box ETag strategy.
// It hashes the file key, size, and modification time using SHA-256.
type DefaultETagGenerator struct{}

// WithETagGenerator sets a custom ETag generation strategy.
func WithETagGenerator(g ETagGenerator) Option {
	if g == nil {
		// No-op when a nil generator is provided; keep the existing/default generator.
		return func(h *Handler) {}
	}
	return func(h *Handler) {
		h.etagGenerator = g
	}
}

func (g DefaultETagGenerator) GenerateETag(path string, info fs.FileInfo) (string, error) {
	hash := sha256.New()
	if err := binary.Write(hash, binary.LittleEndian, info.Size()); err != nil {
		return "", err
	}
	if err := binary.Write(hash, binary.LittleEndian, info.ModTime().UnixNano()); err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%x"`, hash.Sum(nil)), nil
}
