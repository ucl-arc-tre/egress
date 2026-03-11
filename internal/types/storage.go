package types

import (
	"net/url"
)

type StorageProvider string

const (
	StorageProviderS3      = StorageProvider("s3")
	StorageProviderGeneric = StorageProvider("generic")
	StorageProviderUnknown = StorageProvider("unknown")
)

// Location URI for a storage backend
// e.g.
//   - s3://example-bucket/a/path
//   - https://127.0.0.1:443/v1
type LocationURI url.URL

func (l LocationURI) StorageProvider() StorageProvider {
	switch l.Scheme {
	case "s3":
		return StorageProviderS3
	case "http", "https":
		return StorageProviderGeneric
	default:
		return StorageProviderUnknown
	}
}

func (l LocationURI) BucketName() (string, error) {
	if provider := l.StorageProvider(); provider != StorageProviderS3 {
		return "", NewErrInvalidObjectF("storage provider not S3. [%v]", provider)
	}
	return l.Host, nil
}
