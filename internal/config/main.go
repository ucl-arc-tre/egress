package config

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const (
	BaseURL                = "/v0"
	ServerShutdownDuration = 30 * time.Second
	ReadHeaderTimeout      = 1 * time.Second
)

// Initalise config
func Init() {
	if envOrDefault("DEBUG", "false") == "true" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// Server address e.g. ":8080""
func ServerAddress() string {
	return fmt.Sprintf(":%s", envOrDefault("PORT", "8080"))
}

func DevS3URL() string {
	return "http://rustfs-svc.rustfs.svc.cluster.local:9000"
}

func DevS3Bucket() string {
	return "bucket1"
}

func S3Credentials() S3CredentialBundle {
	return S3CredentialBundle{
		Region:          "us-east-1",
		AccessKeyId:     "rustfsadmin",
		SecretAccessKey: "rustfsadmin",
	}
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
