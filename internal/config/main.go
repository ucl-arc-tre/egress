package config

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/types"
)

const (
	configPath       = "/etc/egress/config.yaml"
	defaultHttpPort  = "8000"
	defaultHttpsPort = "8443"

	BaseURL                = "/v1"
	ServerShutdownDuration = 30 * time.Second
	ReadHeaderTimeout      = 1 * time.Second
)

var k *koanf.Koanf

// Initialise config
func Init() {
	InitWithPath(configPath)
}

// Initialise config from given path
func InitWithPath(path string) {
	k = koanf.New(".")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		log.Fatal().Err(err).Msg("error loading config")
	}

	validateConfig()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if IsDebug() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// Server address e.g. ":8080""
func ServerAddress() string {
	var port string
	if MutualTLSEnabled() {
		port = envOrDefault("HTTPS_PORT", defaultHttpsPort)
	} else {
		port = envOrDefault("HTTP_PORT", defaultHttpPort)

	}
	return fmt.Sprintf(":%s", port)
}

func HttpPort() string {
	return envOrDefault("HTTP_PORT", defaultHttpPort)
}

func IsDebug() bool {
	return k.Bool("debug")
}

func MutualTLSEnabled() bool {
	return k.String("auth.mTLS.dir") != ""
}

func mutualTLSDir() string {
	return k.String("auth.mTLS.dir")
}

func StorageConfig() StorageConfigBundle {
	provider := k.String("storage.provider")
	cfg := StorageConfigBundle{
		Provider:   provider,
		TLSCertDir: k.String("storage.generic.mTLS.dir"),
	}
	if provider == string(types.StorageProviderS3) {
		cfg.S3 = S3StorageConfig{
			Region:          k.String("storage.s3.region"),
			AccessKeyId:     k.String("storage.s3.access_key_id"),
			SecretAccessKey: k.String("storage.s3.secret_access_key"),
		}
	}
	return cfg
}

func DBConfig() DBConfigBundle {
	provider := k.String("db.provider")
	cfg := DBConfigBundle{Provider: provider}

	if provider == string(types.DBProviderRqlite) {
		cfg.Rqlite = RqliteConfig{
			BaseURL:  k.String("db.rqlite.baseUrl"),
			Username: k.String("db.rqlite.username"),
			Password: k.String("db.rqlite.password"),
		}
	}
	return cfg
}

func BasicAuthConfig() BasicAuthConfigBundle {
	return BasicAuthConfigBundle{
		Username: k.String("auth.basic.username"),
		Password: k.String("auth.basic.password"),
	}
}

func BearerAuthConfig() BearerAuthConfigBundle {
	return BearerAuthConfigBundle{
		IssuerURL: k.String("auth.bearer.issuer_url"),
		Audience:  k.String("auth.bearer.audience"),
	}
}

func DevS3URL() string {
	return k.String("dev.s3.url")
}

func DevS3Bucket() string {
	return k.String("dev.s3.bucket")
}

func IsDevS3() bool {
	return DevS3URL() != ""
}

// Validate config; fail fast on invalid values
func validateConfig() {
	validateURL("db.rqlite.baseUrl")
	validateURL("auth.bearer.issuer_url")
}

func validateURL(key string) {
	if k.Exists(key) {
		value := k.String(key)
		if u, err := url.ParseRequestURI(value); err != nil || u.Scheme == "" || u.Host == "" {
			log.Fatal().Str(key, value).Msg(fmt.Sprintf("%s must be a valid absolute URL", key))
		}
	}
}

func envOrDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value == "" {
		return defaultValue
	} else {
		return value
	}
}
