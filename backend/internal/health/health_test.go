package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(_ context.Context) error { return f.err }

func newRouter(db, cache Pinger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", Handler(db, cache))
	return r
}

func do(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	return w
}

func TestHealthzOKWhenBothServicesRespond(t *testing.T) {
	w := do(t, newRouter(fakePinger{}, fakePinger{}))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestHealthzUnavailableWhenPostgresIsDown(t *testing.T) {
	w := do(t, newRouter(fakePinger{err: errors.New("no pg")}, fakePinger{}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"postgres":"no pg"`) {
		t.Errorf("body = %s, want the postgres error reported", w.Body.String())
	}
}

func TestHealthzUnavailableWhenRedisIsDown(t *testing.T) {
	w := do(t, newRouter(fakePinger{}, fakePinger{err: errors.New("no redis")}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"redis":"no redis"`) {
		t.Errorf("body = %s, want the redis error reported", w.Body.String())
	}
}
