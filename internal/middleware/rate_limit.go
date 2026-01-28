package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*userRequests
	window   time.Duration
}

type userRequests struct {
	count     int
	windowEnd time.Time
}

// NewRateLimiter creates a new rate limiter with the given window duration
func NewRateLimiter(window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*userRequests),
		window:   window,
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

// cleanup periodically removes expired entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, req := range rl.requests {
			if now.After(req.windowEnd) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed and increments the counter
func (rl *RateLimiter) Allow(key string, quota int) (allowed bool, remaining int, resetAt time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	req, exists := rl.requests[key]

	if !exists || now.After(req.windowEnd) {
		// New window
		rl.requests[key] = &userRequests{
			count:     1,
			windowEnd: now.Add(rl.window),
		}
		return true, quota - 1, now.Add(rl.window)
	}

	if req.count >= quota {
		return false, 0, req.windowEnd
	}

	req.count++
	return true, quota - req.count, req.windowEnd
}

// APIRateLimit middleware enforces API rate limits based on user quota
func APIRateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetAPIUser(c)
		if user == nil {
			c.Next()
			return
		}

		// Parse quota from user
		quota := 60 // default
		if user.APIQuota != "" {
			if q, err := strconv.Atoi(user.APIQuota); err == nil && q > 0 {
				quota = q
			}
		}

		// Use both username and IP as the rate limit key to prevent bypass via multiple IPs
		key := user.Username + ":" + c.ClientIP()
		allowed, remaining, resetAt := limiter.Allow(key, quota)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(quota))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, APIError{
				Error:     "Rate limit exceeded. Please try again later.",
				ErrorCode: ErrCodeRateLimited,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
