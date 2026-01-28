package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders middleware adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS filter in browsers
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy - restrict resource loading
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		// Permissions Policy - restrict browser features
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// HTTPSRedirect middleware redirects HTTP requests to HTTPS
// Only enabled when ENFORCE_HTTPS environment variable is set to "true"
func HTTPSRedirect() gin.HandlerFunc {
	enforceHTTPS := os.Getenv("ENFORCE_HTTPS") == "true"

	return func(c *gin.Context) {
		if !enforceHTTPS {
			c.Next()
			return
		}

		// Check X-Forwarded-Proto header (common when behind a reverse proxy)
		proto := c.GetHeader("X-Forwarded-Proto")
		if proto == "" {
			proto = c.Request.URL.Scheme
		}

		// If not HTTPS, redirect
		if proto != "https" && c.Request.TLS == nil {
			httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(301, httpsURL)
			c.Abort()
			return
		}

		// Add HSTS header for HTTPS connections
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}
