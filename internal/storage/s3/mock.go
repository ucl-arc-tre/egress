package s3

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awsS3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func NewMock(client MockClient) *Storage {
	return &Storage{client: &client}
}

type MockObject struct {
	Key            string
	Etag           string
	LastModifiedAt time.Time
	Content        string
}

func (o MockObject) Size() int {
	return len([]rune(o.Content))
}

type MockBucket struct {
	Objects []MockObject
}

type MockBucketName = string

type MockClient struct {
	Buckets map[MockBucketName]MockBucket
}

func (c *MockClient) ListObjectsV2(
	_ context.Context,
	input *awsS3.ListObjectsV2Input,
	optFns ...func(*awsS3.Options),
) (*awsS3.ListObjectsV2Output, error) {
	if input.Bucket == nil {
		return nil, errors.New("no bucket")
	}
	bucket, exists := c.Buckets[*input.Bucket]
	if !exists {
		return nil, errors.New("bucket did not exist")
	}
	output := awsS3.ListObjectsV2Output{}
	for _, object := range bucket.Objects {
		output.Contents = append(output.Contents, awsS3types.Object{
			Key:          aws.String(object.Key),
			ETag:         aws.String(object.Etag),
			Size:         aws.Int64(int64(object.Size())),
			LastModified: aws.Time(object.LastModifiedAt),
		})
	}
	return &output, nil
}

func (c *MockClient) GetObject(
	_ context.Context,
	input *awsS3.GetObjectInput,
	optFns ...func(*awsS3.Options),
) (*awsS3.GetObjectOutput, error) {
	if input.Key == nil {
		return nil, errors.New("no input")
	}
	if input.Bucket == nil {
		return nil, errors.New("no bucket")
	}
	bucket, exists := c.Buckets[*input.Bucket]
	if !exists {
		return nil, errors.New("bucket did not exist")
	}
	idx := slices.IndexFunc(bucket.Objects, func(o MockObject) bool {
		return o.Key == *input.Key
	})
	if idx < 0 {
		return nil, errors.New("no object")
	}
	object := bucket.Objects[idx]
	output := awsS3.GetObjectOutput{
		ContentLength: aws.Int64(int64(object.Size())),
		Body:          io.NopCloser(strings.NewReader(object.Content)),
	}
	return &output, nil
}
