package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucl-arc-tre/egress/internal/config"
)

const (
	username = "u1234"
	password = "p1234" // pragma: allowlist secret
	audience = "egress"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestBasicAuthValidCreds(t *testing.T) {
	initConfig(t, `
auth:
  basic:
    username: "`+username+`"
    password: "`+password+`"
`)
	ctx, rec, _ := contextAndRecorder(t)
	ctx.Request.SetBasicAuth(username, password)

	basic := basicAuthenticator()
	require.NotNil(t, basic)
	basic(ctx)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", ctx.GetString("sub")) // Basic auth doesnt set sub
}

func TestBasicAuthInvalidCreds(t *testing.T) {
	initConfig(t, `
auth:
  basic:
    username: "`+username+`"
    password: "`+password+`"
`)
	ctx, rec, _ := contextAndRecorder(t)
	ctx.Request.SetBasicAuth(username, "blah")

	basic := basicAuthenticator()
	require.NotNil(t, basic)
	basic(ctx)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBasicAuthMissingHeader(t *testing.T) {
	initConfig(t, `
auth:
  basic:
    username: "`+username+`"
    password: "`+password+`"
`)
	ctx, rec, _ := contextAndRecorder(t)

	basic := basicAuthenticator()
	require.NotNil(t, basic)
	basic(ctx)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBearerAuthValidToken(t *testing.T) {
	as, key := newAuthServer(t)
	issuer := as.URL

	initConfig(t, `
auth:
  bearer:
    issuer_url: "`+issuer+`"
    audience: "`+audience+`"
`)
	ctx, rec, _ := contextAndRecorder(t)
	token := signToken(t, key, issuer, audience, username)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)

	bearer := bearerAuthenticator()
	require.NotNil(t, bearer)

	bearer(ctx)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, username, ctx.GetString("sub"))
}

func TestBearerAuthInvalidToken(t *testing.T) {
	as, _ := newAuthServer(t)
	issuer := as.URL

	initConfig(t, `
auth:
  bearer:
    issuer_url: "`+issuer+`"
    audience: "`+audience+`"
`)
	ctx, rec, _ := contextAndRecorder(t)
	ctx.Request.Header.Set("Authorization", "Bearer bad-token")

	bearer := bearerAuthenticator()
	require.NotNil(t, bearer)

	bearer(ctx)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "", ctx.GetString("sub"))
}

func TestBearerAuthWrongAudience(t *testing.T) {
	as, key := newAuthServer(t)
	issuer := as.URL

	initConfig(t, `
auth:
  bearer:
    issuer_url: "`+issuer+`"
    audience: "`+audience+`"
`)
	ctx, rec, _ := contextAndRecorder(t)
	token := signToken(t, key, issuer, "wrong-audience", username)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)

	bearer := bearerAuthenticator()
	require.NotNil(t, bearer)

	bearer(ctx)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "", ctx.GetString("sub"))
}

func TestMiddlewareBasicAuthSuccess(t *testing.T) {
	initConfig(t, `
auth:
  basic:
    username: "`+username+`"
    password: "`+password+`"
`)
	auth := authMiddleware()

	var authedUserId string
	ctx, rec, router := contextAndRecorder(t)
	router.Use(gin.HandlerFunc(auth))
	router.GET("/", func(ctx *gin.Context) {
		authedUserId = ctx.GetString("sub")
		ctx.String(http.StatusOK, "Ok")
	})
	ctx.Request.SetBasicAuth(username, password)
	router.ServeHTTP(rec, ctx.Request)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", authedUserId) // Basic auth doesnt set sub
}

func TestMiddlewareBasicAuthFailure(t *testing.T) {
	initConfig(t, `
auth:
  basic:
    username: "`+username+`"
    password: "`+password+`"
`)
	auth := authMiddleware()

	ctx, rec, router := contextAndRecorder(t)
	router.Use(gin.HandlerFunc(auth))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "Ok")
	})
	ctx.Request.SetBasicAuth(username, "blah")
	router.ServeHTTP(rec, ctx.Request)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Basic auth failure must advertise Basic scheme
	wwwAuth := rec.Result().Header.Values("WWW-Authenticate")
	assert.Equal(t, []string{`Basic realm="egress"`}, wwwAuth)
}

