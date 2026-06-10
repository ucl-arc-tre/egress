package s3

import (
	"os"
	"path"
	"testing"

	awsCredentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/ucl-arc-tre/egress/internal/config"
)

func TestEmptyCredsLoadsDefaultConfig(t *testing.T) {
	tmpConfigPath := path.Join(t.TempDir(), "config.yaml")
	os.WriteFile(tmpConfigPath, []byte(""), 0o775)
	config.InitWithPath(tmpConfigPath)

	client, err := newClient(config.S3StorageConfig{Region: "eu-west-2"})
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// When no creds are provided, the default credential chain should be used
	// rather than a StaticCredentialsProvider.
	_, isStatic := client.Options().Credentials.(*awsCredentials.StaticCredentialsProvider)
	assert.False(t, isStatic, "expected default credentials chain, not StaticCredentialsProvider")
}
