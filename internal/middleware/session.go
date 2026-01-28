package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/models"
	"gorm.io/gorm"
)

const (
	// SessionKeyUserID is the session key for the user ID
	SessionKeyUserID = "user_id"
	// SessionKeyUsername is the session key for the username
	SessionKeyUsername = "username"
	// SessionKeyRole is the session key for the user role
	SessionKeyRole = "role"
	// ContextKeyUser is the context key for the authenticated user
	ContextKeyUser = "user"
)

// SessionAuth middleware checks for a valid session and loads the user
func SessionAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get(SessionKeyUserID)

		if userID == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Load user from database
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			// User not found or deleted, clear session
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Check if user is active
		if !user.IsActive() {
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, "/login?error=inactive")
			c.Abort()
			return
		}

		// Store user in context for handlers
		c.Set(ContextKeyUser, &user)
		c.Next()
	}
}

// AdminOnly middleware checks if the authenticated user is an admin
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get(ContextKeyUser)
		if !exists {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		u, ok := user.(*models.User)
		if !ok || !u.IsAdmin() {
			c.HTML(http.StatusForbidden, "pages/error.html", gin.H{
				"title":   "Access Denied",
				"message": "You do not have permission to access this page.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalSessionAuth middleware loads the user if logged in, but doesn't require auth
func OptionalSessionAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get(SessionKeyUserID)

		if userID != nil {
			var user models.User
			if err := db.First(&user, userID).Error; err == nil && user.IsActive() {
				c.Set(ContextKeyUser, &user)
			}
		}

		c.Next()
	}
}

// SetSession sets the session values for an authenticated user
func SetSession(c *gin.Context, user *models.User) error {
	session := sessions.Default(c)
	session.Set(SessionKeyUserID, user.ID)
	session.Set(SessionKeyUsername, user.Username)
	session.Set(SessionKeyRole, user.Role)
	return session.Save()
}

// ClearSession clears the session (logout)
func ClearSession(c *gin.Context) error {
	session := sessions.Default(c)
	session.Clear()
	return session.Save()
}

// GetCurrentUser retrieves the authenticated user from context
func GetCurrentUser(c *gin.Context) *models.User {
	user, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil
	}
	u, ok := user.(*models.User)
	if !ok {
		return nil
	}
	return u
}

// IsLoggedIn returns true if there is an authenticated user
func IsLoggedIn(c *gin.Context) bool {
	return GetCurrentUser(c) != nil
}
