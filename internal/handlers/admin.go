package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/config"
	"github.com/nnnc-org/go-polr/internal/helpers"
	"github.com/nnnc-org/go-polr/internal/middleware"
	"github.com/nnnc-org/go-polr/internal/models"
	"github.com/nnnc-org/go-polr/internal/services"
)

// AdminHandler handles admin-related web requests
type AdminHandler struct {
	userService  *services.UserService
	linkService  *services.LinkService
	statsService *services.StatsService
	config       *config.Config
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(userService *services.UserService, linkService *services.LinkService, statsService *services.StatsService, cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		userService:  userService,
		linkService:  linkService,
		statsService: statsService,
		config:       cfg,
	}
}

// Dashboard renders the admin dashboard
func (h *AdminHandler) Dashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	stats, err := h.statsService.GetDashboardStats()
	if err != nil {
		stats = &services.DashboardStats{}
	}

	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"title":  "Admin Dashboard - " + h.config.AppName,
		"user":   user,
		"stats":  stats,
		"active": "dashboard",
	})
}

// Users renders the user management page
func (h *AdminHandler) Users(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	users, total, err := h.userService.GetAllUsers(limit, offset)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":   "Error",
			"message": "Failed to load users",
		})
		return
	}

	totalPages := (int(total) + limit - 1) / limit

	c.HTML(http.StatusOK, "admin_users.html", gin.H{
		"title":      "User Management - " + h.config.AppName,
		"user":       user,
		"users":      users,
		"total":      total,
		"page":       page,
		"totalPages": totalPages,
		"active":     "users",
		"csrf_token": middleware.GetCSRFToken(c),
	})
}

// Links renders the link management page
func (h *AdminHandler) Links(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	// Get search and sort parameters
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "created_at")
	sortOrder := c.DefaultQuery("order", "desc")

	// Get links with search and sort
	links, total, err := h.linkService.SearchLinks(services.LinkSearchParams{
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

	c.HTML(http.StatusOK, "admin_links.html", gin.H{
		"title":      "Link Management - " + h.config.AppName,
		"user":       user,
		"links":      links,
		"total":      total,
		"page":       page,
		"totalPages": totalPages,
		"active":     "links",
		"appURL":     h.config.AppURL,
		"csrf_token": middleware.GetCSRFToken(c),
		"search":     search,
		"sortBy":     sortBy,
		"sortOrder":  sortOrder,
	})
}

// ToggleLinkStatus enables or disables a link
func (h *AdminHandler) ToggleLinkStatus(c *gin.Context) {
	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/links?error=invalid_id")
		return
	}

	link, err := h.linkService.GetByID(uint(linkID))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/links?error=not_found")
		return
	}

	if link.IsDisabled {
		h.linkService.EnableLink(uint(linkID))
	} else {
		h.linkService.DisableLink(uint(linkID))
	}

	c.Redirect(http.StatusFound, "/admin/links")
}

// DeleteLink deletes a link
func (h *AdminHandler) DeleteLink(c *gin.Context) {
	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/links?error=invalid_id")
		return
	}

	if err := h.linkService.DeleteLink(uint(linkID)); err != nil {
		c.Redirect(http.StatusFound, "/admin/links?error=delete_failed")
		return
	}

	c.Redirect(http.StatusFound, "/admin/links?success=deleted")
}

// ToggleUserStatus activates or deactivates a user
func (h *AdminHandler) ToggleUserStatus(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=invalid_id")
		return
	}

	targetUser, err := h.userService.GetByID(uint(userID))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=not_found")
		return
	}

	// Don't allow deactivating yourself
	currentUser := middleware.GetCurrentUser(c)
	if currentUser != nil && currentUser.ID == targetUser.ID {
		c.Redirect(http.StatusFound, "/admin/users?error=cannot_deactivate_self")
		return
	}

	if targetUser.IsActive() {
		h.userService.DeactivateUser(uint(userID))
	} else {
		h.userService.ActivateUser(uint(userID))
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

// DeleteUser deletes a user
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=invalid_id")
		return
	}

	// Don't allow deleting yourself
	currentUser := middleware.GetCurrentUser(c)
	if currentUser != nil && currentUser.ID == uint(userID) {
		c.Redirect(http.StatusFound, "/admin/users?error=cannot_delete_self")
		return
	}

	if err := h.userService.DeleteUser(uint(userID)); err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=delete_failed")
		return
	}

	c.Redirect(http.StatusFound, "/admin/users?success=deleted")
}

// NewUser renders the admin "create user" form.
func (h *AdminHandler) NewUser(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_user_new.html", gin.H{
		"title":      "Add User - " + h.config.AppName,
		"user":       middleware.GetCurrentUser(c),
		"active":     "users",
		"csrf_token": middleware.GetCSRFToken(c),
		"form":       gin.H{},
	})
}

