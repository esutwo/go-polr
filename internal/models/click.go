package models

import (
	"time"
)

// Click represents a click/visit to a shortened URL
// Schema matches the existing Polr clicks table
type Click struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	IP          string    `gorm:"size:255;not null;index" json:"-"`
	Country     *string   `gorm:"size:255" json:"country,omitempty"`
	Referer     *string   `gorm:"size:255" json:"referer,omitempty"`
	RefererHost *string   `gorm:"column:referer_host;size:255;index" json:"referer_host,omitempty"`
	UserAgent   *string   `gorm:"type:text" json:"user_agent,omitempty"`
	LinkID      uint      `gorm:"not null;index" json:"link_id"`
	Link        Link      `gorm:"foreignKey:LinkID;constraint:OnDelete:CASCADE" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Click) TableName() string {
	return "clicks"
}

// NewClick creates a new Click record with the provided details
func NewClick(linkID uint, ip string, referer, refererHost, userAgent, country *string) *Click {
	return &Click{
		LinkID:      linkID,
		IP:          ip,
		Referer:     referer,
		RefererHost: refererHost,
		UserAgent:   userAgent,
		Country:     country,
	}
}
