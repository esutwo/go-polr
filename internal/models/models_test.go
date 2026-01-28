package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"admin role", "admin", true},
		{"user role", "user", false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Role: tt.role}
			assert.Equal(t, tt.expected, u.IsAdmin())
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		active   string
		expected bool
	}{
		{"active 1", "1", true},
		{"active true", "true", true},
		{"inactive 0", "0", false},
		{"inactive false", "false", false},
		{"inactive empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Active: tt.active}
			assert.Equal(t, tt.expected, u.IsActive())
		})
	}
}

func TestUser_HasAPIAccess(t *testing.T) {
	apiKey := "test-api-key"
	emptyKey := ""

	tests := []struct {
		name      string
		apiActive bool
		apiKey    *string
		expected  bool
	}{
		{"has access", true, &apiKey, true},
		{"api disabled", false, &apiKey, false},
		{"no key", true, nil, false},
		{"empty key", true, &emptyKey, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{APIActive: tt.apiActive, APIKey: tt.apiKey}
			assert.Equal(t, tt.expected, u.HasAPIAccess())
		})
	}
}

func TestLink_IsSecret(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		expected  bool
	}{
		{"has secret", "secret123", true},
		{"no secret", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Link{SecretKey: tt.secretKey}
			assert.Equal(t, tt.expected, l.IsSecret())
		})
	}
}

func TestLink_CanAccess(t *testing.T) {
	tests := []struct {
		name        string
		secretKey   string
		providedKey string
		expected    bool
	}{
		{"no secret needed", "", "", true},
		{"no secret any key", "", "anykey", true},
		{"correct key", "secret123", "secret123", true},
		{"wrong key", "secret123", "wrongkey", false},
		{"missing key", "secret123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Link{SecretKey: tt.secretKey}
			assert.Equal(t, tt.expected, l.CanAccess(tt.providedKey))
		})
	}
}

func TestLink_IncrementClicks(t *testing.T) {
	l := &Link{Clicks: 5}
	l.IncrementClicks()
	assert.Equal(t, 6, l.Clicks)
}

func TestNewClick(t *testing.T) {
	country := "US"
	referer := "https://google.com"
	refererHost := "google.com"
	userAgent := "Mozilla/5.0"

	click := NewClick(1, "192.168.1.1", &referer, &refererHost, &userAgent, &country)

	assert.Equal(t, uint(1), click.LinkID)
	assert.Equal(t, "192.168.1.1", click.IP)
	assert.Equal(t, &referer, click.Referer)
	assert.Equal(t, &refererHost, click.RefererHost)
	assert.Equal(t, &userAgent, click.UserAgent)
	assert.Equal(t, &country, click.Country)
}
