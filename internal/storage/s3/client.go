package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCredentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ucl-arc-tre/egress/internal/config"
)

func newClient() *awsS3.Client {
	credentials := config.S3Credentials()
	cfg, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithCredentialsProvider(awsCredentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     credentials.AccessKeyId,
				SecretAccessKey: credentials.SecretAccessKey,
			},
		}),
		awsConfig.WithRegion(credentials.Region),
	)
	if err != nil {
		panic(err)
	}
	if config.IsDevS3() {
		return newDevClientWithBucket(cfg)
	}
	return awsS3.NewFromConfig(cfg)
}
