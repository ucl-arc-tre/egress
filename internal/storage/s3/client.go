package s3

import (
	"context"
	"net/url"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awsEndpoints "github.com/aws/smithy-go/endpoints"

	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/config"
)

func makeResolver() awsS3.EndpointResolverV2 {
	if config.DevS3URL() != "" {
		log.Warn().Msg("S3Host is set - using dev resolver for s3")
		return DevResolver{}
	}
	return awsS3.NewDefaultEndpointResolverV2()
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
