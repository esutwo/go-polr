package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nnnc-org/go-polr/internal/helpers"
	"github.com/nnnc-org/go-polr/internal/models"
	"github.com/nnnc-org/go-polr/testutil"
	"gorm.io/gorm"
)

// activeUserWithAPIKey creates an active user, sets their api_key (stored as hash)
// and api_active=true, and returns the plaintext key.
func activeUserWithAPIKey(t *testing.T, db *gorm.DB, username string) (uint, string) {
	t.Helper()
	u, err := testutil.CreateTestUser(db, username, "p", models.RoleUser)
	require.NoError(t, err)

	plaintext, err := helpers.GenerateAPIKey()
	require.NoError(t, err)
	hashed := helpers.HashAPIKey(plaintext)

	require.NoError(t, db.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"api_key":    hashed,
		"api_active": true,
	}).Error)
	return u.ID, plaintext
}

func TestAPIAuth_MissingKeyAnonDisabled_401(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	r.GET("/api", APIAuth(db, false), func(c *gin.Context) { c.String(200, "ok") })

	w := performRequest(r, httptest.NewRequest("GET", "/api", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body APIError
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, ErrCodeAuthError, body.ErrorCode)
}

func TestAPIAuth_InvalidKey_401(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	r.GET("/api", APIAuth(db, false), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/api?key=garbage", nil)
	w := performRequest(r, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIAuth_ValidKey_AttachesUser(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	uid, plaintext := activeUserWithAPIKey(t, db, "alice")

	r := newTestEngine()
	r.GET("/api", APIAuth(db, false), func(c *gin.Context) {
		u := GetAPIUser(c)
		require.NotNil(t, u)
		assert.Equal(t, uid, u.ID)
		assert.False(t, IsAnonymousAPIUser(c))
		c.String(200, u.Username)
	})

	req := httptest.NewRequest("GET", "/api?key="+plaintext, nil)
	w := performRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "alice", w.Body.String())
}

func TestAPIAuth_KeyViaHeader(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	_, plaintext := activeUserWithAPIKey(t, db, "alice")

	r := newTestEngine()
	r.GET("/api", APIAuth(db, false), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/api", nil)
	req.Header.Set("X-API-Key", plaintext)
	w := performRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIAuth_AnonEnabled_AttachesPseudoUser(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	r.GET("/api", APIAuth(db, true), func(c *gin.Context) {
		assert.True(t, IsAnonymousAPIUser(c))
		u := GetAPIUser(c)
		require.NotNil(t, u)
		c.String(200, u.Username)
	})

	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "10.20.30.40:12345"
	w := performRequest(r, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ANONIP:", "anon username must encode IP for per-IP rate limiting")
}

func TestAPIAuth_InactiveAccount_401(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	uid, plaintext := activeUserWithAPIKey(t, db, "alice")

	// Deactivate the account but leave api_active=true.
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", uid).Update("active", "0").Error)

	r := newTestEngine()
	r.GET("/api", APIAuth(db, false), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/api?key="+plaintext, nil)
	w := performRequest(r, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "inactive user must not authenticate")
}

func TestAPIAuth_APIDisabled_401(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)
	uid, plaintext := activeUserWithAPIKey(t, db, "alice")

	require.NoError(t, db.Model(&models.User{}).Where("id = ?", uid).Update("api_active", false).Error)

	r := newTestEngine()
	r.GET("/api", APIAuth(db, false), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/api?key="+plaintext, nil)
	w := performRequest(r, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "api_active=false must block")
}

func TestCreateAnonymousUser_Shape(t *testing.T) {
	// Already covered by middleware_test.go's TestCreateAnonymousUser for the
	// happy path; here we just ensure a different IP yields a distinct username.
	u := createAnonymousUser("10.0.0.1")
	assert.Equal(t, "ANONIP:10.0.0.1", u.Username)
	assert.Equal(t, "user", u.Role)
}
