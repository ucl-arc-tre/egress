package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBucketNameFromLocation(t *testing.T) {
	_, err := ParseLocation("s3://bucket1/with/path")
	assert.Error(t, err)

	location, err := ParseLocation("s3://bucket1")
	assert.NoError(t, err)
	bucketName, err := location.BucketName()
	assert.NoError(t, err)
	assert.Equal(t, "bucket1", bucketName)

	location, err = ParseLocation("https://example.com/path")
	assert.NoError(t, err)
	_, err = location.BucketName()
	assert.Error(t, err)
}
