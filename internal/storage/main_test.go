package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/types"
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

func TestParseLocationEmptyLocation(t *testing.T) {
	_, err := ParseLocation("")
	assert.Error(t, err)
	assert.ErrorIs(t, err, types.ErrInvalidObject)
}
