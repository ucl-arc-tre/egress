package s3

import "github.com/ucl-arc-tre/egress/internal/types"

type Storage struct {
}

func New() *Storage {
	storage := Storage{}
	return &storage
}

func (s *Storage) List(location types.LocationURI) ([]types.ObjectMeta, error) {
	return nil, nil
}

func (s *Storage) Get(location types.LocationURI, fileId types.FileId) (*types.Object, error) {
	return nil, nil
}
