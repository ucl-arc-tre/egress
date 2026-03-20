package main

import "os"

type StorageProvider interface {
	FilesLocation() string
	PutFile(key, content string) error
}

func newStorageProviderFromEnv() StorageProvider {
	switch os.Getenv("STORAGE_PROVIDER") {
	case "s3":
		return &S3Provider{}
	case "generic":
		return &GenericProvider{}
	}
	panic("STORAGE_PROVIDER not defined or has an invalid value")
}
