package storage

import "github.com/ucl-arc-tre/egress/internal/types"

type Interface interface {
	List(location types.LocationURI) ([]types.ObjectMeta, error)
	Get(location types.LocationURI, fileId types.FileId) (*types.Object, error)
}
