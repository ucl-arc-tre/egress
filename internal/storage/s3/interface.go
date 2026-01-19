package s3

import (
	"context"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type ClientInterface interface {
	ListObjectsV2(
		ctx context.Context,
		input *awsS3.ListObjectsV2Input,
		optFns ...func(*awsS3.Options),
	) (*awsS3.ListObjectsV2Output, error)
	GetObject(
		ctx context.Context,
		input *awsS3.GetObjectInput,
		optFns ...func(*awsS3.Options),
	) (*awsS3.GetObjectOutput, error)
}
