package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// csrfRouter wires up a GET (to fetch the token) and a POST (to validate).
// Returns the router; tests prime the session via GET, then submit POSTs.
func csrfRouter() *gin.Engine {
	r := newTestEngine()
	r.GET("/form", CSRF(), func(c *gin.Context) {
		c.String(http.StatusOK, GetCSRFToken(c))
	})
	r.POST("/submit", CSRF(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

// primeCSRF makes a GET to fetch a fresh token + session cookie.
func primeCSRF(t *testing.T, r *gin.Engine) (string, []*http.Cookie) {
	t.Helper()
	w := performRequest(r, httptest.NewRequest("GET", "/form", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.NotEmpty(t, w.Body.String(), "GET must expose token via GetCSRFToken")
	return w.Body.String(), extractCookies(w)
}

func TestCSRF_GeneratesTokenOnGET(t *testing.T) {
	r := csrfRouter()
	token, cookies := primeCSRF(t, r)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, cookies, "session cookie must be set so the token persists")
}

func TestCSRF_PostWithoutTokenRejected(t *testing.T) {
	r := csrfRouter()
	_, cookies := primeCSRF(t, r)

	req := httptest.NewRequest("POST", "/submit", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRF_PostWithMismatchedTokenRejected(t *testing.T) {
	r := csrfRouter()
	_, cookies := primeCSRF(t, r)

	form := url.Values{"_csrf": {"not-the-real-token"}}
	req := httptest.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRF_PostWithValidTokenAccepted(t *testing.T) {
	r := csrfRouter()
	token, cookies := primeCSRF(t, r)

	form := url.Values{"_csrf": {token}}
	req := httptest.NewRequest("POST", "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestCSRF_HeaderTokenAlsoAccepted(t *testing.T) {
	r := csrfRouter()
	token, cookies := primeCSRF(t, r)

	req := httptest.NewRequest("POST", "/submit", nil)
	req.Header.Set("X-CSRF-Token", token)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_SafeMethodsNotGated(t *testing.T) {
	// HEAD and OPTIONS must pass without a token.
	r := csrfRouter()
	r.HEAD("/safe", CSRF(), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := performRequest(r, httptest.NewRequest("HEAD", "/safe", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_TokenStableAcrossRequests(t *testing.T) {
	r := csrfRouter()
	token1, cookies := primeCSRF(t, r)

	req := httptest.NewRequest("GET", "/form", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	require.Equal(t, http.StatusOK, w.Code)
	token2 := w.Body.String()

	assert.Equal(t, token1, token2, "token must persist across GETs in the same session")
}
