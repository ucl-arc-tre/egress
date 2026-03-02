package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/storage/s3"
	"github.com/ucl-arc-tre/egress/internal/types"
)

// Need to initialise config for this test
func TestS3StorageKind(t *testing.T) {
	yaml := `
storage:
  provider: s3
  s3:
    region: us-east-1
    access_key_id: key1234
    secret_access_key: secret1234
dev:
  s3:
    url: ""
    bucket: ""
`
	cf := makeConfig(t, "s3.yaml", yaml)
	config.InitWithPath(cf)

	cfg := config.StorageConfigBundle{
		Provider: string(types.StorageBackendKindS3),
	}
	storage, err := Provider(cfg)
	assert.NoError(t, err)
	assert.IsType(t, &s3.Storage{}, storage)
}

func TestUnsupportedProvider(t *testing.T) {
	cfg := config.StorageConfigBundle{
		Provider: "blah",
	}
	storage, err := Provider(cfg)
	assert.Error(t, err)
	assert.Nil(t, storage)
}

func makeConfig(t *testing.T, fileName string, yaml string) string {
	dir := t.TempDir()
	cf := filepath.Join(dir, fileName)

	err := os.WriteFile(cf, []byte(yaml), 0644)
	require.NoError(t, err, "Unable to create test config file")
	return cf
}
