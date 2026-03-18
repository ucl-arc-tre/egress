package server

import "fmt"

type InvalidETagError struct {
	ETag    string
	Message string
}

func (e InvalidETagError) Error() string {
	return fmt.Sprintf("invalid ETag %s: %s", e.ETag, e.Message)
}
