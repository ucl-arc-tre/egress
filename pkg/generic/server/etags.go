package server

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/fs"
)

// ETagGenerator computes an ETag value for a stored file.
// Implementations must return a quoted string as per RFC 7232 (e.g. `"abc123"`).
type ETagGenerator interface {
	MakeETag(info fs.FileInfo) (string, error)
}

// DefaultETagGenerator is the out-of-the-box ETag strategy.
// It hashes the file key, size, and modification time using SHA-256.
type DefaultETagGenerator struct{}

// WithETagGenerator sets a custom ETag generation strategy.
func WithETagGenerator(g ETagGenerator) Option {
	if g == nil {
		panic("server: ETagGenerator must not be nil")
	}
	return func(h *Handler) {
		h.etagGenerator = g
	}
}

func (g DefaultETagGenerator) MakeETag(info fs.FileInfo) (string, error) {
	hash := sha256.New()
	hash.Write([]byte(info.Name()))
	if err := binary.Write(hash, binary.LittleEndian, info.Size()); err != nil {
		return "", err
	}
	if err := binary.Write(hash, binary.LittleEndian, info.ModTime().UnixNano()); err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%x"`, hash.Sum(nil)), nil
}
