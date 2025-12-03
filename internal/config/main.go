package config

import (
	"fmt"
	"os"
	"time"
)

const (
	BaseURL                = "/v0"
	ServerShutdownDuration = 30 * time.Second
	ReadHeaderTimeout      = 1 * time.Second
)

// Server address e.g. ":8080""
func ServerAddress() string {
	return fmt.Sprintf(":%s", envOrDefault("PORT", "8080"))
}

func envOrDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value == "" {
		return defaultValue
	} else {
		return value
	}
}
