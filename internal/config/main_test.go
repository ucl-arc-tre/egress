package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucl-arc-tre/egress/internal/types"
)

func TestServerAddressSetPort(t *testing.T) {
	t.Setenv("PORT", "1234")
	assert.Equal(t, ":1234", ServerAddress())
}

func TestServerAddressDefault(t *testing.T) {
	assert.Equal(t, ":8080", ServerAddress())
}

func TestDebugTrue(t *testing.T) {
	yaml := `debug: true`

	cf := makeConfig(t, "debug.yaml", yaml)
	InitWithPath(cf)

	assert.True(t, IsDebug())
}

func TestStorageConfigS3(t *testing.T) {
	yaml := `
storage:
  provider: s3
  s3:
    region: "us-east-1"
    access_key_id: "s3-access-key-123"
    secret_access_key: "s3-secret-key-123"
`
	cf := makeConfig(t, "storage-s3.yaml", yaml)
	InitWithPath(cf)

	storage := StorageConfig()
	assert.Equal(t, string(types.StorageBackendKindS3), storage.Provider)
	assert.Equal(t, "us-east-1", storage.S3.Region)
	assert.Equal(t, "s3-access-key-123", storage.S3.AccessKeyId)
	assert.Equal(t, "s3-secret-key-123", storage.S3.SecretAccessKey)
}

func TestStorageConfigGeneric(t *testing.T) {
	yaml := `
storage:
  provider: generic
  generic: {}
`
	cf := makeConfig(t, "storage-generic.yaml", yaml)
	InitWithPath(cf)

	storage := StorageConfig()
	assert.Equal(t, string(types.StorageBackendKindGeneric), storage.Provider)
}

func TestDBConfig(t *testing.T) {
	yaml := `
db:
  provider: rqlite
  rqlite:
    baseUrl: "http://rqlite.local"
    username: "dbusername123"
    password: "dbpassword123"
`
	cf := makeConfig(t, "db.yaml", yaml)
	InitWithPath(cf)

	db := DBConfig()
	assert.Equal(t, string(types.DBProviderRqlite), db.Provider)
	assert.Equal(t, "http://rqlite.local", db.Rqlite.BaseURL)
	assert.Equal(t, "dbusername123", db.Rqlite.Username)
	assert.Equal(t, "dbpassword123", db.Rqlite.Password)
}

func TestAuthBasicCredentials(t *testing.T) {
	yaml := `
auth:
  basic:
    username: "username123"
    password: "password123"
`
	cf := makeConfig(t, "basic-auth.yaml", yaml)
	InitWithPath(cf)

	auth := AuthBasicCredentials()
	assert.Equal(t, "username123", auth.Username)
	assert.Equal(t, "password123", auth.Password)
}

func makeConfig(t *testing.T, fileName string, yaml string) string {
	dir := t.TempDir()
	cf := filepath.Join(dir, fileName)

	err := os.WriteFile(cf, []byte(yaml), 0644)
	require.NoError(t, err, "Unable to create test config file")
	return cf
}
