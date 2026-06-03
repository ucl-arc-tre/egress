// A minimal OIDC-ish auth provider with JWKS to use with e2e tests
//
// Not a real IdP; does not verify users, has no persistence and accepts
// whatever claims the e2e tests ask us to sign. The purpose of this harness is
// to act as an in-cluster source of signed JWTs and matching JWKS documents,
// to enable exercising the Bearer token auth e2e without an eternal IdP
//
// Endpoints:
//
//	GET  /.well-known/openid-configuration  - minimal OIDC discovery doc
//	GET  /.well-known/jwks.json             - JWKS with a single RSA public key
//	POST /token                             - mint a JWT for the given claims
//	GET  /healthz                           - liveness/readiness probe
//
// The signing keypair is generated in-memory at startup; restarting rotates
// the key; tests should refetch JWKS (or rely on the JWKS cache refresh)
// after restarting this Pod
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	serverAddr = ":8900"
	keyBits    = 2048

	tokenDefaultIssuer   = "http://auth-server.auth.svc.cluster.local" + serverAddr
	tokenDefaultSubject  = "egressuser"
	tokenDefaultAudience = "egress"
	tokenTTL             = 1 * time.Hour
)

// Request body of POST /token; all fields are optional
type tokenRequest struct {
	Subject     string         `json:"sub,omitempty"`
	Audience    string         `json:"aud,omitempty"`
	ExtraClaims map[string]any `json:"claims,omitempty"`
}

// Response body of POST /token
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type signer struct {
	key    jwk.Key
	jwks   jwk.Set
	issuer string
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	issuer := envOrDefault("ISSUER_URL", tokenDefaultIssuer)

	s, err := newSigner(issuer)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialise signer")
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.GET("/.well-known/openid-configuration", s.handleDiscovery)
	router.GET("/.well-known/jwks.json", s.handleJWKS)
	router.POST("/token", s.handleToken)

	server := &http.Server{
		Addr:              serverAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Info().
		Str("serverAddr", serverAddr).
		Str("issuer", issuer).
		Msg("Starting JWKS server...")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("JWKS server exited with error")
	}
}

// Generates a RSA keypair, wraps it as a jwk.Key, and pre-builds
// the public JWKS that is served via /.well-known/jwks.json
func newSigner(iss string) (*signer, error) {
	rawKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}
	key, err := jwk.Import(rawKey)
	if err != nil {
		return nil, err
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		return nil, err
	}
	// Assign a key ID used in 'kid'
	if err := jwk.AssignKeyID(key); err != nil {
		return nil, err
	}
	// Extract public key
	pub, err := jwk.PublicKeyOf(key)
	if err != nil {
		return nil, err
	}
	// Create a JWKS with public key
	set := jwk.NewSet()
	if err := set.AddKey(pub); err != nil {
		return nil, err
	}
	return &signer{
		key:    key,
		jwks:   set,
		issuer: iss,
	}, nil
}

func (s *signer) handleDiscovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                s.issuer,
		"jwks_uri":                              s.issuer + "/.well-known/jwks.json",
		"token_endpoint":                        s.issuer + "/token",
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"response_types_supported":              []string{"token"},
		"subject_types_supported":               []string{"public"},
	})
}

func (s *signer) handleJWKS(c *gin.Context) {
	// jwk.Set implements json.Marshaler and emits {"keys":[...]}
	// with the correct encoding for each key
	c.JSON(http.StatusOK, s.jwks)
}

func (s *signer) handleToken(c *gin.Context) {
	req := tokenRequest{}
	// Allow an empty body; clients that don't have claims can just POST
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
			return
		}
	}
	if req.Subject == "" {
		req.Subject = tokenDefaultSubject
	}
	if req.Audience == "" {
		req.Audience = tokenDefaultAudience
	}
	now := time.Now()
	builder := jwt.NewBuilder().
		Issuer(s.issuer).
		Subject(req.Subject).
		Audience([]string{req.Audience}).
		IssuedAt(now).
		NotBefore(now).
		Expiration(now.Add(tokenTTL))

	// Callers cannot override the standard claims below
	// So ignore them if found in extra_claims
	reservedClaims := map[string]struct{}{
		"iss": {},
		"sub": {},
		"aud": {},
		"iat": {},
		"nbf": {},
		"exp": {},
	}
	for k, v := range req.ExtraClaims {
		if _, rc := reservedClaims[k]; rc {
			continue
		}
		builder = builder.Claim(k, v)
	}

	token, err := builder.Build()
	if err != nil {
		log.Error().Err(err).Msg("Failed to build token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), s.key))
	if err != nil {
		log.Error().Err(err).Msg("Failed to sign token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, tokenResponse{
		AccessToken: string(signed),
		IDToken:     string(signed),
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenTTL.Seconds()),
	})
}

func envOrDefault(key, d string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return d
}
