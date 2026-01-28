package services

import (
	"time"

	"github.com/nnnc-org/go-polr/internal/models"
	"gorm.io/gorm"
)

// StatsService handles analytics and statistics
type StatsService struct {
	db *gorm.DB
}

// NewStatsService creates a new StatsService
func NewStatsService(db *gorm.DB) *StatsService {
	return &StatsService{db: db}
}

// DayStat represents click statistics for a single day
type DayStat struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// CountryStat represents click statistics by country
type CountryStat struct {
	Country string `json:"country"`
	Count   int64  `json:"count"`
}

// RefererStat represents click statistics by referer
type RefererStat struct {
	RefererHost string `json:"referer_host"`
	Count       int64  `json:"count"`
}

// LinkStats represents overall statistics for a link
type LinkStats struct {
	TotalClicks   int64         `json:"total_clicks"`
	UniqueClicks  int64         `json:"unique_clicks"`
	DayStats      []DayStat     `json:"day_stats,omitempty"`
	CountryStats  []CountryStat `json:"country_stats,omitempty"`
	RefererStats  []RefererStat `json:"referer_stats,omitempty"`
}

// GetLinkStats retrieves statistics for a link
func (s *StatsService) GetLinkStats(linkID uint) (*LinkStats, error) {
	stats := &LinkStats{}

	// Get total clicks
	if err := s.db.Model(&models.Click{}).Where("link_id = ?", linkID).Count(&stats.TotalClicks).Error; err != nil {
		return nil, err
	}

	// Get unique clicks (by IP)
	if err := s.db.Model(&models.Click{}).
		Where("link_id = ?", linkID).
		Distinct("ip").
		Count(&stats.UniqueClicks).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetDayStats retrieves daily click statistics for a link within a date range
func (s *StatsService) GetDayStats(linkID uint, startDate, endDate time.Time) ([]DayStat, error) {
	var stats []DayStat

	err := s.db.Model(&models.Click{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("link_id = ? AND created_at >= ? AND created_at <= ?", linkID, startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetCountryStats retrieves click statistics by country for a link
func (s *StatsService) GetCountryStats(linkID uint, startDate, endDate time.Time) ([]CountryStat, error) {
	var stats []CountryStat

	err := s.db.Model(&models.Click{}).
		Select("COALESCE(country, 'Unknown') as country, COUNT(*) as count").
		Where("link_id = ? AND created_at >= ? AND created_at <= ?", linkID, startDate, endDate).
		Group("country").
		Order("count DESC").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetRefererStats retrieves click statistics by referer for a link
func (s *StatsService) GetRefererStats(linkID uint, startDate, endDate time.Time) ([]RefererStat, error) {
	var stats []RefererStat

	err := s.db.Model(&models.Click{}).
		Select("COALESCE(referer_host, 'Direct') as referer_host, COUNT(*) as count").
		Where("link_id = ? AND created_at >= ? AND created_at <= ?", linkID, startDate, endDate).
		Group("referer_host").
		Order("count DESC").
		Limit(20).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// DashboardStats represents overall dashboard statistics
type DashboardStats struct {
	TotalLinks      int64 `json:"total_links"`
	TotalClicks     int64 `json:"total_clicks"`
	TotalUsers      int64 `json:"total_users"`
	LinksToday      int64 `json:"links_today"`
	ClicksToday     int64 `json:"clicks_today"`
}

// GetDashboardStats retrieves overall statistics for the dashboard
func (s *StatsService) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}
	today := time.Now().Truncate(24 * time.Hour)

	// Total links
	if err := s.db.Model(&models.Link{}).Count(&stats.TotalLinks).Error; err != nil {
		return nil, err
	}

	// Total clicks
	if err := s.db.Model(&models.Click{}).Count(&stats.TotalClicks).Error; err != nil {
		return nil, err
	}

	// Total users
	if err := s.db.Model(&models.User{}).Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	// Links created today
	if err := s.db.Model(&models.Link{}).Where("created_at >= ?", today).Count(&stats.LinksToday).Error; err != nil {
		return nil, err
	}

	// Clicks today
	if err := s.db.Model(&models.Click{}).Where("created_at >= ?", today).Count(&stats.ClicksToday).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetUserStats retrieves statistics for a specific user
func (s *StatsService) GetUserStats(username string) (*DashboardStats, error) {
	stats := &DashboardStats{}
	today := time.Now().Truncate(24 * time.Hour)

	// Total links by user
	if err := s.db.Model(&models.Link{}).Where("creator = ?", username).Count(&stats.TotalLinks).Error; err != nil {
		return nil, err
	}

	// Total clicks on user's links
	if err := s.db.Model(&models.Click{}).
		Joins("JOIN links ON clicks.link_id = links.id").
		Where("links.creator = ?", username).
		Count(&stats.TotalClicks).Error; err != nil {
		return nil, err
	}

	// Links created today by user
	if err := s.db.Model(&models.Link{}).Where("creator = ? AND created_at >= ?", username, today).Count(&stats.LinksToday).Error; err != nil {
		return nil, err
	}

	// Clicks today on user's links
	if err := s.db.Model(&models.Click{}).
		Joins("JOIN links ON clicks.link_id = links.id").
		Where("links.creator = ? AND clicks.created_at >= ?", username, today).
		Count(&stats.ClicksToday).Error; err != nil {
		return nil, err
	}

	return stats, nil
}
