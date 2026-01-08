package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	"github.com/ucl-arc-tre/egress/internal/types"
)

type Storage struct {
	client *awsS3.Client
}

func New() *Storage {
	return &Storage{client: newClient()}
}

func (s *Storage) List(ctx context.Context, location types.LocationURI) ([]types.ObjectMeta, error) {
	resp, err := s.client.ListObjectsV2(ctx, &awsS3.ListObjectsV2Input{
		Bucket: aws.String("bucket1"),
	})
	log.Debug().Any("resp", resp).Err(err).Msg("listed buckets")

	objectsMeta := []types.ObjectMeta{}
	bucketName, err := location.BucketName()
	if err != nil {
		return objectsMeta, err
	}
	objectPaginator := s.newListObjectsPaginator(bucketName)
	for objectPaginator.HasMorePages() {
		output, err := objectPaginator.NextPage(ctx)
		if err != nil {
			return objectsMeta, types.NewErrServerF("failed to list [%w]", err)
		}
		for _, o := range output.Contents {
			if o.Key == nil || o.ETag == nil || o.Size == nil || o.LastModified == nil {
				log.Error().Any("object", o).Msg("Object missing a required field")
				continue
			}
			objectsMeta = append(objectsMeta, types.ObjectMeta{
				Name:           *o.Key,
				Id:             types.FileId(stripQuotes(*o.ETag)),
				Size:           *o.Size,
				LastModifiedAt: *o.LastModified,
			})
		}
	}
	log.Debug().Any("location", location).Str("bucketName", bucketName).Msg("Found objects")
	return objectsMeta, nil
}

func (s *Storage) Get(ctx context.Context, location types.LocationURI, fileId types.FileId) (*types.Object, error) {
	bucketName, err := location.BucketName()
	if err != nil {
		return nil, err
	}
	objectKey, err := s.objectKeyWithFileId(ctx, bucketName, fileId)
	if err != nil {
		return nil, err
	}
	output, err := s.client.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    objectKey,
	})
	if err != nil {
		return nil, types.NewErrServerF("failed to get object [%w]", err)
	}
	if output.ETag != nil && !eTagEqualsFileId(output.ETag, fileId) {
		if err := output.Body.Close(); err != nil {
			return nil, types.NewErrServerF("failed to close [%w]", err)
		}
		return nil, types.NewErrNotFoundF("no object with fileId [%v]", fileId)
	}
	if output.ContentLength == nil {
		if err := output.Body.Close(); err != nil {
			return nil, types.NewErrServerF("failed to close [%w]", err)
		}
		return nil, types.NewErrServerF("object missing content length")
	}
	return &types.Object{Content: output.Body, Size: *output.ContentLength}, nil
}

func (s *Storage) objectKeyWithFileId(ctx context.Context, bucketName string, fileId types.FileId) (*string, error) {
	objectPaginator := s.newListObjectsPaginator(bucketName)
	for objectPaginator.HasMorePages() {
		output, err := objectPaginator.NextPage(ctx)
		if err != nil {
			return nil, types.NewErrServerF("failed to list objects [%v]", err)
		}
		for _, o := range output.Contents {
			if o.Key == nil || o.ETag == nil {
				log.Error().Any("object", o).Msg("Object missing a required field")
				continue
			}
			if eTagEqualsFileId(o.ETag, fileId) {
				return o.Key, nil
			}
		}
	}
	return nil, types.NewErrNotFoundF("no object with fileId [%v]", fileId)
}

func (s *Storage) newListObjectsPaginator(bucketName string) *awsS3.ListObjectsV2Paginator {
	input := awsS3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}
	return awsS3.NewListObjectsV2Paginator(s.client, &input)
}

func eTagEqualsFileId(eTag *string, fileId types.FileId) bool {
	if eTag == nil {
		return false
	}
	return *eTag == fmt.Sprintf(`"%v"`, fileId)
}

func stripQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, "")
}
