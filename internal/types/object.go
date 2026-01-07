package types

import (
	"io"
	"time"
)

type Object struct {
	Content io.ReadCloser
	Size    int64 // Number of  bytes
}

type ObjectMeta struct {
	Name           string
	LastModifiedAt time.Time
	Id             FileId
	Size           int64 // Number of  bytes
}
