package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/helpers"
	"github.com/nnnc-org/go-polr/internal/models"
	"gorm.io/gorm"
)

const (
	// ContextKeyAPIUser is the context key for the API authenticated user
	ContextKeyAPIUser = "api_user"
	// ContextKeyIsAnonymous is the context key indicating anonymous API access
	ContextKeyIsAnonymous = "is_anonymous"
)

// APIError represents an API error response
type APIError struct {
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}

// API error codes matching Polr
const (
	ErrCodeAuthError     = "AUTH_ERROR"
	ErrCodeRateLimited   = "RATE_LIMIT_EXCEEDED"
	ErrCodeMissingParams = "MISSING_PARAMETERS"
	ErrCodeInvalidParams = "INVALID_PARAMETERS"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeCreationError = "CREATION_ERROR"
	ErrCodeAccessDenied  = "ACCESS_DENIED"
)

// APIAuth middleware authenticates API requests using API keys
func APIAuth(db *gorm.DB, allowAnonymous bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key in query parameter or header
		apiKey := c.Query("key")
		if apiKey == "" {
			apiKey = c.GetHeader("X-API-Key")
		}

		if apiKey == "" {
			// No API key provided
			if allowAnonymous {
				// Allow anonymous access with a pseudo-user
				c.Set(ContextKeyIsAnonymous, true)
				c.Set(ContextKeyAPIUser, createAnonymousUser(c.ClientIP()))
				c.Next()
				return
			}

			c.JSON(http.StatusUnauthorized, APIError{
				Error:     "Authentication required. Please provide an API key.",
				ErrorCode: ErrCodeAuthError,
			})
			c.Abort()
			return
		}

		// Hash the provided API key and look up by the hash
		hashedKey := helpers.HashAPIKey(apiKey)

		var user models.User
		err := db.Where("api_key = ? AND api_active = ? AND active = ?", hashedKey, true, "1").First(&user).Error
		if err != nil {
			c.JSON(http.StatusUnauthorized, APIError{
				Error:     "Invalid API key or API access is disabled.",
				ErrorCode: ErrCodeAuthError,
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set(ContextKeyIsAnonymous, false)
		c.Set(ContextKeyAPIUser, &user)
		c.Next()
	}
}

// GetAPIUser retrieves the authenticated API user from context
func GetAPIUser(c *gin.Context) *models.User {
	user, exists := c.Get(ContextKeyAPIUser)
	if !exists {
		return nil
	}
	u, ok := user.(*models.User)
	if !ok {
		return nil
	}
	return u
}

// IsAnonymousAPIUser returns true if the request is an anonymous API request
func IsAnonymousAPIUser(c *gin.Context) bool {
	isAnon, exists := c.Get(ContextKeyIsAnonymous)
	if !exists {
		return false
	}
	b, ok := isAnon.(bool)
	return ok && b
}

// createAnonymousUser creates a pseudo-user for anonymous API access
// The username includes the IP to ensure rate limiting is per-IP for anonymous users
func createAnonymousUser(ip string) *models.User {
	return &models.User{
		Username: "ANONIP:" + ip, // Include IP in username for per-IP rate limiting
		Role:     models.RoleUser,
		Active:   "1",
		APIQuota: "60", // Lower quota for anonymous users
	}
}

// RespondAPIError sends a JSON error response with the given status and error info
func RespondAPIError(c *gin.Context, status int, message string, errorCode string) {
	c.JSON(status, APIError{
		Error:     message,
		ErrorCode: errorCode,
	})
}
