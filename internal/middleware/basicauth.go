package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/config"
)

// Closure for authenticating HTTP Basic
func basicAuthenticator() authFunction {
	cfg := config.BasicAuthConfig()
	if cfg.Username == "" || cfg.Password == "" {
		log.Info().Msg("Basic auth not configured")
		return nil
	}

	return func(ctx *gin.Context) {
		username, password, ok := ctx.Request.BasicAuth()
		if !ok || username != cfg.Username || password != cfg.Password { /* pragma: allowlist secret */
			fail(ctx, []string{"Basic"}, "authentication failed")
		}
	}
}
