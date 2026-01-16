package middleware

import "github.com/ucl-arc-tre/egress/internal/openapi"

func All() []openapi.MiddlewareFunc {
	return []openapi.MiddlewareFunc{
		validateBasicAuth(),
	}
}
