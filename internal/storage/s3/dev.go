package s3

import (
	"context"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awsEndpoints "github.com/aws/smithy-go/endpoints"
	"github.com/rs/zerolog/log"

	"github.com/ucl-arc-tre/egress/internal/config"
)

func newDevClientWithBucket(cfg aws.Config) *awsS3.Client {
	client := awsS3.NewFromConfig(
		cfg,
		awsS3.WithEndpointResolverV2(DevResolver{}),
		func(o *awsS3.Options) { o.UsePathStyle = true },
	)
	mustCreateDevBucket(client)
	return client
}

func mustCreateDevBucket(client *awsS3.Client) {
	bucketName := config.DevS3Bucket()
	log.Debug().Str("bucketName", bucketName).Msg("Creating dev bucket")
	_, err := client.CreateBucket(context.Background(), &awsS3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil && strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
		log.Debug().Str("bucketName", bucketName).Msg("Bucket already exists")
	} else if err != nil {
		panic(err)
	}
}

type DevResolver struct{}

func (r DevResolver) ResolveEndpoint(ctx context.Context, params awsS3.EndpointParameters) (
	awsEndpoints.Endpoint, error,
) {
	url, err := url.Parse(config.DevS3URL())
	if err != nil {
		panic(err)
	}
	if params.Bucket != nil {
		url.Path = *params.Bucket
	}
	return awsEndpoints.Endpoint{URI: *url}, nil
}
