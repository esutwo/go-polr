package middleware

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

// noopRender accepts any template name and writes nothing — lets handlers that
// call c.HTML() succeed in tests without a real template harness.
type noopRender struct{}

func (noopRender) Instance(string, interface{}) render.Render { return noopRender{} }
func (noopRender) Render(http.ResponseWriter) error           { return nil }
func (noopRender) WriteContentType(http.ResponseWriter)       {}

// newTestEngine returns a Gin engine with cookie-store sessions and a no-op
// HTML renderer, suitable for middleware tests.
func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.HTMLRender = noopRender{}
	store := cookie.NewStore([]byte("test-session-secret-32-chars-long!"))
	r.Use(sessions.Sessions("test_session", store))
	return r
}

// performWithCookieJar replays a single request through r and returns the
// recorder so callers can inspect status, body and Set-Cookie.
func performRequest(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// extractCookies grabs all Set-Cookie response values to be replayed on the
// next request.
func extractCookies(w *httptest.ResponseRecorder) []*http.Cookie {
	resp := http.Response{Header: w.Result().Header}
	return resp.Cookies()
}
