package storage

import (
	"fmt"

	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/storage/s3"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func Provider(cfg config.StorageConfigBundle) (Interface, error) {
	switch types.StorageBackendKind(cfg.Provider) {
	case types.StorageBackendKindS3:
		storage, err := s3.New(cfg.S3)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise s3 provider: %w", err)
		}
		return storage, nil

	case types.StorageBackendKindGeneric:
		return nil, fmt.Errorf("generic storage backend not yet implemented")
	}
	// An unsupported backend should have been failed by Helm
	// So, this is fallback
	return nil, fmt.Errorf("unsupported storage backend %s", cfg.Provider)
}
