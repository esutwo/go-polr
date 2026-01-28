package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/config"
	"github.com/nnnc-org/go-polr/internal/helpers"
	"github.com/nnnc-org/go-polr/internal/middleware"
	"github.com/nnnc-org/go-polr/internal/services"
)

// UserHandler handles user-related web requests
type UserHandler struct {
	userService *services.UserService
	linkService *services.LinkService
	statsService *services.StatsService
	config      *config.Config
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *services.UserService, linkService *services.LinkService, statsService *services.StatsService, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService: userService,
		linkService: linkService,
		statsService: statsService,
		config:      cfg,
	}
}

// validLoginErrors maps error codes to user-friendly messages
var validLoginErrors = map[string]string{
	"invalid":  "Invalid username or password",
	"inactive": "Your account is inactive",
	"session":  "Session expired, please log in again",
}

// LoginPage renders the login page
func (h *UserHandler) LoginPage(c *gin.Context) {
	if middleware.IsLoggedIn(c) {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	// Use whitelist for error messages to prevent XSS
	errorCode := c.Query("error")
	errorMsg := validLoginErrors[errorCode]

	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":      "Login - " + h.config.AppName,
		"error":      errorMsg,
		"csrf_token": middleware.GetCSRFToken(c),
	})
}

// LoginRequest represents the login form data
type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

// Login handles user login
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"title": "Login - " + h.config.AppName,
			"error": "Please provide username and password",
		})
		return
	}

	user, err := h.userService.Authenticate(req.Username, req.Password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title":    "Login - " + h.config.AppName,
			"error":    "Invalid username or password",
			"username": req.Username,
		})
		return
	}

	if err := middleware.SetSession(c, user); err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"title": "Login - " + h.config.AppName,
			"error": "Failed to create session",
		})
		return
	}

	c.Redirect(http.StatusFound, "/")
}

// Logout handles user logout
func (h *UserHandler) Logout(c *gin.Context) {
	middleware.ClearSession(c)
	c.Redirect(http.StatusFound, "/")
}

// RegisterPage renders the registration page
func (h *UserHandler) RegisterPage(c *gin.Context) {
	if !h.config.RegistrationEnabled {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"title":   "Registration Disabled",
			"message": "Registration is currently disabled.",
		})
		return
	}

	if middleware.IsLoggedIn(c) {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	c.HTML(http.StatusOK, "register.html", gin.H{
		"title":      "Register - " + h.config.AppName,
		"csrf_token": middleware.GetCSRFToken(c),
	})
}

// RegisterRequest represents the registration form data
type RegisterRequest struct {
	Username        string `form:"username" binding:"required"`
	Email           string `form:"email" binding:"required,email"`
	Password        string `form:"password" binding:"required,min=12"`
	PasswordConfirm string `form:"password_confirm" binding:"required"`
}

// Register handles user registration
func (h *UserHandler) Register(c *gin.Context) {
	if !h.config.RegistrationEnabled {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"title":   "Registration Disabled",
			"message": "Registration is currently disabled.",
		})
		return
	}

	var req RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title": "Register - " + h.config.AppName,
			"error": "Please fill in all fields correctly",
		})
		return
	}

	if req.Password != req.PasswordConfirm {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Register - " + h.config.AppName,
			"error":    "Passwords do not match",
			"username": req.Username,
			"email":    req.Email,
		})
		return
	}

	// Validate password strength
	if err := helpers.ValidatePasswordStrength(req.Password); err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":      "Register - " + h.config.AppName,
			"error":      err.Error(),
			"username":   req.Username,
			"email":      req.Email,
			"csrf_token": middleware.GetCSRFToken(c),
		})
		return
	}

	user, err := h.userService.Create(services.CreateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IP:       c.ClientIP(),
	})

	if err != nil {
		errorMsg := "Failed to create account"
		switch err {
		case services.ErrUsernameTaken:
			errorMsg = "Username is already taken"
		case services.ErrEmailTaken:
			errorMsg = "Email is already registered"
		}

		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title":    "Register - " + h.config.AppName,
			"error":    errorMsg,
			"username": req.Username,
			"email":    req.Email,
		})
		return
	}

	// Auto-login after registration
	if err := middleware.SetSession(c, user); err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	c.Redirect(http.StatusFound, "/dashboard")
}

// Dashboard renders the user dashboard
func (h *UserHandler) Dashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Get user's links
	links, totalLinks, _ := h.linkService.GetLinksByCreator(user.Username, 10, 0)

	// Get user stats
	stats, _ := h.statsService.GetUserStats(user.Username)

	// Check for newly generated API key (flash message - shown only once)
	var newAPIKey string
	session := sessions.Default(c)
	if key := session.Get("new_api_key"); key != nil {
		newAPIKey = key.(string)
		session.Delete("new_api_key")
		session.Save()
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":      "Dashboard - " + h.config.AppName,
		"user":       user,
		"links":      links,
		"totalLinks": totalLinks,
		"stats":      stats,
		"appURL":     h.config.AppURL,
		"csrf_token": middleware.GetCSRFToken(c),
		"newAPIKey":  newAPIKey,
	})
}

// ChangePasswordRequest represents the change password form data
type ChangePasswordRequest struct {
	CurrentPassword string `form:"current_password" binding:"required"`
	NewPassword     string `form:"new_password" binding:"required,min=12"`
	ConfirmPassword string `form:"confirm_password" binding:"required"`
}

