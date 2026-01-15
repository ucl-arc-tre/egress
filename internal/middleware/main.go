package middleware

import "github.com/ucl-arc-tre/egress/internal/openapi"

func New() []openapi.MiddlewareFunc {
	return []openapi.MiddlewareFunc{
		validateBasicAuth(),
	}
}
