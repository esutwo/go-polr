package router

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/nnnc-org/go-polr/internal/config"
	"github.com/nnnc-org/go-polr/internal/handlers"
	"github.com/nnnc-org/go-polr/internal/handlers/api"
	"github.com/nnnc-org/go-polr/internal/middleware"
	"github.com/nnnc-org/go-polr/internal/services"
	"gorm.io/gorm"
)

// Setup configures the Gin router with all routes and middleware
func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())

	// Health check — registered before HTTPS redirect / session middleware so it
	// stays reachable on any host or scheme (e.g. reverse-proxy health probes
	// hitting the container directly over HTTP on an internal IP).
	router.GET("/health", handlers.Health(db))

	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.HTTPSRedirect())

	// Session store with secure cookie settings
	store := cookie.NewStore([]byte(cfg.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   len(cfg.AppURL) >= 5 && cfg.AppURL[:5] == "https",
		SameSite: http.SameSiteStrictMode,
	})
	router.Use(sessions.Sessions("polr_session", store))

	// Register custom template functions
	funcMap := template.FuncMap{
		"minus":   func(a, b int) int { return a - b },
		"plus":    func(a, b int) int { return a + b },
		"appName": func() string { return cfg.AppName },
	}

	// Load HTML templates with base layout inheritance
	router.HTMLRender = loadTemplates("web/templates", funcMap)

	// Static files
	router.Static("/static", "web/static")

	// Initialize services
	linkService := services.NewLinkService(db)
	userService := services.NewUserService(db)
	clickService := services.NewClickService(db)
	statsService := services.NewStatsService(db)

	// Initialize handlers
	linkHandler := handlers.NewLinkHandler(linkService, clickService, cfg)
	userHandler := handlers.NewUserHandler(userService, linkService, statsService, cfg)
	adminHandler := handlers.NewAdminHandler(userService, linkService, statsService, cfg)
	statsHandler := handlers.NewStatsHandler(linkService, statsService, cfg)

	// Initialize API handlers
	linkAPIHandler := api.NewLinkAPIHandler(linkService, cfg)
	analyticsAPIHandler := api.NewAnalyticsAPIHandler(linkService, statsService)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(time.Minute)

	// CSRF middleware for web routes
	csrfMiddleware := middleware.CSRF()

	// ============================================
	// Public Routes
	// ============================================

	// Home page
	router.GET("/", csrfMiddleware, middleware.OptionalSessionAuth(db), linkHandler.Home)

	// Login/Logout/Register
	router.GET("/login", csrfMiddleware, middleware.OptionalSessionAuth(db), userHandler.LoginPage)
	router.POST("/login", csrfMiddleware, userHandler.Login)
	router.GET("/logout", userHandler.Logout)
	router.GET("/register", csrfMiddleware, middleware.OptionalSessionAuth(db), userHandler.RegisterPage)
	router.POST("/register", csrfMiddleware, userHandler.Register)

	// ============================================
	// Authenticated Web Routes
	// ============================================

	authGroup := router.Group("/")
	authGroup.Use(middleware.SessionAuth(db))
	authGroup.Use(csrfMiddleware)
	{
		authGroup.POST("/shorten", linkHandler.Create)
		authGroup.GET("/dashboard", userHandler.Dashboard)
		authGroup.POST("/password", userHandler.ChangePassword)
		authGroup.POST("/api-key/generate", userHandler.GenerateAPIKey)
		authGroup.GET("/stats/:id", statsHandler.LinkStats)
		authGroup.GET("/stats/:id/json", statsHandler.LinkStatsJSON)

		// User link management
		authGroup.GET("/links", userHandler.MyLinks)
		authGroup.POST("/links/:id/toggle", userHandler.ToggleMyLink)
		authGroup.POST("/links/:id/delete", userHandler.DeleteMyLink)
	}

	// ============================================
	// Admin Routes
	// ============================================

	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.SessionAuth(db))
	adminGroup.Use(middleware.AdminOnly())
	adminGroup.Use(csrfMiddleware)
	{
		adminGroup.GET("", adminHandler.Dashboard)
		adminGroup.GET("/dashboard", adminHandler.Dashboard)
		adminGroup.GET("/users", adminHandler.Users)
		adminGroup.GET("/users/:id/edit", adminHandler.EditUser)
		adminGroup.POST("/users/:id/edit", adminHandler.UpdateUser)
		adminGroup.POST("/users/:id/toggle", adminHandler.ToggleUserStatus)
		adminGroup.POST("/users/:id/delete", adminHandler.DeleteUser)
		adminGroup.GET("/links", adminHandler.Links)
		adminGroup.POST("/links/:id/toggle", adminHandler.ToggleLinkStatus)
		adminGroup.POST("/links/:id/delete", adminHandler.DeleteLink)
	}

	// ============================================
	// API v2 Routes
	// ============================================

	apiV2 := router.Group("/api/v2")
	apiV2.Use(middleware.APIAuth(db, cfg.AnonAPIEnabled))
	apiV2.Use(middleware.APIRateLimit(rateLimiter))
	{
		// Link operations - POST only for state-changing operations
		apiV2.POST("/action/shorten", linkAPIHandler.Shorten)
		apiV2.POST("/action/shorten_bulk", linkAPIHandler.BulkShorten)
		apiV2.GET("/link/lookup", linkAPIHandler.Lookup)

		// Analytics
		apiV2.GET("/analytics/lookup", analyticsAPIHandler.Lookup)
	}

	// ============================================
	// Redirect Route - using NoRoute handler
	// ============================================
	// This ensures all explicit routes take priority,
	// and only unmatched GET requests are treated as short URLs
	router.NoRoute(func(c *gin.Context) {
		// Only handle GET requests as potential short URLs
		if c.Request.Method != "GET" {
			c.HTML(404, "error.html", gin.H{
				"title":   "Not Found",
				"message": "The requested page was not found.",
			})
			return
		}

		// Parse the path to extract short URL and optional secret key
		path := c.Request.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}

		// Skip if path is empty
		if path == "" {
			c.HTML(404, "error.html", gin.H{
				"title":   "Not Found",
				"message": "The requested page was not found.",
			})
			return
		}

		// Split path into parts
		parts := splitPath(path)
		if len(parts) == 0 || len(parts) > 2 {
			c.HTML(404, "error.html", gin.H{
				"title":   "Not Found",
				"message": "The requested page was not found.",
			})
			return
		}

		// Set params for the redirect handler
		c.Params = append(c.Params, gin.Param{Key: "shortURL", Value: parts[0]})
		if len(parts) == 2 {
			c.Params = append(c.Params, gin.Param{Key: "secretKey", Value: parts[1]})
		}

		linkHandler.Redirect(c)
	})

	return router
}

