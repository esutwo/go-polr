package helpers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Hash should start with bcrypt prefix
	assert.True(t, strings.HasPrefix(hash, "$2a$"))
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	// Correct password should match
	assert.True(t, CheckPassword(password, hash))

	// Wrong password should not match
	assert.False(t, CheckPassword("wrongpassword", hash))
}

func TestGenerateSecretKey(t *testing.T) {
	key1, err := GenerateSecretKey(16)
	require.NoError(t, err)
	assert.Len(t, key1, 16)

	key2, err := GenerateSecretKey(16)
	require.NoError(t, err)

	// Keys should be different
	assert.NotEqual(t, key1, key2)

	// Default length
	key3, err := GenerateSecretKey(0)
	require.NoError(t, err)
	assert.Len(t, key3, 16)
}

func TestGenerateAPIKey(t *testing.T) {
	key1, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.Len(t, key1, 64) // 32 bytes = 64 hex chars

	key2, err := GenerateAPIKey()
	require.NoError(t, err)

	// Keys should be different
	assert.NotEqual(t, key1, key2)
}

func TestHashAPIKey(t *testing.T) {
	apiKey := "test-api-key-12345"

	hash1 := HashAPIKey(apiKey)
	assert.Len(t, hash1, 64) // SHA-256 = 32 bytes = 64 hex chars

	// Same key should produce same hash
	hash2 := HashAPIKey(apiKey)
	assert.Equal(t, hash1, hash2)

	// Different key should produce different hash
	hash3 := HashAPIKey("different-key")
	assert.NotEqual(t, hash1, hash3)
}

func TestCheckAPIKeyHash(t *testing.T) {
	apiKey := "test-api-key-12345"
	hash := HashAPIKey(apiKey)

	// Correct key should match
	assert.True(t, CheckAPIKeyHash(apiKey, hash))

	// Wrong key should not match
	assert.False(t, CheckAPIKeyHash("wrong-key", hash))

	// Empty key should not match
	assert.False(t, CheckAPIKeyHash("", hash))
}

func TestGenerateRecoveryKey(t *testing.T) {
	key1, err := GenerateRecoveryKey()
	require.NoError(t, err)
	assert.Len(t, key1, 32) // 16 bytes = 32 hex chars

	key2, err := GenerateRecoveryKey()
	require.NoError(t, err)

	// Keys should be different
	assert.NotEqual(t, key1, key2)
}

func TestHashURL(t *testing.T) {
	url1 := "https://example.com/path"
	url2 := "https://different.com/path"

	hash1 := HashURL(url1)
	hash2 := HashURL(url2)

	assert.Len(t, hash1, 8) // CRC32 = 4 bytes = 8 hex chars
	assert.NotEqual(t, hash1, hash2)

	// Same URL should produce same hash
	hash1Again := HashURL(url1)
	assert.Equal(t, hash1, hash1Again)
}

func TestGenerateShortCode(t *testing.T) {
	code1, err := GenerateShortCode(6)
	require.NoError(t, err)
	assert.Len(t, code1, 6)

	code2, err := GenerateShortCode(6)
	require.NoError(t, err)

	// Codes should be different
	assert.NotEqual(t, code1, code2)

	// All characters should be valid Base62
	assert.True(t, IsValidShortCode(code1))

	// Default length
	code3, err := GenerateShortCode(0)
	require.NoError(t, err)
	assert.Len(t, code3, DefaultShortCodeLength)
}

func TestEncodeBase62(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{61, "z"},
		{62, "10"},
		{123456, "W7E"},
	}

	for _, tt := range tests {
		result := EncodeBase62(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestDecodeBase62(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"0", 0},
		{"1", 1},
		{"z", 61},
		{"10", 62},
		{"W7E", 123456},
	}

	for _, tt := range tests {
		result := DecodeBase62(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestBase62RoundTrip(t *testing.T) {
	// Test that encode/decode are inverses
	values := []uint64{0, 1, 100, 1000, 123456789, 9999999999}

	for _, v := range values {
		encoded := EncodeBase62(v)
		decoded := DecodeBase62(encoded)
		assert.Equal(t, v, decoded)
	}
}

func TestIsValidShortCode(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"abc123", true},
		{"ABC", true},
		{"0123456789", true},
		{"abc-123", false}, // dash not allowed
		{"abc_123", false}, // underscore not allowed
		{"abc 123", false}, // space not allowed
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidShortCode(tt.code))
		})
	}
}

func TestSanitizeShortCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc123", "abc123"},
		{"abc-123", "abc123"},
		{"abc_123", "abc123"},
		{"abc 123", "abc123"},
		{"!@#$%abc", "abc"},
	}

	for _, tt := range tests {
		result := SanitizeShortCode(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
