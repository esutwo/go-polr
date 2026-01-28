package services

import (
	"net/url"

	"github.com/nnnc-org/go-polr/internal/models"
	"gorm.io/gorm"
)

// ClickService handles click tracking and recording
type ClickService struct {
	db *gorm.DB
}

// NewClickService creates a new ClickService
func NewClickService(db *gorm.DB) *ClickService {
	return &ClickService{db: db}
}

// RecordClickInput represents the input for recording a click
type RecordClickInput struct {
	LinkID    uint
	IP        string
	Referer   string
	UserAgent string
}

// Input validation constants
const (
	maxRefererLength   = 2048
	maxUserAgentLength = 1024
	maxHostLength      = 255
)

// RecordClick records a click event for a link
func (s *ClickService) RecordClick(input RecordClickInput) error {
	// Extract referer host with length validation
	var refererHost *string
	var referer *string
	if input.Referer != "" && len(input.Referer) <= maxRefererLength {
		if u, err := url.Parse(input.Referer); err == nil && u.Host != "" && len(u.Host) <= maxHostLength {
			host := u.Host
			refererHost = &host
		}
		referer = &input.Referer
	}

	// Build click record with User-Agent length validation
	var userAgent *string
	if input.UserAgent != "" && len(input.UserAgent) <= maxUserAgentLength {
		userAgent = &input.UserAgent
	}

	click := models.NewClick(input.LinkID, input.IP, referer, refererHost, userAgent, nil)

	return s.db.Create(click).Error
}

// GetClicksByLinkID retrieves all clicks for a link
func (s *ClickService) GetClicksByLinkID(linkID uint, limit, offset int) ([]models.Click, int64, error) {
	var clicks []models.Click
	var total int64

	if err := s.db.Model(&models.Click{}).Where("link_id = ?", linkID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("link_id = ?", linkID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&clicks).Error; err != nil {
		return nil, 0, err
	}

	return clicks, total, nil
}

// GetRecentClicks retrieves the most recent clicks across all links
func (s *ClickService) GetRecentClicks(limit int) ([]models.Click, error) {
	var clicks []models.Click
	if err := s.db.Preload("Link").
		Order("created_at DESC").
		Limit(limit).
		Find(&clicks).Error; err != nil {
		return nil, err
	}
	return clicks, nil
}
