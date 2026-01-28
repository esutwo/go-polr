package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("APP_NAME")
	os.Unsetenv("APP_PORT")
	os.Unsetenv("DB_HOST")

	// Enable dev mode to allow default secrets in tests
	os.Setenv("DEV_MODE", "true")
	defer os.Unsetenv("DEV_MODE")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "go-polr", cfg.AppName)
	assert.Equal(t, "8080", cfg.AppPort)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "3306", cfg.DBPort)
	assert.False(t, cfg.AnonAPIEnabled)
	assert.True(t, cfg.RegistrationEnabled)
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("APP_NAME", "test-polr")
	os.Setenv("APP_PORT", "9000")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("ANON_API_ENABLED", "true")
	os.Setenv("DEV_MODE", "true") // Allow default secrets in tests
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_PORT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("ANON_API_ENABLED")
		os.Unsetenv("DEV_MODE")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "test-polr", cfg.AppName)
	assert.Equal(t, "9000", cfg.AppPort)
	assert.Equal(t, "db.example.com", cfg.DBHost)
	assert.True(t, cfg.AnonAPIEnabled)
}

func TestDSN(t *testing.T) {
	cfg := &Config{
		DBUser:     "testuser",
		DBPassword: "testpass",
		DBHost:     "localhost",
		DBPort:     "3306",
		DBName:     "testdb",
	}

	expected := "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	assert.Equal(t, expected, cfg.DSN())
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		defValue bool
		expected bool
	}{
		{"true string", "true", false, true},
		{"false string", "false", true, false},
		{"1 string", "1", false, true},
		{"0 string", "0", true, false},
		{"invalid uses default", "invalid", true, true},
		{"empty uses default", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_BOOL", tt.envValue)
				defer os.Unsetenv("TEST_BOOL")
			} else {
				os.Unsetenv("TEST_BOOL")
			}
			result := getEnvBool("TEST_BOOL", tt.defValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
