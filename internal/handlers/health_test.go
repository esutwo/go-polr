package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/nnnc-org/go-polr/testutil"
)

func TestHealth_OK(t *testing.T) {
	db, err := testutil.SetupTestDB()
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", Health(db))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "ok", body["database"])
}

func TestHealth_DBDown(t *testing.T) {
	db, err := testutil.SetupTestDB()
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, sqlDB.Close())

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", Health(db))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unhealthy", body["status"])
	assert.Equal(t, "unreachable", body["database"])
}

func TestHealth_BypassesHostHeader(t *testing.T) {
	db, err := testutil.SetupTestDB()
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", Health(db))

	// Simulate a reverse-proxy probe hitting an internal IP/host that doesn't
	// match the configured AppURL.
	req := httptest.NewRequest("GET", "/health", nil)
	req.Host = "10.0.0.42:8080"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
