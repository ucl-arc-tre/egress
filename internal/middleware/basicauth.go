package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/openapi"
)

// Closure for validating HTTP Basic Auth against creds configured
func validateBasicAuth() func(*gin.Context) {
	creds := config.AuthBasicCredentials()
	return func(ctx *gin.Context) {
		username, password, ok := ctx.Request.BasicAuth()
		if !ok || username != creds.Username || password != creds.Password { /* pragma: allowlist secret */
			ctx.Header("WWW-Authenticate", "Basic")
			ctx.JSON(http.StatusUnauthorized, openapi.Unauthorized{
				Message: "Unauthorized; authentication required",
			})
			ctx.Abort()
			return
		}
	}
}