// CreateUser handles admin user creation. Bypasses the RegistrationEnabled
// flag (which gates public self-signup) since this is an authenticated admin action.
func (h *AdminHandler) CreateUser(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	role := c.PostForm("role")

	if role != models.RoleAdmin {
		role = models.RoleUser
	}

	rerender := func(status int, errMsg string) {
		c.HTML(status, "admin_user_new.html", gin.H{
			"title":      "Add User - " + h.config.AppName,
			"user":       middleware.GetCurrentUser(c),
			"active":     "users",
			"csrf_token": middleware.GetCSRFToken(c),
			"error":      errMsg,
			"form": gin.H{
				"username": username,
				"email":    email,
				"role":     role,
			},
		})
	}

	if username == "" || email == "" || password == "" {
		rerender(http.StatusBadRequest, "Username, email, and password are required")
		return
	}

	if err := helpers.ValidatePasswordStrength(password); err != nil {
		rerender(http.StatusBadRequest, err.Error())
		return
	}

	_, err := h.userService.Create(services.CreateUserInput{
		Username: username,
		Email:    email,
		Password: password,
		Role:     role,
		IP:       c.ClientIP(),
	})
	if err != nil {
		switch err {
		case services.ErrUsernameTaken:
			rerender(http.StatusBadRequest, "Username is already taken")
		case services.ErrEmailTaken:
			rerender(http.StatusBadRequest, "Email is already registered")
		default:
			rerender(http.StatusInternalServerError, "Failed to create user")
		}
		return
	}

	c.Redirect(http.StatusFound, "/admin/users?success=created")
}

// EditUser renders the user edit page
func (h *AdminHandler) EditUser(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=invalid_id")
		return
	}

	targetUser, err := h.userService.GetByID(uint(userID))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=not_found")
		return
	}

	c.HTML(http.StatusOK, "admin_user_edit.html", gin.H{
		"title":      "Edit User - " + h.config.AppName,
		"user":       user,
		"targetUser": targetUser,
		"active":     "users",
		"csrf_token": middleware.GetCSRFToken(c),
	})
}

// UpdateUser handles the user update form submission
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=invalid_id")
		return
	}

	currentUser := middleware.GetCurrentUser(c)

	// Get the target user first
	targetUser, err := h.userService.GetByID(uint(userID))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=not_found")
		return
	}

	// Parse form data
	username := c.PostForm("username")
	email := c.PostForm("email")
	role := c.PostForm("role")
	active := c.PostForm("active") == "1"
	apiQuota := c.PostForm("api_quota")
	newPassword := c.PostForm("password")

	// Validate role
	if role != "admin" && role != "user" {
		role = "user"
	}

	// Prevent demoting yourself
	if currentUser != nil && currentUser.ID == uint(userID) && role != "admin" {
		c.HTML(http.StatusOK, "admin_user_edit.html", gin.H{
			"title":      "Edit User - " + h.config.AppName,
			"user":       currentUser,
			"targetUser": targetUser,
			"active":     "users",
			"csrf_token": middleware.GetCSRFToken(c),
			"error":      "You cannot demote yourself from admin",
		})
		return
	}

	// Prevent deactivating yourself
	if currentUser != nil && currentUser.ID == uint(userID) && !active {
		c.HTML(http.StatusOK, "admin_user_edit.html", gin.H{
			"title":      "Edit User - " + h.config.AppName,
			"user":       currentUser,
			"targetUser": targetUser,
			"active":     "users",
			"csrf_token": middleware.GetCSRFToken(c),
			"error":      "You cannot deactivate yourself",
		})
		return
	}

	// Update user profile
	err = h.userService.UpdateUser(uint(userID), services.UpdateUserInput{
		Username: username,
		Email:    email,
		Role:     role,
		Active:   active,
		APIQuota: apiQuota,
	})

	if err != nil {
		errorMsg := "Failed to update user"
		if err == services.ErrUsernameTaken {
			errorMsg = "Username is already taken"
		} else if err == services.ErrEmailTaken {
			errorMsg = "Email is already taken"
		}

		// Refresh target user data
		targetUser, _ = h.userService.GetByID(uint(userID))

		c.HTML(http.StatusOK, "admin_user_edit.html", gin.H{
			"title":      "Edit User - " + h.config.AppName,
			"user":       currentUser,
			"targetUser": targetUser,
			"active":     "users",
			"csrf_token": middleware.GetCSRFToken(c),
			"error":      errorMsg,
		})
		return
	}

	// Update password if provided
	if newPassword != "" {
		if len(newPassword) < 6 {
			targetUser, _ = h.userService.GetByID(uint(userID))
			c.HTML(http.StatusOK, "admin_user_edit.html", gin.H{
				"title":      "Edit User - " + h.config.AppName,
				"user":       currentUser,
				"targetUser": targetUser,
				"active":     "users",
				"csrf_token": middleware.GetCSRFToken(c),
				"error":      "Password must be at least 6 characters",
			})
			return
		}

		if err := h.userService.UpdateUserPassword(uint(userID), newPassword); err != nil {
			targetUser, _ = h.userService.GetByID(uint(userID))
			c.HTML(http.StatusOK, "admin_user_edit.html", gin.H{
				"title":      "Edit User - " + h.config.AppName,
				"user":       currentUser,
				"targetUser": targetUser,
				"active":     "users",
				"csrf_token": middleware.GetCSRFToken(c),
				"error":      "Failed to update password",
			})
			return
		}
	}

	c.Redirect(http.StatusFound, "/admin/users?success=updated")
}
