package middleware

import (
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
	issuer, _ := url.Parse(cfg.IssuerURL) // Issuer url has already been validated

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

		claims, err := validator.ValidateToken(ctx, token)
		if err != nil {
			fail(ctx, []string{"Bearer"}, "could not validate token")
			log.Error().Err(err).Msg("failed to validate bearer token")
			return
		}
		validated, ok := claims.(*jwtv.ValidatedClaims)
		if !ok {
			fail(ctx, []string{"Bearer"}, "could not read token claims")
			log.Error().Msg("failed to assert validated claims type")
			return
		}
		// Save authenticated user ID (i.e. sub) to cross-check against
		// the user-id argument (if any) of the API request
		ctx.Set("sub", validated.RegisteredClaims.Subject)
	}
}
