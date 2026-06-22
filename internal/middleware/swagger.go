package middleware

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/openapi"
)

func swaggerMiddleware() openapi.MiddlewareFunc {
	swagger, err := openapi.GetSpec()
	if err != nil {
		panic(fmt.Sprintf("Error loading swagger spec\n: %s", err))
	}
	options := ginmiddleware.Options{
		ErrorHandler: func(ctx *gin.Context, message string, statusCode int) {
			log.Debug().Int("statusCode", statusCode).Msg(message)
			ctx.AbortWithStatusJSON(statusCode, openapi.ErrorResponse{
				Message: message,
			})
		},
		Options: openapi3filter.Options{
			// Authentication is handled with custom middleware. Noop to satisfy swagger
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
	}
	validator := ginmiddleware.OapiRequestValidatorWithOptions(swagger, &options)
	return openapi.MiddlewareFunc(validator)
}
