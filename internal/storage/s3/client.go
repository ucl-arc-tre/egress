package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCredentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ucl-arc-tre/egress/internal/config"
)

func newClient(s3Config config.S3StorageConfig) (*awsS3.Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithCredentialsProvider(awsCredentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     s3Config.AccessKeyId,
				SecretAccessKey: s3Config.SecretAccessKey,
			},
		}),
		awsConfig.WithRegion(s3Config.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("error configuring S3 client: %w", err)
	}
	if config.IsDevS3() {
		return newDevClientWithBucket(cfg), nil
	}
	return awsS3.NewFromConfig(cfg), nil
}
