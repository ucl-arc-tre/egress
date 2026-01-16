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
)

const (
	configPath  = "/etc/egress/config.yaml"
	defaultPort = "8080"

	BaseURL                = "/v0"
	ServerShutdownDuration = 30 * time.Second
	ReadHeaderTimeout      = 1 * time.Second
)

var k *koanf.Koanf

// Initialise config
func Init() {
	initWithPath(configPath)
}

// Initialise config from given path
func initWithPath(path string) {
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

func S3Credentials() S3CredentialBundle {
	return S3CredentialBundle{
		Region:          k.String("s3.region"),
		AccessKeyId:     k.String("s3.access_key_id"),
		SecretAccessKey: k.String("s3.secret_access_key"),
	}
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
