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
	initTestConfig(t, "test.yaml", `
auth:
  mTLS:
    dir: null
`)
	t.Setenv("HTTP_PORT", "1234")
	t.Setenv("HTTPS_PORT", "1235")
	assert.Equal(t, ":1234", ServerAddress())

	initTestConfig(t, "test.yaml", `
auth:
  mTLS:
    dir: /etc/tls
`)
	assert.Equal(t, ":1235", ServerAddress())
}

func TestServerAddressDefault(t *testing.T) {
	initTestConfig(t, "test.yaml", `
auth:
  mTLS:
    dir: null
`)
	assert.Equal(t, ":8000", ServerAddress())

	initTestConfig(t, "test.yaml", `
auth:
  mTLS:
    dir: /etc/tls
`)
	assert.Equal(t, ":8443", ServerAddress())
}

func TestDebugTrue(t *testing.T) {
	initTestConfig(t, "debug.yaml", `debug: true`)
	assert.True(t, IsDebug())
}

func TestStorageConfigS3(t *testing.T) {
	initTestConfig(t, "storage-s3.yaml", `
storage:
  provider: s3
  s3:
    region: "us-east-1"
    access_key_id: "s3-access-key-123"
    secret_access_key: "s3-secret-key-123"
`)

	storage := StorageConfig()
	assert.Equal(t, string(types.StorageProviderS3), storage.Provider)
	assert.Equal(t, "us-east-1", storage.S3.Region)
	assert.Equal(t, "s3-access-key-123", storage.S3.AccessKeyId)
	assert.Equal(t, "s3-secret-key-123", storage.S3.SecretAccessKey)
}

func TestStorageConfigGeneric(t *testing.T) {
	initTestConfig(t, "storage-generic.yaml", `
storage:
  provider: generic
  generic: {}
`)

	storage := StorageConfig()
	assert.Equal(t, string(types.StorageProviderGeneric), storage.Provider)
}

func TestDBConfig(t *testing.T) {
	initTestConfig(t, "db.yaml", `
db:
  provider: rqlite
  rqlite:
    baseUrl: "http://rqlite.local"
    username: "dbusername123"
    password: "dbpassword123"
`)

	db := DBConfig()
	assert.Equal(t, string(types.DBProviderRqlite), db.Provider)
	assert.Equal(t, "http://rqlite.local", db.Rqlite.BaseURL)
	assert.Equal(t, "dbusername123", db.Rqlite.Username)
	assert.Equal(t, "dbpassword123", db.Rqlite.Password)
}

func TestBearerAuthConfig(t *testing.T) {
	initTestConfig(t, "bearer-auth.yaml", `
auth:
  bearer:
    issuer_url: "http://example.com"
    audience: "egress"
`)
	auth := BearerAuthConfig()
	assert.Equal(t, "http://example.com", auth.IssuerURL)
	assert.Equal(t, "egress", auth.Audience)
}

func initTestConfig(t *testing.T, fileName string, yaml string) {
	dir := t.TempDir()
	configFilepath := filepath.Join(dir, fileName)

	err := os.WriteFile(configFilepath, []byte(yaml), 0644)
	require.NoError(t, err, "Unable to create test config file")
	InitWithPath(configFilepath)
}
