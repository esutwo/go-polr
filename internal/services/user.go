package services

import (
	"errors"

	"github.com/nnnc-org/go-polr/internal/helpers"
	"github.com/nnnc-org/go-polr/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrUserInactive is returned when a user account is inactive
	ErrUserInactive = errors.New("user account is inactive")
	// ErrUsernameTaken is returned when the username is already taken
	ErrUsernameTaken = errors.New("username already taken")
	// ErrEmailTaken is returned when the email is already taken
	ErrEmailTaken = errors.New("email already taken")
)

// UserService handles user-related business logic
type UserService struct {
	db *gorm.DB
}

// NewUserService creates a new UserService
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// CreateUserInput represents the input for creating a new user
type CreateUserInput struct {
	Username string
	Password string
	Email    string
	IP       string
	Role     string
}

// Create creates a new user
func (s *UserService) Create(input CreateUserInput) (*models.User, error) {
	// Check if username is taken
	var existing models.User
	if err := s.db.Where("username = ?", input.Username).First(&existing).Error; err == nil {
		return nil, ErrUsernameTaken
	}

	// Check if email is taken
	if err := s.db.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := helpers.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	// Generate recovery key
	recoveryKey, err := helpers.GenerateRecoveryKey()
	if err != nil {
		return nil, err
	}

	role := input.Role
	if role == "" {
		role = models.RoleUser
	}

	user := &models.User{
		Username:    input.Username,
		Password:    hashedPassword,
		Email:       input.Email,
		IP:          input.IP,
		RecoveryKey: recoveryKey,
		Role:        role,
		Active:      "1",
		APIQuota:    "60",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// Authenticate authenticates a user by username and password
func (s *UserService) Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !helpers.CheckPassword(password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, ErrUserInactive
	}

	return &user, nil
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by username
func (s *UserService) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByAPIKey retrieves a user by API key
func (s *UserService) GetByAPIKey(apiKey string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("api_key = ? AND api_active = ?", apiKey, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdatePassword updates a user's password
func (s *UserService) UpdatePassword(userID uint, newPassword string) error {
	hashedPassword, err := helpers.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

// GenerateAPIKey generates a new API key for a user
// Returns the plaintext key (only shown once) and stores the hash in the database
func (s *UserService) GenerateAPIKey(userID uint) (string, error) {
	apiKey, err := helpers.GenerateAPIKey()
	if err != nil {
		return "", err
	}

	// Store the hash, not the plaintext key
	hashedKey := helpers.HashAPIKey(apiKey)

	err = s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"api_key":    hashedKey,
		"api_active": true,
	}).Error
	if err != nil {
		return "", err
	}

	// Return the plaintext key - this is the only time it's available
	return apiKey, nil
}

// DisableAPIAccess disables API access for a user
func (s *UserService) DisableAPIAccess(userID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("api_active", false).Error
}

// GetAllUsers retrieves all users (for admin)
func (s *UserService) GetAllUsers(limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ActivateUser activates a user account
func (s *UserService) ActivateUser(userID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("active", "1").Error
}

// DeactivateUser deactivates a user account
func (s *UserService) DeactivateUser(userID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("active", "0").Error
}

// DeleteUser soft-deletes a user
func (s *UserService) DeleteUser(userID uint) error {
	return s.db.Delete(&models.User{}, userID).Error
}

// UpdateUserInput represents the input for updating a user
type UpdateUserInput struct {
	Username string
	Email    string
	Role     string
	Active   bool
	APIQuota string
}

// UpdateUser updates a user's profile information
func (s *UserService) UpdateUser(userID uint, input UpdateUserInput) error {
	// Check if username is taken by another user
	var existing models.User
	if err := s.db.Where("username = ? AND id != ?", input.Username, userID).First(&existing).Error; err == nil {
		return ErrUsernameTaken
	}

	// Check if email is taken by another user
	if err := s.db.Where("email = ? AND id != ?", input.Email, userID).First(&existing).Error; err == nil {
		return ErrEmailTaken
	}

	activeStr := "0"
	if input.Active {
		activeStr = "1"
	}

	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"username":  input.Username,
		"email":     input.Email,
		"role":      input.Role,
		"active":    activeStr,
		"api_quota": input.APIQuota,
	}).Error
}

// UpdateUserPassword updates a user's password (admin function)
func (s *UserService) UpdateUserPassword(userID uint, newPassword string) error {
	hashedPassword, err := helpers.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}
