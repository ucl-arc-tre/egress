package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

	return func(ctx *gin.Context) {
		var auth authFunction
		authHeader := ctx.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Basic ") {
			auth = basicAuth
		} else if strings.HasPrefix(authHeader, "Bearer ") {
			auth = bearerAuth
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
