package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	initWithPath(cf)

	assert.True(t, IsDebug())
}

func TestS3Credentials(t *testing.T) {
	yaml := `
s3:
  region: "eu-west-1"
  access_key_id: "access-key-123"
  secret_access_key: "secret-key-123"
`
	cf := makeConfig(t, "s3.yaml", yaml)
	initWithPath(cf)

	s3 := S3Credentials()
	assert.Equal(t, "eu-west-1", s3.Region)
	assert.Equal(t, "access-key-123", s3.AccessKeyId)
	assert.Equal(t, "secret-key-123", s3.SecretAccessKey)
}

func TestS3CredentialsWithMissingValues(t *testing.T) {
	yaml := `debug: false`

	cf := makeConfig(t, "no-s3-creds.yaml", yaml)
	initWithPath(cf)

	s3 := S3Credentials()
	assert.Equal(t, "", s3.Region)
	assert.Equal(t, "", s3.AccessKeyId)
	assert.Equal(t, "", s3.SecretAccessKey)
}

func makeConfig(t *testing.T, fileName string, yaml string) string {
	dir := t.TempDir()
	cf := filepath.Join(dir, fileName)

	err := os.WriteFile(cf, []byte(yaml), 0644)
	require.NoError(t, err, "Unable to create test config file")
	return cf
}
