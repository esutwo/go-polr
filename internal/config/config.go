package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// ErrDefaultSecrets is returned when default secrets are detected in production
var ErrDefaultSecrets = errors.New("default secrets detected: SESSION_SECRET and CSRF_SECRET must be changed from their default values")

// Config holds all configuration for the application
type Config struct {
	// Application
	AppName string
	AppURL  string
	AppPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Security
	SessionSecret string
	CSRFSecret    string

	// Features
	AnonAPIEnabled      bool
	RegistrationEnabled bool
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		// Application
		AppName: getEnv("APP_NAME", "go-polr"),
		AppURL:  getEnv("APP_URL", "http://localhost:8080"),
		AppPort: getEnv("APP_PORT", "8080"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "polr"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "polrdb"),

		// Security
		SessionSecret: getEnv("SESSION_SECRET", "change-me-in-production-32chars"),
		CSRFSecret:    getEnv("CSRF_SECRET", "change-me-csrf-secret-32chars"),

		// Features
		AnonAPIEnabled:      getEnvBool("ANON_API_ENABLED", false),
		RegistrationEnabled: getEnvBool("REGISTRATION_ENABLED", true),
	}

	// Security check: reject default secrets in production
	// Only allow defaults if explicitly in development mode
	if !getEnvBool("DEV_MODE", false) {
		if cfg.SessionSecret == "change-me-in-production-32chars" ||
			cfg.CSRFSecret == "change-me-csrf-secret-32chars" {
			return nil, ErrDefaultSecrets
		}
	}

	return cfg, nil
}

// DSN returns the MySQL data source name for GORM
func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return b
	}
	return defaultValue
}
