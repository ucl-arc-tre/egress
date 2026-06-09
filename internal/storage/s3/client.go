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

// newClient generates a new API client for AWS S3, optionally configuring the AWS AccessKeyId and SecretAccessKey if provided.
func newClient(s3Config config.S3StorageConfig) (*awsS3.Client, error) {
	opts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(s3Config.Region),
	}
	if s3Config.AccessKeyId != "" && s3Config.SecretAccessKey != "" {
		opts = append(opts, awsConfig.WithCredentialsProvider(
			awsCredentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     s3Config.AccessKeyId,
					SecretAccessKey: s3Config.SecretAccessKey,
				},
			},
		))
	}
	cfg, err := awsConfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("error configuring S3 client: %w", err)
	}
	if config.IsDevS3() {
		return newDevClientWithBucket(cfg), nil
	}
	return awsS3.NewFromConfig(cfg), nil
}
