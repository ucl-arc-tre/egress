package types

import (
	"net/url"
)

type StorageBackendKind string

const (
	StorageBackendKindS3      = StorageBackendKind("s3")
	StorageBackendKindGeneric = StorageBackendKind("generic")
	StorageBackendKindUnknown = StorageBackendKind("unknown")
)

// Location URI for a storage backend
// e.g.
//   - s3://example-bucket/a/path
//   - https://127.0.0.1:443/v1
type LocationURI url.URL

func (l LocationURI) StorageBackendKind() StorageBackendKind {
	switch l.Scheme {
	case "s3":
		return StorageBackendKindS3
	case "http", "https":
		return StorageBackendKindGeneric
	default:
		return StorageBackendKindUnknown
	}
}

func (l LocationURI) BucketName() (string, error) {
	if kind := l.StorageBackendKind(); l.StorageBackendKind() != StorageBackendKindS3 {
		return "", NewErrInvalidObjectF("storage backend kind not S3. [%v]", kind)
	}
	return l.Host, nil
}
