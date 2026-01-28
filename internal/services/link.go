package services

import (
	"errors"
	"net/url"

	"github.com/nnnc-org/go-polr/internal/helpers"
	"github.com/nnnc-org/go-polr/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrLinkNotFound is returned when a link is not found
	ErrLinkNotFound = errors.New("link not found")
	// ErrLinkDisabled is returned when a link is disabled
	ErrLinkDisabled = errors.New("link is disabled")
	// ErrInvalidURL is returned when the URL is invalid
	ErrInvalidURL = errors.New("invalid URL")
	// ErrShortURLTaken is returned when the custom short URL is already taken
	ErrShortURLTaken = errors.New("short URL already taken")
	// ErrAccessDenied is returned when access to a secret link is denied
	ErrAccessDenied = errors.New("access denied")
)

// LinkService handles link-related business logic
type LinkService struct {
	db *gorm.DB
}

// NewLinkService creates a new LinkService
func NewLinkService(db *gorm.DB) *LinkService {
	return &LinkService{db: db}
}

// CreateLinkInput represents the input for creating a new link
type CreateLinkInput struct {
	LongURL      string
	CustomEnding string
	IsSecret     bool
	Creator      string
	IP           string
	IsAPI        bool
}

// CreateLinkResult represents the result of creating a link
type CreateLinkResult struct {
	Link      *models.Link
	SecretKey string
}

// Create creates a new shortened link
func (s *LinkService) Create(input CreateLinkInput) (*CreateLinkResult, error) {
	// Validate URL
	if !isValidURL(input.LongURL) {
		return nil, ErrInvalidURL
	}

	// Generate or validate short URL
	var shortURL string
	var isCustom bool

	if input.CustomEnding != "" {
		// Validate custom ending
		if !helpers.IsValidShortCode(input.CustomEnding) {
			return nil, ErrInvalidURL
		}

		// Check if custom ending is taken
		var existing models.Link
		if err := s.db.Where("short_url = ?", input.CustomEnding).First(&existing).Error; err == nil {
			return nil, ErrShortURLTaken
		}

		shortURL = input.CustomEnding
		isCustom = true
	} else {
		// Generate unique short URL
		for {
			code, err := helpers.GenerateShortCode(6)
			if err != nil {
				return nil, err
			}

			var existing models.Link
			if err := s.db.Where("short_url = ?", code).First(&existing).Error; err != nil {
				shortURL = code
				break
			}
		}
	}

	// Generate secret key if requested
	var secretKey string
	if input.IsSecret {
		var err error
		secretKey, err = helpers.GenerateSecretKey(8)
		if err != nil {
			return nil, err
		}
	}

	// Generate URL hash for duplicate detection
	urlHash := helpers.HashURL(input.LongURL)

	link := &models.Link{
		ShortURL:    shortURL,
		LongURL:     input.LongURL,
		IP:          input.IP,
		Creator:     input.Creator,
		SecretKey:   secretKey,
		IsCustom:    isCustom,
		IsAPI:       input.IsAPI,
		LongURLHash: &urlHash,
	}

	if err := s.db.Create(link).Error; err != nil {
		return nil, err
	}

	return &CreateLinkResult{
		Link:      link,
		SecretKey: secretKey,
	}, nil
}

// GetByShortURL retrieves a link by its short URL
func (s *LinkService) GetByShortURL(shortURL string) (*models.Link, error) {
	var link models.Link
	if err := s.db.Where("short_url = ?", shortURL).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}
	return &link, nil
}

// GetByID retrieves a link by its ID
func (s *LinkService) GetByID(id uint) (*models.Link, error) {
	var link models.Link
	if err := s.db.First(&link, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}
	return &link, nil
}

// GetLongURL retrieves the long URL for a short URL, checking access
func (s *LinkService) GetLongURL(shortURL, secretKey string) (string, error) {
	link, err := s.GetByShortURL(shortURL)
	if err != nil {
		return "", err
	}

	if link.IsDisabled {
		return "", ErrLinkDisabled
	}

	if !link.CanAccess(secretKey) {
		return "", ErrAccessDenied
	}

	return link.LongURL, nil
}

// IncrementClicks atomically increments the click count for a link
func (s *LinkService) IncrementClicks(linkID uint) error {
	return s.db.Model(&models.Link{}).Where("id = ?", linkID).
		Update("clicks", gorm.Expr("clicks + 1")).Error
}

// GetLinksByCreator retrieves all links created by a user
// If creator is empty, returns all links
func (s *LinkService) GetLinksByCreator(creator string, limit, offset int) ([]models.Link, int64, error) {
	var links []models.Link
	var total int64

	query := s.db.Model(&models.Link{})
	if creator != "" {
		query = query.Where("creator = ?", creator)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query = s.db.Model(&models.Link{})
	if creator != "" {
		query = query.Where("creator = ?", creator)
	}
	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&links).Error; err != nil {
		return nil, 0, err
	}

	return links, total, nil
}

// LinkSearchParams contains parameters for searching and sorting links
type LinkSearchParams struct {
	Creator   string
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

// validSortColumns defines allowed sort columns to prevent SQL injection
var validSortColumns = map[string]string{
	"id":         "id",
	"short_url":  "short_url",
	"long_url":   "long_url",
	"creator":    "creator",
	"clicks":     "clicks",
	"status":     "is_disabled",
	"created_at": "created_at",
}

// SearchLinks retrieves links with search and sort support
func (s *LinkService) SearchLinks(params LinkSearchParams) ([]models.Link, int64, error) {
	var links []models.Link
	var total int64

	query := s.db.Model(&models.Link{})

	// Filter by creator if specified
	if params.Creator != "" {
		query = query.Where("creator = ?", params.Creator)
	}

	// Apply search filter
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		query = query.Where(
			"short_url LIKE ? OR long_url LIKE ? OR creator LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Determine sort column (default to created_at)
	sortColumn := "created_at"
	if col, ok := validSortColumns[params.SortBy]; ok {
		sortColumn = col
	}

	// Determine sort order (default to DESC)
	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Get paginated results with sorting
	if err := query.Order(sortColumn + " " + sortOrder).
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&links).Error; err != nil {
		return nil, 0, err
	}

	return links, total, nil
}

// DisableLink disables a link
func (s *LinkService) DisableLink(id uint) error {
	return s.db.Model(&models.Link{}).Where("id = ?", id).Update("is_disabled", true).Error
}

// EnableLink enables a link
func (s *LinkService) EnableLink(id uint) error {
	return s.db.Model(&models.Link{}).Where("id = ?", id).Update("is_disabled", false).Error
}

// DeleteLink deletes a link (this will cascade delete clicks)
func (s *LinkService) DeleteLink(id uint) error {
	return s.db.Delete(&models.Link{}, id).Error
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}
