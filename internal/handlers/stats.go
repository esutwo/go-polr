package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/config"
	"github.com/nnnc-org/go-polr/internal/middleware"
	"github.com/nnnc-org/go-polr/internal/services"
)

// StatsHandler handles statistics-related web requests
type StatsHandler struct {
	linkService  *services.LinkService
	statsService *services.StatsService
	config       *config.Config
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(linkService *services.LinkService, statsService *services.StatsService, cfg *config.Config) *StatsHandler {
	return &StatsHandler{
		linkService:  linkService,
		statsService: statsService,
		config:       cfg,
	}
}

// LinkStats renders the statistics page for a link
func (h *StatsHandler) LinkStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title":   "Invalid Link",
			"message": "Invalid link ID provided.",
		})
		return
	}

	link, err := h.linkService.GetByID(uint(linkID))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title":   "Not Found",
			"message": "Link not found.",
		})
		return
	}

	// Check access: must be creator or admin
	if user == nil || (link.Creator != user.Username && !user.IsAdmin()) {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"title":   "Access Denied",
			"message": "You do not have permission to view these statistics.",
		})
		return
	}

	// Parse date range from query params
	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")

	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, _ = time.Parse("2006-01-02", startDateStr)
	} else {
		startDate = time.Now().AddDate(0, 0, -30) // Default: last 30 days
	}

	if endDateStr != "" {
		endDate, _ = time.Parse("2006-01-02", endDateStr)
	} else {
		endDate = time.Now()
	}

	// Ensure end date includes the full day
	endDate = endDate.Add(24*time.Hour - time.Second)

	// Get statistics
	linkStats, _ := h.statsService.GetLinkStats(uint(linkID))
	dayStats, _ := h.statsService.GetDayStats(uint(linkID), startDate, endDate)
	refererStats, _ := h.statsService.GetRefererStats(uint(linkID), startDate, endDate)

	c.HTML(http.StatusOK, "stats.html", gin.H{
		"title":        "Statistics - " + h.config.AppName,
		"user":         user,
		"link":         link,
		"linkStats":    linkStats,
		"dayStats":     dayStats,
		"refererStats": refererStats,
		"startDate":    startDate.Format("2006-01-02"),
		"endDate":      endDate.Format("2006-01-02"),
		"appURL":       h.config.AppURL,
	})
}

// LinkStatsJSON returns link statistics as JSON
func (h *StatsHandler) LinkStatsJSON(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	linkID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link ID"})
		return
	}

	link, err := h.linkService.GetByID(uint(linkID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
		return
	}

	// Check access: must be creator or admin
	if user == nil || (link.Creator != user.Username && !user.IsAdmin()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Parse date range
	startDateStr := c.DefaultQuery("start_date", "")
	endDateStr := c.DefaultQuery("end_date", "")
	statsType := c.DefaultQuery("type", "day")

	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, _ = time.Parse("2006-01-02", startDateStr)
	} else {
		startDate = time.Now().AddDate(0, 0, -30)
	}

	if endDateStr != "" {
		endDate, _ = time.Parse("2006-01-02", endDateStr)
	} else {
		endDate = time.Now()
	}
	endDate = endDate.Add(24*time.Hour - time.Second)

	switch statsType {
	case "day":
		stats, err := h.statsService.GetDayStats(uint(linkID), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
			return
		}
		c.JSON(http.StatusOK, stats)
	case "referer":
		stats, err := h.statsService.GetRefererStats(uint(linkID), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
			return
		}
		c.JSON(http.StatusOK, stats)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stats type. Must be 'day' or 'referer'"})
	}
}
