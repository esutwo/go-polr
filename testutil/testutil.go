package testutil

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/nnnc-org/go-polr/internal/models"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the models
	err = db.AutoMigrate(&models.User{}, &models.Link{}, &models.Click{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// SetupTestRouter creates a Gin router in test mode
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// MakeRequest performs an HTTP request and returns the response recorder
func MakeRequest(router *gin.Engine, method, path string, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// MakeFormRequest performs a form POST request
func MakeFormRequest(router *gin.Engine, path string, formData map[string]string) *httptest.ResponseRecorder {
	form := make([]string, 0, len(formData))
	for k, v := range formData {
		form = append(form, k+"="+v)
	}
	body := strings.Join(form, "&")

	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// CreateTestUser creates a test user in the database
func CreateTestUser(db *gorm.DB, username, password, role string) (*models.User, error) {
	user := &models.User{
		Username:    username,
		Password:    password, // In tests, we might use plain passwords
		Email:       username + "@test.com",
		IP:          "127.0.0.1",
		RecoveryKey: "test-recovery-key",
		Role:        role,
		Active:      "1",
		APIQuota:    "60",
	}

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// CreateTestLink creates a test link in the database
func CreateTestLink(db *gorm.DB, shortURL, longURL, creator string) (*models.Link, error) {
	link := &models.Link{
		ShortURL: shortURL,
		LongURL:  longURL,
		IP:       "127.0.0.1",
		Creator:  creator,
	}

	if err := db.Create(link).Error; err != nil {
		return nil, err
	}

	return link, nil
}

// CreateTestClick creates a test click in the database
func CreateTestClick(db *gorm.DB, linkID uint, ip string) (*models.Click, error) {
	click := &models.Click{
		LinkID: linkID,
		IP:     ip,
	}

	if err := db.Create(click).Error; err != nil {
		return nil, err
	}

	return click, nil
}
