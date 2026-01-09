package types

import (
	"io"
	"time"
)

// Unique file identifier. e.g. a SHA512 checksum
type FileId string

type File struct {
	Content io.ReadCloser
	Size    int64 // Number of  bytes
}

type FileMetadata struct {
	Name           string
	LastModifiedAt time.Time
	Id             FileId
	Size           int64 // Number of  bytes
}
