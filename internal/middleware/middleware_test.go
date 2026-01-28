package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(time.Minute)

	// First request should be allowed
	allowed, remaining, _ := limiter.Allow("user1", 5)
	assert.True(t, allowed)
	assert.Equal(t, 4, remaining)

	// Second request should be allowed
	allowed, remaining, _ = limiter.Allow("user1", 5)
	assert.True(t, allowed)
	assert.Equal(t, 3, remaining)

	// Use up remaining quota
	limiter.Allow("user1", 5)
	limiter.Allow("user1", 5)
	allowed, remaining, _ = limiter.Allow("user1", 5)
	assert.True(t, allowed)
	assert.Equal(t, 0, remaining)

	// Should be rate limited
	allowed, remaining, _ = limiter.Allow("user1", 5)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)

	// Different user should have separate quota
	allowed, remaining, _ = limiter.Allow("user2", 5)
	assert.True(t, allowed)
	assert.Equal(t, 4, remaining)
}

func TestRateLimiter_WindowReset(t *testing.T) {
	limiter := NewRateLimiter(100 * time.Millisecond)

	// Use up quota
	for i := 0; i < 3; i++ {
		limiter.Allow("user1", 3)
	}

	// Should be rate limited
	allowed, _, _ := limiter.Allow("user1", 3)
	assert.False(t, allowed)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, remaining, _ := limiter.Allow("user1", 3)
	assert.True(t, allowed)
	assert.Equal(t, 2, remaining)
}

func TestCreateAnonymousUser(t *testing.T) {
	user := createAnonymousUser("192.168.1.1")

	assert.Equal(t, "ANONIP:192.168.1.1", user.Username)
	assert.Equal(t, "user", user.Role)
	assert.Equal(t, "1", user.Active)
	assert.Equal(t, "60", user.APIQuota)
}
