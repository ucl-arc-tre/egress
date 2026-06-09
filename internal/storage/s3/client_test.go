package s3

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/config"
)

func TestEmptyCredsLoadsDefaultConfig(t *testing.T) {
	tmpConfigPath := path.Join(t.TempDir(), "config.yaml")
	os.WriteFile(tmpConfigPath, []byte(""), 0o775)
	config.InitWithPath(tmpConfigPath)

	client, err := newClient(config.S3StorageConfig{Region: "eu-west-2"})
	assert.NoError(t, err)

	creds, err := client.Options().Credentials.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "", creds.AccessKeyID)
	assert.Equal(t, "", creds.SecretAccessKey)
}
