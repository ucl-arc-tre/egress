package middleware

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	jwtv "github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/config"
)

const tokenCacheDuration = 5 * time.Minute

// Closure for authenticating HTTP Bearer
func bearerAuthenticator() authFunction {
	cfg := config.BearerAuthConfig()
	if cfg.IssuerURL == "" || cfg.Audience == "" {
		log.Info().Msg("Bearer auth not configured")
		return nil
	}

	issuer, err := url.Parse(cfg.IssuerURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse token issuer URL")
		return nil
	}

	provider := jwks.NewCachingProvider(issuer, tokenCacheDuration)
	validator, err := jwtv.New(
		provider.KeyFunc,
		jwtv.RS256,
		issuer.String(),
		[]string{cfg.Audience},
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create token validator")
		return nil
	}

	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer")

		claims, err := validator.ValidateToken(context.Background(), token)
		if err != nil {
			fail(ctx, []string{"Bearer"}, err.Error())
			return
		}
		validated, ok := claims.(*jwtv.ValidatedClaims)
		if !ok {
			fail(ctx, []string{"Bearer"}, "unexpected claims type")
			return
		}
		// Save authenticated user ID (i.e. sub) to cross-check against
		// the user-id argument (if any) of the API request
		ctx.Set("sub", validated.RegisteredClaims.Subject)
	}
}
