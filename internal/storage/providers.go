package storage

import (
	"fmt"

	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/storage/generic"
	"github.com/ucl-arc-tre/egress/internal/storage/s3"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func Provider(cfg config.StorageConfigBundle) (Interface, error) {
	switch types.StorageProvider(cfg.Provider) {
	case types.StorageProviderS3:
		storage, err := s3.New(cfg.S3)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise s3 provider: %w", err)
		}
		return storage, nil

	case types.StorageProviderGeneric:
		storage, err := generic.New(cfg.TLSCertDir)
		if err != nil {
			return nil, fmt.Errorf("failed to initialise generic provider: %w", err)
		}
		return storage, nil
	}
	// An unsupported backend should have been failed by Helm
	// So, this is fallback
	return nil, fmt.Errorf("unsupported storage backend %s", cfg.Provider)
}
