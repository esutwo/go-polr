package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/config"
	"github.com/nnnc-org/go-polr/internal/middleware"
	"github.com/nnnc-org/go-polr/internal/services"
)

// LinkAPIHandler handles API requests for link operations
type LinkAPIHandler struct {
	linkService *services.LinkService
	config      *config.Config
}

// NewLinkAPIHandler creates a new LinkAPIHandler
func NewLinkAPIHandler(linkService *services.LinkService, cfg *config.Config) *LinkAPIHandler {
	return &LinkAPIHandler{
		linkService: linkService,
		config:      cfg,
	}
}

// ShortenRequest represents the API request for shortening a URL
type ShortenRequest struct {
	URL          string `json:"url" form:"url" binding:"required"`
	CustomEnding string `json:"custom_ending" form:"custom_ending"`
	IsSecret     bool   `json:"is_secret" form:"is_secret"`
}

// ShortenResponse represents the API response for a shortened URL
type ShortenResponse struct {
	Action    string `json:"action"`
	Result    string `json:"result"`
	ShortURL  string `json:"short_url,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
}

// Shorten creates a new shortened URL via API
func (h *LinkAPIHandler) Shorten(c *gin.Context) {
	var req ShortenRequest
	if err := c.ShouldBind(&req); err != nil {
		middleware.RespondAPIError(c, http.StatusBadRequest, "Missing required parameter: url", middleware.ErrCodeMissingParams)
		return
	}

	user := middleware.GetAPIUser(c)
	creator := "anonymous"
	if user != nil {
		creator = user.Username
	}

	result, err := h.linkService.Create(services.CreateLinkInput{
		LongURL:      req.URL,
		CustomEnding: req.CustomEnding,
		IsSecret:     req.IsSecret,
		Creator:      creator,
		IP:           c.ClientIP(),
		IsAPI:        true,
	})

	if err != nil {
		switch err {
		case services.ErrInvalidURL:
			middleware.RespondAPIError(c, http.StatusBadRequest, "Invalid URL provided", middleware.ErrCodeInvalidParams)
		case services.ErrShortURLTaken:
			middleware.RespondAPIError(c, http.StatusConflict, "Custom ending already exists", middleware.ErrCodeCreationError)
		default:
			middleware.RespondAPIError(c, http.StatusInternalServerError, "Failed to create short URL", middleware.ErrCodeCreationError)
		}
		return
	}

	shortURL := h.config.AppURL + "/" + result.Link.ShortURL

	response := ShortenResponse{
		Action:   "shorten",
		Result:   shortURL,
		ShortURL: result.Link.ShortURL,
	}

	if result.SecretKey != "" {
		response.SecretKey = result.SecretKey
	}

	c.JSON(http.StatusOK, response)
}

// BulkShortenRequest represents the API request for bulk shortening
type BulkShortenRequest struct {
	URLs []ShortenRequest `json:"urls" binding:"required"`
}

// BulkShortenResponseItem represents a single item in bulk shorten response
type BulkShortenResponseItem struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url,omitempty"`
	SecretKey   string `json:"secret_key,omitempty"`
	Error       string `json:"error,omitempty"`
}

// BulkShorten creates multiple shortened URLs via API
func (h *LinkAPIHandler) BulkShorten(c *gin.Context) {
	var req BulkShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondAPIError(c, http.StatusBadRequest, "Invalid request format", middleware.ErrCodeMissingParams)
		return
	}

	if len(req.URLs) == 0 {
		middleware.RespondAPIError(c, http.StatusBadRequest, "No URLs provided", middleware.ErrCodeMissingParams)
		return
	}

	if len(req.URLs) > 100 {
		middleware.RespondAPIError(c, http.StatusBadRequest, "Maximum 100 URLs per request", middleware.ErrCodeInvalidParams)
		return
	}

	user := middleware.GetAPIUser(c)
	creator := "anonymous"
	if user != nil {
		creator = user.Username
	}

	results := make([]BulkShortenResponseItem, len(req.URLs))

	for i, urlReq := range req.URLs {
		results[i].OriginalURL = urlReq.URL

		result, err := h.linkService.Create(services.CreateLinkInput{
			LongURL:      urlReq.URL,
			CustomEnding: urlReq.CustomEnding,
			IsSecret:     urlReq.IsSecret,
			Creator:      creator,
			IP:           c.ClientIP(),
			IsAPI:        true,
		})

		if err != nil {
			switch err {
			case services.ErrInvalidURL:
				results[i].Error = "Invalid URL"
			case services.ErrShortURLTaken:
				results[i].Error = "Custom ending already exists"
			default:
				results[i].Error = "Failed to create short URL"
			}
			continue
		}

		results[i].ShortURL = h.config.AppURL + "/" + result.Link.ShortURL
		if result.SecretKey != "" {
			results[i].SecretKey = result.SecretKey
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"action":  "shorten_bulk",
		"results": results,
	})
}

// LookupRequest represents the API request for looking up a link
type LookupRequest struct {
	URLEnding string `form:"url_ending" binding:"required"`
	URLKey    string `form:"url_key"`
}

// LookupResponse represents the API response for a link lookup
type LookupResponse struct {
	Action    string `json:"action"`
	LongURL   string `json:"long_url"`
	CreatedAt string `json:"created_at"`
	Clicks    int    `json:"clicks"`
	UpdatedAt string `json:"updated_at"`
}

// Lookup retrieves information about a shortened URL
func (h *LinkAPIHandler) Lookup(c *gin.Context) {
	urlEnding := c.Query("url_ending")
	if urlEnding == "" {
		middleware.RespondAPIError(c, http.StatusBadRequest, "Missing required parameter: url_ending", middleware.ErrCodeMissingParams)
		return
	}

	urlKey := c.Query("url_key")

	link, err := h.linkService.GetByShortURL(urlEnding)
	if err != nil {
		if err == services.ErrLinkNotFound {
			middleware.RespondAPIError(c, http.StatusNotFound, "Link not found", middleware.ErrCodeNotFound)
		} else {
			middleware.RespondAPIError(c, http.StatusInternalServerError, "Failed to lookup link", middleware.ErrCodeNotFound)
		}
		return
	}

	if link.IsDisabled {
		middleware.RespondAPIError(c, http.StatusGone, "Link has been disabled", middleware.ErrCodeNotFound)
		return
	}

	if !link.CanAccess(urlKey) {
		middleware.RespondAPIError(c, http.StatusForbidden, "Invalid or missing secret key", middleware.ErrCodeAccessDenied)
		return
	}

	c.JSON(http.StatusOK, LookupResponse{
		Action:    "lookup",
		LongURL:   link.LongURL,
		CreatedAt: link.CreatedAt.Format("2006-01-02 15:04:05"),
		Clicks:    link.Clicks,
		UpdatedAt: link.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}
