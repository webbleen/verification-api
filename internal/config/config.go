package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server configuration
	Port string
	Mode string

	// Database configuration
	DatabaseURL string

	// Redis configuration
	RedisURL string

	// Brevo email configuration
	BrevoAPIKey    string
	BrevoFromEmail string

	// Verification code configuration
	CodeExpireMinutes int
	RateLimitMinutes  int
	ServiceName       string
}

var AppConfig *Config

func InitConfig() error {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Ignore error if .env file doesn't exist
	}

	AppConfig = &Config{
		Port:              getEnv("PORT", "8080"),
		Mode:              getEnv("GIN_MODE", "debug"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		BrevoAPIKey:       getEnv("BREVO_API_KEY", ""),
		BrevoFromEmail:    getEnv("BREVO_FROM_EMAIL", ""),
		CodeExpireMinutes: getEnvInt("CODE_EXPIRE_MINUTES", 5),
		RateLimitMinutes:  getEnvInt("RATE_LIMIT_MINUTES", 1),
		ServiceName:       getEnv("SERVICE_NAME", "Verification Service"),
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