func TestMiddlewareBasicAuthNoConfig(t *testing.T) {
	initConfig(t, `
auth:
  basic:
    username: ""
    password: ""
`)
	auth := authMiddleware()

	ctx, rec, router := contextAndRecorder(t)
	router.Use(gin.HandlerFunc(auth))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "Ok")
	})
	ctx.Request.SetBasicAuth(username, password)
	router.ServeHTTP(rec, ctx.Request)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareBearerAuthSuccess(t *testing.T) {
	as, key := newAuthServer(t)
	issuer := as.URL

	initConfig(t, `
auth:
  bearer:
    issuer_url: "`+issuer+`"
    audience: "`+audience+`"
`)
	auth := authMiddleware()

	var authedUserId string
	ctx, rec, router := contextAndRecorder(t)
	router.Use(gin.HandlerFunc(auth))
	router.GET("/", func(c *gin.Context) {
		authedUserId = c.GetString("sub")
		c.String(http.StatusOK, "Ok")
	})
	token := signToken(t, key, issuer, audience, username)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, ctx.Request)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, username, authedUserId)
}

func TestMiddlewareBearerInvalidToken(t *testing.T) {
	as, _ := newAuthServer(t)
	issuer := as.URL

	initConfig(t, `
auth:
  bearer:
    issuer_url: "`+issuer+`"
    audience: "`+audience+`"
`)
	auth := authMiddleware()

	var authedUserId string
	ctx, rec, router := contextAndRecorder(t)
	router.Use(gin.HandlerFunc(auth))
	router.GET("/", func(c *gin.Context) {
		authedUserId = c.GetString("sub")
		c.String(http.StatusOK, "Ok")
	})
	ctx.Request.Header.Set("Authorization", "Bearer bad-token")
	router.ServeHTTP(rec, ctx.Request)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "", authedUserId)

	// Bearer auth failure must advertise Bearer scheme
	wwwAuth := rec.Result().Header.Values("WWW-Authenticate")
	assert.Equal(t, []string{`Bearer realm="egress"`}, wwwAuth)
}

func TestMiddlewareBearerInvalidAudience(t *testing.T) {
	as, key := newAuthServer(t)
	issuer := as.URL

	initConfig(t, `
auth:
  bearer:
    issuer_url: "`+issuer+`"
    audience: "not-egress"
`)
	auth := authMiddleware()

	ctx, rec, router := contextAndRecorder(t)
	router.Use(gin.HandlerFunc(auth))
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Ok")
	})
	token := signToken(t, key, issuer, audience, username)
	ctx.Request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, ctx.Request)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareNoAuthHeader(t *testing.T) {
	initConfig(t, `
auth:
  basic:
    username: "`+username+`"
    password: "`+password+`"
`)
	auth := authMiddleware()

	ctx, rec, router := contextAndRecorder(t)
	router.Use(gin.HandlerFunc(auth))
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "Ok")
	})
	router.ServeHTTP(rec, ctx.Request)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// When no auth header, both Basic and Bearer schemes must be advertised
	wwwAuth := rec.Result().Header.Values("WWW-Authenticate")
	assert.Contains(t, wwwAuth, `Basic realm="egress"`)
	assert.Contains(t, wwwAuth, `Bearer realm="egress"`)
}

func initConfig(t *testing.T, yaml string) {
	t.Helper()
	cf := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cf, []byte(yaml), 0644))
	config.InitWithPath(cf)
}

func contextAndRecorder(t *testing.T) (*gin.Context, *httptest.ResponseRecorder, *gin.Engine) {
	t.Helper()
	rec := httptest.NewRecorder()
	ctx, router := gin.CreateTestContext(rec)
	ctx.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	return ctx, rec, router
}

func newAuthServer(t *testing.T) (*httptest.Server, jwk.Key) {
	t.Helper()
	rawKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKey, err := jwk.Import(rawKey)
	require.NoError(t, err)
	require.NoError(t, privateKey.Set(jwk.KeyIDKey, "test-key"))
	require.NoError(t, privateKey.Set(jwk.AlgorithmKey, jwa.RS256()))

	publicKey, err := jwk.Import(rawKey.Public())
	require.NoError(t, err)
	require.NoError(t, publicKey.Set(jwk.KeyIDKey, "test-key"))
	require.NoError(t, publicKey.Set(jwk.AlgorithmKey, jwa.RS256()))

	pubSet := jwk.NewSet()
	require.NoError(t, pubSet.AddKey(publicKey))

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		// Determine the base URL from the request
		base := "http://" + r.Host
		resp := map[string]string{
			"issuer":   base + "/",
			"jwks_uri": base + "/.well-known/jwks.json",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pubSet)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, privateKey
}

func signToken(t *testing.T, key jwk.Key, iss, aud, sub string) string {
	t.Helper()
	token, err := jwt.NewBuilder().
		Issuer(iss).
		Audience([]string{aud}).
		Subject(sub).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(5 * time.Minute)).
		Build()
	require.NoError(t, err)

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), key))
	require.NoError(t, err)
	return string(signed)
}
