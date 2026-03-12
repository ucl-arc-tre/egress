package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageProviderFromLocation(t *testing.T) {
	tests := []struct {
		scheme   string
		expected StorageProvider
	}{
		{scheme: "s3", expected: StorageProviderS3},
		{scheme: "http", expected: StorageProviderGeneric},
		{scheme: "https", expected: StorageProviderGeneric},
		{scheme: "blah", expected: StorageProviderUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.scheme, func(t *testing.T) {
			location := LocationURI{Scheme: tc.scheme}
			assert.Equal(t, tc.expected, location.StorageProvider())
		})
	}
}

func TestBucketNameFromLocation(t *testing.T) {
	genericLocation := LocationURI{Scheme: "http"}
	_, err := genericLocation.BucketName()
	assert.Error(t, err)

	bucketName := "bucket1"
	s3Location := LocationURI{Scheme: "s3", Host: bucketName}
	actual, err := s3Location.BucketName()
	assert.NoError(t, err)
	assert.Equal(t, bucketName, actual)
}
