package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/middleware"
	"github.com/nnnc-org/go-polr/internal/services"
)

// AnalyticsAPIHandler handles API requests for analytics
type AnalyticsAPIHandler struct {
	linkService  *services.LinkService
	statsService *services.StatsService
}

// NewAnalyticsAPIHandler creates a new AnalyticsAPIHandler
func NewAnalyticsAPIHandler(linkService *services.LinkService, statsService *services.StatsService) *AnalyticsAPIHandler {
	return &AnalyticsAPIHandler{
		linkService:  linkService,
		statsService: statsService,
	}
}

// LookupResponse represents the analytics lookup API response
type AnalyticsLookupResponse struct {
	Action string      `json:"action"`
	Result interface{} `json:"result"`
}

// Lookup retrieves analytics for a shortened URL
func (h *AnalyticsAPIHandler) Lookup(c *gin.Context) {
	urlEnding := c.Query("url_ending")
	if urlEnding == "" {
		middleware.RespondAPIError(c, http.StatusBadRequest, "Missing required parameter: url_ending", middleware.ErrCodeMissingParams)
		return
	}

	statsType := c.DefaultQuery("stats_type", "day")
	leftBound := c.Query("left_bound")
	rightBound := c.Query("right_bound")

	// Get the link
	link, err := h.linkService.GetByShortURL(urlEnding)
	if err != nil {
		if err == services.ErrLinkNotFound {
			middleware.RespondAPIError(c, http.StatusNotFound, "Link not found", middleware.ErrCodeNotFound)
		} else {
			middleware.RespondAPIError(c, http.StatusInternalServerError, "Failed to lookup link", middleware.ErrCodeNotFound)
		}
		return
	}

	// Check access: must be creator or admin
	user := middleware.GetAPIUser(c)
	if user == nil {
		middleware.RespondAPIError(c, http.StatusUnauthorized, "Authentication required", middleware.ErrCodeAuthError)
		return
	}

	if link.Creator != user.Username && !user.IsAdmin() {
		middleware.RespondAPIError(c, http.StatusForbidden, "You do not have permission to view these analytics", middleware.ErrCodeAccessDenied)
		return
	}

	// Parse date bounds with proper error handling
	var startDate, endDate time.Time
	var parseErr error

	if leftBound != "" {
		startDate, parseErr = time.Parse("2006-01-02", leftBound)
		if parseErr != nil {
			middleware.RespondAPIError(c, http.StatusBadRequest, "Invalid date format for left_bound. Use YYYY-MM-DD", middleware.ErrCodeInvalidParams)
			return
		}
	} else {
		startDate = time.Now().AddDate(0, 0, -30) // Default: last 30 days
	}

	if rightBound != "" {
		endDate, parseErr = time.Parse("2006-01-02", rightBound)
		if parseErr != nil {
			middleware.RespondAPIError(c, http.StatusBadRequest, "Invalid date format for right_bound. Use YYYY-MM-DD", middleware.ErrCodeInvalidParams)
			return
		}
	} else {
		endDate = time.Now()
	}
	// Include the full end day
	endDate = endDate.Add(24*time.Hour - time.Second)

	// Get requested statistics
	var result interface{}

	switch statsType {
	case "day":
		stats, err := h.statsService.GetDayStats(link.ID, startDate, endDate)
		if err != nil {
			middleware.RespondAPIError(c, http.StatusInternalServerError, "Failed to get statistics", middleware.ErrCodeNotFound)
			return
		}
		result = stats

	case "referer":
		stats, err := h.statsService.GetRefererStats(link.ID, startDate, endDate)
		if err != nil {
			middleware.RespondAPIError(c, http.StatusInternalServerError, "Failed to get statistics", middleware.ErrCodeNotFound)
			return
		}
		result = stats

	default:
		middleware.RespondAPIError(c, http.StatusBadRequest, "Invalid stats_type. Must be 'day' or 'referer'", middleware.ErrCodeInvalidParams)
		return
	}

	c.JSON(http.StatusOK, AnalyticsLookupResponse{
		Action: "analytics_lookup",
		Result: result,
	})
}
