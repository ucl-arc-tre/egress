package storage

import (
	"fmt"

	"github.com/ucl-arc-tre/egress/internal/storage/s3"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func Provider(kind types.StorageBackendKind) (Interface, error) {
	switch kind {
	case types.StorageBackendKindS3:
		return s3.New(), nil

	case types.StorageBackendKindGeneric:
		return nil, fmt.Errorf("generic storage backend not yet implemented")
	}
	// An unsupported backend should have been failed by Helm
	// So, this is fallback
	return nil, fmt.Errorf("unsupported storage backend %s", kind)
}
