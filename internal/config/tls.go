package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog/log"
)

const (
	mtlsLoadTimeout  = 1 * time.Minute
	mtlsCertCacheTTL = 1 * time.Hour // assumes the time between renewal and expiry is above this value
)

func MutualTLS() *tls.Config {
	ctx, cancel := context.WithTimeout(context.Background(), mtlsLoadTimeout)
	defer cancel()

	dir := mutualTLSDir()
	log.Info().Str("dir", dir).Msg("Loading mTLS config from dir")

	var tlsConfig *tls.Config
	retryDelay := 1 * time.Second
	for {
		if cfg, err := loadMTLSConfig(dir); err != nil {
			log.Err(err).Msg("Failed to load tls config")
		} else {
			tlsConfig = cfg
			break
		}
		select {
		case <-ctx.Done():
			panic("deadline exceeded loading mtls config")
		default:
			time.Sleep(retryDelay)
			retryDelay *= 2
		}
	}

	// Get the cached TLS config for the client, allowing for key/cert
	// rotation without reloading the server
	cache := expirable.NewLRU[string, *tls.Config](1, nil, mtlsCertCacheTTL)
	tlsConfig.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
		if cfg, exists := cache.Get(dir); exists {
			return cfg, nil
		}
		cfg, err := loadMTLSConfig(dir)
		if err != nil {
			return nil, err
		}
		cache.Add(dir, cfg)
		return cfg, nil
	}

	return tlsConfig
}

// loadMTLSConfig reads the server cert/key and the CA cert from dir and
// returns a fully populated tls.Config.
// {ca.crt. tls.crt. tls.key} must all exist in the directory
func loadMTLSConfig(dir string) (*tls.Config, error) {
	log.Debug().Str("dir", dir).Msg("Reloading TLS certs")
	caCertPEM, err := os.ReadFile(
		filepath.Join(dir, "ca.crt"),
	) // #nosec G304 // cert dir is from trusted config
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
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}
