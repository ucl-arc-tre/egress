package types

import (
	"io"
	"time"
)

type Object struct {
	Content io.ReadCloser
}

type ObjectMeta struct {
	Name           string
	LastModifiedAt time.Time
	Id             FileId
	NumBytes       int
}