// splitPath splits a URL path into its components
func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// templateRenderer implements gin's HTMLRender interface with isolated template sets
type templateRenderer struct {
	templates map[string]*template.Template
}

func (r *templateRenderer) Instance(name string, data interface{}) render.Render {
	return &templateInstance{
		Template: r.templates[name],
		Data:     data,
	}
}

type templateInstance struct {
	Template *template.Template
	Data     interface{}
}

func (t *templateInstance) Render(w http.ResponseWriter) error {
	t.WriteContentType(w)
	return t.Template.Execute(w, t.Data)
}

func (t *templateInstance) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

// loadTemplates loads all templates with base layout inheritance
func loadTemplates(templatesDir string, funcMap template.FuncMap) *templateRenderer {
	templates := make(map[string]*template.Template)

	// Get the base layout content
	baseLayout := filepath.Join(templatesDir, "layouts", "base.html")

	// Find all page templates
	pageTemplates, _ := filepath.Glob(filepath.Join(templatesDir, "pages", "*.html"))
	adminTemplates, _ := filepath.Glob(filepath.Join(templatesDir, "admin", "*.html"))

	allTemplates := append(pageTemplates, adminTemplates...)

	// Parse each page template into its own isolated template set
	for _, pageFile := range allTemplates {
		name := filepath.Base(pageFile)

		// Create a fresh template set for this page
		t := template.New(name).Funcs(funcMap)

		// Parse base layout and page template together
		t, err := t.ParseFiles(baseLayout, pageFile)
		if err != nil {
			panic("Error parsing template " + name + ": " + err.Error())
		}

		// Get the "base" template which is our entry point
		baseT := t.Lookup("base")
		if baseT == nil {
			panic("Template " + name + " does not define 'base'")
		}

		// Store the template - we'll execute "base" which calls navbar, content, scripts
		templates[name] = baseT
	}

	return &templateRenderer{templates: templates}
}
