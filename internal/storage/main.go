package storage

import (
	"net/url"

	"github.com/ucl-arc-tre/egress/internal/types"
)

func ParseLocation(raw string) (*types.LocationURI, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	uri := types.LocationURI(*parsed)
	if uri.Path != "" && uri.StorageBackendKind() == types.StorageBackendKindS3 {
		return nil, types.NewErrInvalidObjectF("s3 location [%s] cannot have path", raw)
	}
	return &uri, nil
}
