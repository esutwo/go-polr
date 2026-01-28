package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"valid http", "http://example.com", true},
		{"valid https", "https://example.com", true},
		{"valid with path", "https://example.com/path/to/page", true},
		{"valid with query", "https://example.com?foo=bar", true},
		{"missing scheme", "example.com", false},
		{"missing host", "http://", false},
		{"empty string", "", false},
		{"just path", "/path/to/page", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
