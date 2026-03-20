package main

import (
	"os"
	"path/filepath"
)

const (
	genericStorageLocation = "https://storage-server.storage.svc.cluster.local:8800/v0"
	storageRootOnHost      = "/tmp/storage"
)

type GenericProvider struct{}

func (p *GenericProvider) FilesLocation() string {
	return genericStorageLocation
}

func (p *GenericProvider) PutFile(key, content string) error {
	// Storage root directory is created when K3d is spun up
	dest := filepath.Join(storageRootOnHost, filepath.FromSlash(key))
	return os.WriteFile(dest, []byte(content), 0o644)
}
