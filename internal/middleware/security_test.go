package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_SetsExpectedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, w.Header().Get("Permissions-Policy"))
}

func TestHTTPSRedirect_DisabledByDefault(t *testing.T) {
	t.Setenv("ENFORCE_HTTPS", "")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(HTTPSRedirect())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTPSRedirect must be a no-op when ENFORCE_HTTPS != true")
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
}

func TestHTTPSRedirect_RedirectsHTTPToHTTPS(t *testing.T) {
	t.Setenv("ENFORCE_HTTPS", "true")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(HTTPSRedirect())
	r.GET("/path", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/path?a=1", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "https://example.com/path?a=1", w.Header().Get("Location"))
}

func TestHTTPSRedirect_HonorsForwardedProto(t *testing.T) {
	t.Setenv("ENFORCE_HTTPS", "true")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(HTTPSRedirect())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "trust X-Forwarded-Proto=https from upstream proxy")
	assert.NotEmpty(t, w.Header().Get("Strict-Transport-Security"))
}

func TestHTTPSRedirect_HSTSOnHTTPSConnections(t *testing.T) {
	t.Setenv("ENFORCE_HTTPS", "true")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(HTTPSRedirect())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=")
	assert.Contains(t, hsts, "includeSubDomains")
}
