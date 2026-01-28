package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/config"
	"github.com/nnnc-org/go-polr/internal/middleware"
	"github.com/nnnc-org/go-polr/internal/services"
)

// LinkHandler handles link-related web requests
type LinkHandler struct {
	linkService  *services.LinkService
	clickService *services.ClickService
	config       *config.Config
}

// NewLinkHandler creates a new LinkHandler
func NewLinkHandler(linkService *services.LinkService, clickService *services.ClickService, cfg *config.Config) *LinkHandler {
	return &LinkHandler{
		linkService:  linkService,
		clickService: clickService,
		config:       cfg,
	}
}

// Home renders the homepage with the URL shortening form
func (h *LinkHandler) Home(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	c.HTML(http.StatusOK, "home.html", gin.H{
		"title":      h.config.AppName,
		"user":       user,
		"loggedIn":   user != nil,
		"csrf_token": middleware.GetCSRFToken(c),
	})
}

// CreateLinkRequest represents the request body for creating a link
type CreateLinkRequest struct {
	URL          string `form:"url" binding:"required"`
	CustomEnding string `form:"custom_ending"`
	IsSecret     bool   `form:"is_secret"`
}

// Create handles link creation from the web form
func (h *LinkHandler) Create(c *gin.Context) {
	var req CreateLinkRequest
	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "home.html", gin.H{
			"title":      h.config.AppName,
			"error":      "Please provide a valid URL",
			"csrf_token": middleware.GetCSRFToken(c),
		})
		return
	}

	// Determine creator
	creator := "anonymous"
	if user := middleware.GetCurrentUser(c); user != nil {
		creator = user.Username
	}

	result, err := h.linkService.Create(services.CreateLinkInput{
		LongURL:      req.URL,
		CustomEnding: req.CustomEnding,
		IsSecret:     req.IsSecret,
		Creator:      creator,
		IP:           c.ClientIP(),
		IsAPI:        false,
	})

	if err != nil {
		errorMsg := "Failed to create short URL"
		switch err {
		case services.ErrInvalidURL:
			errorMsg = "Please provide a valid URL"
		case services.ErrShortURLTaken:
			errorMsg = "That custom ending is already taken"
		}

		c.HTML(http.StatusBadRequest, "home.html", gin.H{
			"title":      h.config.AppName,
			"error":      errorMsg,
			"url":        req.URL,
			"csrf_token": middleware.GetCSRFToken(c),
		})
		return
	}

	shortURL := h.config.AppURL + "/" + result.Link.ShortURL
	if result.SecretKey != "" {
		shortURL += "/" + result.SecretKey
	}

	c.HTML(http.StatusOK, "home.html", gin.H{
		"title":      h.config.AppName,
		"success":    true,
		"shortURL":   shortURL,
		"secretKey":  result.SecretKey,
		"user":       middleware.GetCurrentUser(c),
		"loggedIn":   middleware.IsLoggedIn(c),
		"csrf_token": middleware.GetCSRFToken(c),
	})
}

// Redirect handles URL redirection
func (h *LinkHandler) Redirect(c *gin.Context) {
	shortURL := c.Param("shortURL")
	secretKey := c.Param("secretKey")

	// Also check query parameter for secret key
	if secretKey == "" {
		secretKey = c.Query("key")
	}

	link, err := h.linkService.GetByShortURL(shortURL)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title":   "Not Found",
			"message": "The requested short URL was not found.",
		})
		return
	}

	if link.IsDisabled {
		c.HTML(http.StatusGone, "error.html", gin.H{
			"title":   "Link Disabled",
			"message": "This link has been disabled.",
		})
		return
	}

	if !link.CanAccess(secretKey) {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"title":   "Access Denied",
			"message": "This link requires a secret key to access.",
		})
		return
	}

	// Record click asynchronously
	go func() {
		h.clickService.RecordClick(services.RecordClickInput{
			LinkID:    link.ID,
			IP:        c.ClientIP(),
			Referer:   c.GetHeader("Referer"),
			UserAgent: c.GetHeader("User-Agent"),
		})
		h.linkService.IncrementClicks(link.ID)
	}()

	// Perform redirect
	c.Redirect(http.StatusMovedPermanently, link.LongURL)
}
