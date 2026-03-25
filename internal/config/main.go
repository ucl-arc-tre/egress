package config

import (
	"fmt"
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
	configPath  = "/etc/egress/config.yaml"
	tlsCertDir  = "/etc/egress/tls"
	defaultPort = "8080"

	BaseURL                = "/v0"
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
		log.Err(err).Msg("error loading config")
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if IsDebug() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// Server address e.g. ":8080""
// Load from env to match with Gin
func ServerAddress() string {
	return fmt.Sprintf(":%s", envOrDefault("PORT", defaultPort))
}

func IsDebug() bool {
	return k.Bool("debug")
}

func StorageConfig() StorageConfigBundle {
	provider := k.String("storage.provider")
	cfg := StorageConfigBundle{
		Provider:   provider,
		TLSCertDir: tlsCertDir,
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

func AuthBasicCredentials() AuthBasicCredentialsBundle {
	return AuthBasicCredentialsBundle{
		Username: k.String("auth.basic.username"),
		Password: k.String("auth.basic.password"),
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

func envOrDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value == "" {
		return defaultValue
	} else {
		return value
	}
}
