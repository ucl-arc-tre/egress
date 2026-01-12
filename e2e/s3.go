package main

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCredentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awsEndpoints "github.com/aws/smithy-go/endpoints"
)

const (
	bucketName = "bucket1"
)

func newS3Client() *awsS3.Client {
	cfg := must(awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithCredentialsProvider(awsCredentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     "rustfsadmin",
				SecretAccessKey: "rustfsadmin",
			},
		}),
		awsConfig.WithRegion("us-east-1"),
	))
	client := awsS3.NewFromConfig(
		cfg,
		awsS3.WithEndpointResolverV2(DevResolver{}),
		func(o *awsS3.Options) { o.UsePathStyle = true },
	)
	return client
}

type DevResolver struct{}

func (r DevResolver) ResolveEndpoint(ctx context.Context, params awsS3.EndpointParameters) (
	awsEndpoints.Endpoint, error,
) {
	url, err := url.Parse("http://localhost:8081")
	if err != nil {
		panic(err)
	}
	if params.Bucket != nil {
		url.Path = *params.Bucket
	}
	return awsEndpoints.Endpoint{URI: *url}, nil
}

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}

func must[T any](obj T, err error) T {
	assertNoError(err)
	return obj
}
