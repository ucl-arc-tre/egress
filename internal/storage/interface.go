package storage

import (
	"context"

	"github.com/ucl-arc-tre/egress/internal/types"
)

type Interface interface {
	List(ctx context.Context, location types.LocationURI) ([]types.FileMetadata, error)
	Get(ctx context.Context, location types.LocationURI, fileId types.FileId) (*types.File, error)
}
