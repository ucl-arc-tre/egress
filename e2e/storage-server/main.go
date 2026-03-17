// A minimal generic storage server used in e2e tests
// This wraps the pkg/generic/server handler and exposes the REST API
// defined in api/storage.yaml on a hard-coded address
package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	gsserver "github.com/ucl-arc-tre/egress/pkg/generic/server"
)

const (
	serverAddr = ":8800"
	basePath   = "/v0"
	rootDir    = "/storage-root"
	tlsCertDir = "/etc/storage-server/tls"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		log.Fatal().Err(err).Str("dir", rootDir).Msg("failed to create root directory")
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	handler := gsserver.New(rootDir)
	gsserver.RegisterHandlersWithOptions(router, handler, gsserver.GinServerOptions{
		BaseURL: basePath,
	})

	tlsCfg, err := newMTLSConfig(tlsCertDir)
	if err != nil {
		log.Fatal().Err(err).Str("dir", tlsCertDir).Msg("failed to configure mTLS")
	}

	server := &http.Server{
		Addr:      serverAddr,
		Handler:   router,
		TLSConfig: tlsCfg,
	}
	log.Info().
		Str("serverAddr", serverAddr).
		Str("basePath", basePath).
		Str("rootDir", rootDir).
		Str("tlsCertDir", tlsCertDir).
		Msg("starting storage server with mTLS...")

	// Server TLSConfig already has cert/key, so pass empty strings here
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatal().Err(err).Msg("storage server exited with error")
	}
}

func newMTLSConfig(dir string) (*tls.Config, error) {
	caCertPEM, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}
	cert, err := tls.LoadX509KeyPair(
		filepath.Join(dir, "tls.crt"),
		filepath.Join(dir, "tls.key"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load server cert/key: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    clientCAs,
		ClientAuth:   tls.RequireAndVerifyClientCert, // Enforce mTLS
	}, nil
}
