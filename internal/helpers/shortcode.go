package helpers

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const (
	// Base62Charset contains alphanumeric characters for URL-safe short codes
	Base62Charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// DefaultShortCodeLength is the default length for generated short codes
	DefaultShortCodeLength = 6
)

// GenerateShortCode generates a random short code using Base62 encoding
func GenerateShortCode(length int) (string, error) {
	if length <= 0 {
		length = DefaultShortCodeLength
	}

	charsetLen := big.NewInt(int64(len(Base62Charset)))
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = Base62Charset[num.Int64()]
	}

	return string(result), nil
}

// EncodeBase62 encodes an integer to a Base62 string
func EncodeBase62(num uint64) string {
	if num == 0 {
		return string(Base62Charset[0])
	}

	var result strings.Builder
	base := uint64(len(Base62Charset))

	for num > 0 {
		result.WriteByte(Base62Charset[num%base])
		num /= base
	}

	// Reverse the string
	s := result.String()
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// DecodeBase62 decodes a Base62 string to an integer
func DecodeBase62(s string) uint64 {
	var result uint64
	base := uint64(len(Base62Charset))

	for _, c := range s {
		result *= base
		result += uint64(strings.IndexRune(Base62Charset, c))
	}

	return result
}

// IsValidShortCode checks if a string is a valid short code (alphanumeric only)
func IsValidShortCode(code string) bool {
	if code == "" {
		return false
	}

	for _, c := range code {
		if strings.IndexRune(Base62Charset, c) == -1 {
			return false
		}
	}

	return true
}

// SanitizeShortCode removes any invalid characters from a short code
func SanitizeShortCode(code string) string {
	var result strings.Builder

	for _, c := range code {
		if strings.IndexRune(Base62Charset, c) != -1 {
			result.WriteRune(c)
		}
	}

	return result.String()
}
