package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCredentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	"github.com/ucl-arc-tre/egress/internal/types"
)

type Storage struct {
	client *awsS3.Client
}

func New() *Storage {
	config, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithCredentialsProvider(awsCredentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     "rustfsadmin",
				SecretAccessKey: "rustfsadmin",
			},
		}),
		awsConfig.WithRegion("eu-west-2"),
	)
	if err != nil {
		panic(err)
	}
	storage := Storage{
		client: awsS3.NewFromConfig(config),
	}
	return &storage
}

func (s *Storage) List(ctx context.Context, location types.LocationURI) ([]types.ObjectMeta, error) {
	objectsMeta := []types.ObjectMeta{}
	bucketName, err := location.BucketName()
	if err != nil {
		return objectsMeta, err
	}
	objectPaginator := s.newListObjectsPaginator(bucketName)
	for objectPaginator.HasMorePages() {
		output, err := objectPaginator.NextPage(ctx)
		if err != nil {
			return objectsMeta, err
		}
		for _, o := range output.Contents {
			if o.Key == nil || o.ETag == nil || o.Size == nil || o.LastModified == nil {
				log.Error().Any("object", o).Msg("Object missing a required field")
				continue
			}
			objectsMeta = append(objectsMeta, types.ObjectMeta{
				Name:           *o.Key,
				Id:             types.FileId(*o.ETag),
				Size:           *o.Size,
				LastModifiedAt: *o.LastModified,
			})
		}
	}
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
	if output.ETag != nil && *output.ETag != string(fileId) {
		return nil, types.NewErrNotFoundF("no object with fileId [%v]", fileId)
	}
	return &types.Object{Content: output.Body}, nil
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
			if *o.ETag == string(fileId) {
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
