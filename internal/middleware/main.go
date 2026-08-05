package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/openapi"
)

type authFunction func(*gin.Context)

func All() []openapi.MiddlewareFunc {
	return []openapi.MiddlewareFunc{
		authMiddleware(),
		swaggerMiddleware(),
	}
}

func authMiddleware() openapi.MiddlewareFunc {
	basicAuth := basicAuthenticator()
	bearerAuth := bearerAuthenticator()
	mtlsAuth := func(*gin.Context) {} // no-op auth handled at the transport layer

	return func(ctx *gin.Context) {
		var auth authFunction
		authHeader := ctx.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			auth = bearerAuth
		} else if strings.HasPrefix(authHeader, "Basic ") {
			auth = basicAuth
		} else if config.MutualTLSEnabled() {
			// if mTLS and bearer auth are both enabled then no additional
			// middleware is required for the mTLS part, as it's handled at
			// the transport layer - only the auth header needs to be parsed
			auth = mtlsAuth
		} else {
			fail(ctx, []string{"Basic", "Bearer"}, "authentication required")
			return
		}
		if auth == nil {
			fail(ctx, []string{}, "authentication method unavailable")
			return
		}
		auth(ctx)
	}
}

func fail(ctx *gin.Context, schemes []string, message string) {
	for _, s := range schemes {
		ctx.Writer.Header().Add("WWW-Authenticate", s+` realm="egress"`)
	}
	ctx.JSON(http.StatusUnauthorized, openapi.Unauthorized{
		Message: "Unauthorized; " + message,
	})
	ctx.Abort()
}
