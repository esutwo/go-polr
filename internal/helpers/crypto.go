package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// Password validation errors
var (
	ErrPasswordTooShort   = errors.New("password must be at least 12 characters")
	ErrPasswordNoUpper    = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower    = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit    = errors.New("password must contain at least one digit")
	ErrPasswordNoSpecial  = errors.New("password must contain at least one special character")
)

const (
	// DefaultBcryptCost is the default bcrypt cost factor
	DefaultBcryptCost = 10
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	return string(bytes), err
}

// CheckPassword compares a password with a bcrypt hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSecretKey generates a random secret key for link protection
func GenerateSecretKey(length int) (string, error) {
	if length <= 0 {
		length = 16
	}
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// GenerateAPIKey generates a random API key for user authentication
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashAPIKey creates a SHA-256 hash of an API key for secure storage
// We use SHA-256 instead of bcrypt because:
// 1. API keys are high-entropy random strings (not user-chosen passwords)
// 2. We need fast lookups for every API request
// 3. The key is 64 hex chars (256 bits of entropy), immune to brute force
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// CheckAPIKeyHash compares an API key with its hash using constant-time comparison
func CheckAPIKeyHash(apiKey, hash string) bool {
	computed := HashAPIKey(apiKey)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

// GenerateRecoveryKey generates a recovery key for account recovery
func GenerateRecoveryKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashURL generates a CRC32 hash of a URL for duplicate detection
// Returns a 10-character hexadecimal string (matching Polr's long_url_hash field)
func HashURL(url string) string {
	hash := crc32.ChecksumIEEE([]byte(url))
	return hex.EncodeToString([]byte{
		byte(hash >> 24),
		byte(hash >> 16),
		byte(hash >> 8),
		byte(hash),
	})
}

// ValidatePasswordStrength validates password meets security requirements:
// - At least 12 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one digit
// - At least one special character
func ValidatePasswordStrength(password string) error {
	if len(password) < 12 {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}
	if !hasSpecial {
		return ErrPasswordNoSpecial
	}

	return nil
}
