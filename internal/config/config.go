package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment         string
	Port                string
	DatabaseURL         string
	Issuer              string
	PrivateKeyPEM       string
	AccessTokenTTLMin   int
	RefreshTokenTTLDays int
}

func Load() (Config, error) {
	_ = godotenv.Load()
	cfg := Config{
		Environment:         value("ENVIRONMENT", "development"),
		Port:                value("PORT", "8081"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		Issuer:              value("AUTH_ISSUER", "http://localhost:8081"),
		PrivateKeyPEM:       strings.ReplaceAll(os.Getenv("AUTH_PRIVATE_KEY_PEM"), "\\n", "\n"),
		AccessTokenTTLMin:   intValue("ACCESS_TOKEN_TTL_MINUTES", 15),
		RefreshTokenTTLDays: intValue("REFRESH_TOKEN_TTL_DAYS", 30),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Environment != "development" && cfg.PrivateKeyPEM == "" {
		return Config{}, fmt.Errorf("AUTH_PRIVATE_KEY_PEM is required outside development")
	}
	if cfg.AccessTokenTTLMin < 1 || cfg.RefreshTokenTTLDays < 1 {
		return Config{}, fmt.Errorf("token durations must be positive")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if current := os.Getenv(key); current != "" {
		return current
	}
	return fallback
}

func intValue(key string, fallback int) int {
	if current, err := strconv.Atoi(os.Getenv(key)); err == nil && current > 0 {
		return current
	}
	return fallback
}
