package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nnnc-org/go-polr/internal/models"
	"github.com/nnnc-org/go-polr/testutil"
)

// loginAs hits a /test-login route that calls SetSession with the given user,
// returning the cookies that subsequent requests should replay.
func loginAs(t *testing.T, r *gin.Engine, user *models.User) []*http.Cookie {
	t.Helper()
	r.GET("/test-login", func(c *gin.Context) {
		require.NoError(t, SetSession(c, user))
		c.String(http.StatusOK, "ok")
	})
	w := performRequest(r, httptest.NewRequest("GET", "/test-login", nil))
	require.Equal(t, http.StatusOK, w.Code)
	return extractCookies(w)
}

func TestSessionAuth_AnonymousRedirects(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	r.GET("/protected", SessionAuth(db), func(c *gin.Context) { c.String(200, "ok") })

	w := performRequest(r, httptest.NewRequest("GET", "/protected", nil))
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestSessionAuth_DeletedUserClearsAndRedirects(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)

	cookies := loginAs(t, r, user)

	// Now delete the user out from under the session.
	require.NoError(t, db.Delete(&models.User{}, user.ID).Error)

	r.GET("/protected", SessionAuth(db), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/protected", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestSessionAuth_InactiveUserRedirectsWithError(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)

	cookies := loginAs(t, r, user)

	// Deactivate the account.
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", user.ID).Update("active", "0").Error)

	r.GET("/protected", SessionAuth(db), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/protected", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login?error=inactive", w.Header().Get("Location"))
}

func TestSessionAuth_Success_AttachesUser(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)

	cookies := loginAs(t, r, user)

	r.GET("/protected", SessionAuth(db), func(c *gin.Context) {
		u := GetCurrentUser(c)
		require.NotNil(t, u)
		c.String(200, u.Username)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "alice", w.Body.String())
}

func TestAdminOnly_NonAdminForbidden(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)
	cookies := loginAs(t, r, user)

	r.GET("/admin", SessionAuth(db), AdminOnly(), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/admin", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminOnly_AdminAllowed(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "boss", "p", models.RoleAdmin)
	require.NoError(t, err)
	cookies := loginAs(t, r, user)

	r.GET("/admin", SessionAuth(db), AdminOnly(), func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/admin", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOptionalSessionAuth_AnonPassThrough(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	r.GET("/maybe", OptionalSessionAuth(db), func(c *gin.Context) {
		if u := GetCurrentUser(c); u != nil {
			c.String(200, u.Username)
		} else {
			c.String(200, "anon")
		}
	})

	w := performRequest(r, httptest.NewRequest("GET", "/maybe", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "anon", w.Body.String())
}

func TestOptionalSessionAuth_LoggedInAttaches(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)
	cookies := loginAs(t, r, user)

	r.GET("/maybe", OptionalSessionAuth(db), func(c *gin.Context) {
		if u := GetCurrentUser(c); u != nil {
			c.String(200, u.Username)
		} else {
			c.String(200, "anon")
		}
	})

	req := httptest.NewRequest("GET", "/maybe", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, "alice", w.Body.String())
}

func TestOptionalSessionAuth_InactiveNotAttached(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)
	cookies := loginAs(t, r, user)
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", user.ID).Update("active", "0").Error)

	r.GET("/maybe", OptionalSessionAuth(db), func(c *gin.Context) {
		if u := GetCurrentUser(c); u != nil {
			c.String(200, "loggedIn")
		} else {
			c.String(200, "anon")
		}
	})

	req := httptest.NewRequest("GET", "/maybe", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	assert.Equal(t, "anon", w.Body.String(), "inactive user must not be attached")
}

func TestClearSession_RemovesUser(t *testing.T) {
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	r := newTestEngine()
	user, err := testutil.CreateTestUser(db, "alice", "p", models.RoleUser)
	require.NoError(t, err)
	cookies := loginAs(t, r, user)

	r.GET("/logout", func(c *gin.Context) {
		require.NoError(t, ClearSession(c))
		c.String(200, "bye")
	})
	r.GET("/protected", SessionAuth(db), func(c *gin.Context) { c.String(200, "ok") })

	// Logout
	req := httptest.NewRequest("GET", "/logout", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := performRequest(r, req)
	require.Equal(t, http.StatusOK, w.Code)
	postLogout := extractCookies(w)

	// Now try protected with the post-logout cookies — should redirect.
	req2 := httptest.NewRequest("GET", "/protected", nil)
	for _, c := range postLogout {
		req2.AddCookie(c)
	}
	w2 := performRequest(r, req2)
	assert.Equal(t, http.StatusFound, w2.Code)
}

func TestIsLoggedIn(t *testing.T) {
	r := newTestEngine()
	r.GET("/check", func(c *gin.Context) {
		if IsLoggedIn(c) {
			c.String(200, "yes")
		} else {
			c.String(200, "no")
		}
	})

	w := performRequest(r, httptest.NewRequest("GET", "/check", nil))
	assert.Equal(t, "no", w.Body.String())
}
