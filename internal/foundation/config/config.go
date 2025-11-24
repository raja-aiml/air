package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Port                string
	DatabaseURL         string
	LogLevel            string
	JWTSecret           string
	OTELEndpoint        string
	PrometheusNamespace string
	OpenAIKey           string
}

// BackfillConfig holds backfill configuration
type BackfillConfig struct {
	DatabaseURL string
	OpenAIKey   string
	LogLevel    string
}

// JWTGenConfig holds JWT generation configuration
type JWTGenConfig struct {
	Subject    string
	Issuer     string
	Audience   string
	Secret     string
	ExpMinutes int
}

// LoadServerConfig loads server configuration from environment
func LoadServerConfig() (*ServerConfig, error) {
	_ = godotenv.Load()

	cfg := &ServerConfig{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		OTELEndpoint:        getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		PrometheusNamespace: getEnv("PROMETHEUS_NAMESPACE", "skillflow"),
		OpenAIKey:           os.Getenv("OPENAI_API_KEY"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// LoadBackfillConfig loads backfill configuration from environment
func LoadBackfillConfig() (*BackfillConfig, error) {
	_ = godotenv.Load()

	cfg := &BackfillConfig{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.OpenAIKey == "" {
		log.Warn().Msg("OPENAI_API_KEY not set")
	}

	return cfg, nil
}

// LoadJWTGenConfig loads JWT generation configuration from flags/env
func LoadJWTGenConfig(subject, issuer, audience, secret string, expMinutes int) (*JWTGenConfig, error) {
	_ = godotenv.Load()

	cfg := &JWTGenConfig{
		Subject:    subject,
		Issuer:     issuer,
		Audience:   audience,
		Secret:     secret,
		ExpMinutes: expMinutes,
	}

	if cfg.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ParseLogLevel converts string log level to zerolog level
func ParseLogLevel(level string) string {
	level = strings.ToLower(level)
	switch level {
	case "debug", "info", "warn", "error":
		return level
	default:
		return "info"
	}
}

// ParseInt parses string to int with default
func ParseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}
