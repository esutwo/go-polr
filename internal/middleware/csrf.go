package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	// CSRFTokenKey is the session key for the CSRF token
	CSRFTokenKey = "csrf_token"
	// CSRFTokenLength is the length of the CSRF token in bytes
	CSRFTokenLength = 32
)

// CSRF middleware protects against cross-site request forgery
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		// Generate or retrieve CSRF token
		token := session.Get(CSRFTokenKey)
		if token == nil {
			newToken, err := generateCSRFToken()
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			session.Set(CSRFTokenKey, newToken)
			session.Save()
			token = newToken
		}

		// Store token in context for templates
		c.Set("csrf_token", token)

		// For GET, HEAD, OPTIONS - just continue
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// For POST, PUT, DELETE - validate token
		submittedToken := c.PostForm("_csrf")
		if submittedToken == "" {
			submittedToken = c.GetHeader("X-CSRF-Token")
		}

		sessionToken, ok := token.(string)
		if !ok || !validateCSRFToken(sessionToken, submittedToken) {
			c.HTML(http.StatusForbidden, "error.html", gin.H{
				"title":   "Forbidden",
				"message": "Invalid CSRF token. Please try again.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// generateCSRFToken generates a secure random CSRF token
func generateCSRFToken() (string, error) {
	b := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// validateCSRFToken performs constant-time comparison of tokens
func validateCSRFToken(sessionToken, submittedToken string) bool {
	if sessionToken == "" || submittedToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(sessionToken), []byte(submittedToken)) == 1
}

// GetCSRFToken retrieves the CSRF token from the context
func GetCSRFToken(c *gin.Context) string {
	token, exists := c.Get("csrf_token")
	if !exists {
		return ""
	}
	t, ok := token.(string)
	if !ok {
		return ""
	}
	return t
}