// ChangePassword handles password change requests
func (h *UserHandler) ChangePassword(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "dashboard.html", gin.H{
			"title":         "Dashboard - " + h.config.AppName,
			"user":          user,
			"passwordError": "Please fill in all fields",
			"csrf_token":    middleware.GetCSRFToken(c),
		})
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		c.HTML(http.StatusBadRequest, "dashboard.html", gin.H{
			"title":         "Dashboard - " + h.config.AppName,
			"user":          user,
			"passwordError": "New passwords do not match",
			"csrf_token":    middleware.GetCSRFToken(c),
		})
		return
	}

	// Validate password strength
	if err := helpers.ValidatePasswordStrength(req.NewPassword); err != nil {
		c.HTML(http.StatusBadRequest, "dashboard.html", gin.H{
			"title":         "Dashboard - " + h.config.AppName,
			"user":          user,
			"passwordError": err.Error(),
			"csrf_token":    middleware.GetCSRFToken(c),
		})
		return
	}

	// Verify current password
	_, err := h.userService.Authenticate(user.Username, req.CurrentPassword)
	if err != nil {
		c.HTML(http.StatusBadRequest, "dashboard.html", gin.H{
			"title":         "Dashboard - " + h.config.AppName,
			"user":          user,
			"passwordError": "Current password is incorrect",
			"csrf_token":    middleware.GetCSRFToken(c),
		})
		return
	}

	// Update password
	if err := h.userService.UpdatePassword(user.ID, req.NewPassword); err != nil {
		c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"title":         "Dashboard - " + h.config.AppName,
			"user":          user,
			"passwordError": "Failed to update password",
			"csrf_token":    middleware.GetCSRFToken(c),
		})
		return
	}

	c.Redirect(http.StatusFound, "/dashboard?success=password_changed")
}

// GenerateAPIKey generates a new API key for the current user
func (h *UserHandler) GenerateAPIKey(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Generate the key - this returns the plaintext key (only time it's available)
	apiKey, err := h.userService.GenerateAPIKey(user.ID)
	if err != nil {
		c.Redirect(http.StatusFound, "/dashboard?error=api_key_failed")
		return
	}

	// Store the plaintext key in session flash so it can be shown once
	session := sessions.Default(c)
	session.Set("new_api_key", apiKey)
	session.Save()

	c.Redirect(http.StatusFound, "/dashboard?success=api_key_generated")
}

// MyLinks renders the user's links management page
func (h *UserHandler) MyLinks(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Parse pagination
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := parseInt(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	limit := 20
	offset := (page - 1) * limit

	// Get search and sort parameters
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "created_at")
	sortOrder := c.DefaultQuery("order", "desc")

	// Get user's links with search and sort
	links, total, err := h.linkService.SearchLinks(services.LinkSearchParams{
		Creator:   user.Username,
		Search:    search,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":   "Error",
			"message": "Failed to load links",
		})
		return
	}

	totalPages := (int(total) + limit - 1) / limit

	c.HTML(http.StatusOK, "my_links.html", gin.H{
		"title":      "My Links - " + h.config.AppName,
		"user":       user,
		"links":      links,
		"total":      total,
		"page":       page,
		"totalPages": totalPages,
		"appURL":     h.config.AppURL,
		"csrf_token": middleware.GetCSRFToken(c),
		"search":     search,
		"sortBy":     sortBy,
		"sortOrder":  sortOrder,
	})
}

// ToggleMyLink enables or disables a user's own link
func (h *UserHandler) ToggleMyLink(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	linkID, err := parseUint(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/links?error=invalid_id")
		return
	}

	// Get the link and verify ownership
	link, err := h.linkService.GetByID(linkID)
	if err != nil {
		c.Redirect(http.StatusFound, "/links?error=not_found")
		return
	}

	// Check ownership
	if link.Creator != user.Username {
		c.Redirect(http.StatusFound, "/links?error=access_denied")
		return
	}

	if link.IsDisabled {
		h.linkService.EnableLink(linkID)
	} else {
		h.linkService.DisableLink(linkID)
	}

	c.Redirect(http.StatusFound, "/links")
}

// DeleteMyLink deletes a user's own link
func (h *UserHandler) DeleteMyLink(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	linkID, err := parseUint(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusFound, "/links?error=invalid_id")
		return
	}

	// Get the link and verify ownership
	link, err := h.linkService.GetByID(linkID)
	if err != nil {
		c.Redirect(http.StatusFound, "/links?error=not_found")
		return
	}

	// Check ownership
	if link.Creator != user.Username {
		c.Redirect(http.StatusFound, "/links?error=access_denied")
		return
	}

	if err := h.linkService.DeleteLink(linkID); err != nil {
		c.Redirect(http.StatusFound, "/links?error=delete_failed")
		return
	}

	c.Redirect(http.StatusFound, "/links?success=deleted")
}

// parseInt is a helper to parse page numbers
func parseInt(s string) (int, error) {
	var i int
	_, err := parseIntHelper(s, &i)
	return i, err
}

func parseIntHelper(s string, i *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid number")
		}
		n = n*10 + int(c-'0')
	}
	*i = n
	return n, nil
}

// parseUint is a helper to parse link IDs
func parseUint(s string) (uint, error) {
	n, err := parseInt(s)
	if err != nil || n < 0 {
		return 0, errors.New("invalid number")
	}
	return uint(n), nil
}
