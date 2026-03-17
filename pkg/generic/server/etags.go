package server

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/fs"
)

type ETagGenerator interface {
	MakeETag(key string, info fs.FileInfo) (string, error)
}

type DefaultETagGenerator struct{}

// WithETagGenerator sets a custom ETag generation strategy.
func WithETagGenerator(g ETagGenerator) Option {
	return func(h *Handler) {
		h.etagGenerator = g
	}
}

func (g DefaultETagGenerator) MakeETag(key string, info fs.FileInfo) (string, error) {
	hash := sha256.New()
	hash.Write([]byte(key))
	if err := binary.Write(hash, binary.LittleEndian, info.Size()); err != nil {
		return "", err
	}
	if err := binary.Write(hash, binary.LittleEndian, info.ModTime().UnixNano()); err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%x"`, hash.Sum(nil)), nil
}
