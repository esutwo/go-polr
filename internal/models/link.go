package models

import (
	"crypto/subtle"
	"time"
)

// Link represents a shortened URL
// Schema matches the existing Polr links table
type Link struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ShortURL    string    `gorm:"column:short_url;uniqueIndex;size:255;not null" json:"short_url"`
	LongURL     string    `gorm:"column:long_url;type:longtext;not null" json:"long_url"`
	IP          string    `gorm:"size:255;not null" json:"-"`
	Creator     string    `gorm:"size:255;not null" json:"creator"`
	Clicks      int       `gorm:"not null;default:0" json:"clicks"`
	SecretKey   string    `gorm:"column:secret_key;size:255;not null" json:"-"`
	IsDisabled  bool      `gorm:"column:is_disabled;not null;default:false" json:"is_disabled"`
	IsCustom    bool      `gorm:"column:is_custom;not null;default:false" json:"is_custom"`
	IsAPI       bool      `gorm:"column:is_api;not null;default:false" json:"is_api"`
	LongURLHash *string   `gorm:"column:long_url_hash;size:10;index" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Link) TableName() string {
	return "links"
}

// IsSecret returns true if the link requires a secret key for access
func (l *Link) IsSecret() bool {
	return l.SecretKey != ""
}

// CanAccess checks if the provided key allows access to this link
// Uses constant-time comparison to prevent timing attacks
func (l *Link) CanAccess(key string) bool {
	if !l.IsSecret() {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(l.SecretKey), []byte(key)) == 1
}

// IncrementClicks increments the click counter by 1
// Note: Use services.LinkService.IncrementClicks for atomic updates
func (l *Link) IncrementClicks() {
	l.Clicks++
}
