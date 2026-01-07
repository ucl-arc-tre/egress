package types

import "net/url"

const (
	StorageBackendKindS3      = StorageBackendKind("s3")
	StorageBackendKindGeneric = StorageBackendKind("generic")
	StorageBackendKindUnknown = StorageBackendKind("unknown")
)

type StorageBackendKind string

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
