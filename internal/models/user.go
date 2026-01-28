package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user account in the system
// Schema matches the existing Polr users table
type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Username    string         `gorm:"uniqueIndex;size:255;not null" json:"username"`
	Password    string         `gorm:"size:255;not null" json:"-"`
	Email       string         `gorm:"size:255;not null" json:"email"`
	IP          string         `gorm:"type:text;not null" json:"-"`
	RecoveryKey string         `gorm:"column:recovery_key;size:255;not null" json:"-"`
	Role        string         `gorm:"size:255;not null" json:"role"`
	Active      string         `gorm:"size:255;not null" json:"active"`
	APIKey      *string        `gorm:"column:api_key;size:255" json:"api_key,omitempty"`
	APIActive   bool           `gorm:"column:api_active;not null;default:false" json:"api_active"`
	APIQuota    string         `gorm:"column:api_quota;size:255;not null;default:'60'" json:"api_quota"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsActive returns true if the user account is active
func (u *User) IsActive() bool {
	return u.Active == "1" || u.Active == "true"
}

// HasAPIAccess returns true if the user has API access enabled
func (u *User) HasAPIAccess() bool {
	return u.APIActive && u.APIKey != nil && *u.APIKey != ""
}

// MaskedAPIKey returns a masked version of the API key showing only first and last 4 characters
func (u *User) MaskedAPIKey() string {
	if u.APIKey == nil || *u.APIKey == "" {
		return ""
	}
	key := *u.APIKey
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// Role constants
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)
