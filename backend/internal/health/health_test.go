package health

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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

// driverError is what pgx really says: the connection identity is inside it.
const driverError = "failed to connect to `user=english database=english`: 10.0.0.7:5432 (10.0.0.7): dial error: connection refused"

func TestHealthzUnavailableWhenPostgresIsDownNamesItWithoutTheDriverError(t *testing.T) {
	w := do(t, newRouter(fakePinger{err: errors.New(driverError)}, fakePinger{}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"postgres":"unavailable"`) || !strings.Contains(body, `"redis":"ok"`) || !strings.Contains(body, `"status":"unavailable"`) {
		t.Errorf("body = %s, want postgres unavailable, redis ok, status unavailable", body)
	}
	for _, leak := range []string{"user=", "database=", "10.0.0.7", "5432", "dial error"} {
		if strings.Contains(body, leak) {
			t.Errorf("body leaks %q on an unauthenticated route: %s", leak, body)
		}
	}
}

func TestHealthzUnavailableWhenRedisIsDownNamesItWithoutTheDriverError(t *testing.T) {
	w := do(t, newRouter(fakePinger{}, fakePinger{err: errors.New("dial tcp 10.0.0.9:6379: connect: connection refused")}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"redis":"unavailable"`) || strings.Contains(body, "10.0.0.9") || strings.Contains(body, "6379") {
		t.Errorf("body = %s, want the marker and no host/port", body)
	}
}

func TestHealthzNamesBothDependenciesWhenBothAreDown(t *testing.T) {
	w := do(t, newRouter(fakePinger{err: errors.New("no pg")}, fakePinger{err: errors.New("no redis")}))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"postgres":"unavailable"`) || !strings.Contains(w.Body.String(), `"redis":"unavailable"`) {
		t.Errorf("status %d body %s; want 503 with both markers", w.Code, w.Body.String())
	}
}

func TestHealthzLogsTheDriverErrorServerSide(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	do(t, newRouter(fakePinger{err: errors.New(driverError)}, fakePinger{}))

	if !strings.Contains(buf.String(), "user=english database=english") {
		t.Errorf("server log = %q, want the full driver error for the operator", buf.String())
	}
}

// slowPinger never answers: it returns only when its own deadline fires.
type slowPinger struct{}

func (slowPinger) Ping(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }

func TestHealthzGivesEachDependencyItsOwnBudget(t *testing.T) {
	old := PingTimeout
	PingTimeout = 30 * time.Millisecond
	t.Cleanup(func() { PingTimeout = old })

	start := time.Now()
	w := do(t, newRouter(slowPinger{}, slowPinger{}))
	elapsed := time.Since(start)

	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"postgres":"unavailable"`) || !strings.Contains(w.Body.String(), `"redis":"unavailable"`) {
		t.Errorf("status %d body %s; want 503 with both markers", w.Code, w.Body.String())
	}
	// Two sequential budgets: the second dependency is not starved by the
	// first having spent a shared deadline (the folded healthz-shares-one-
	// 2s-deadline finding), and the probe still finishes promptly.
	if elapsed < 2*PingTimeout || elapsed > time.Second {
		t.Errorf("elapsed = %v, want between %v and 1s", elapsed, 2*PingTimeout)
	}
	if strings.Contains(w.Body.String(), "deadline") {
		t.Errorf("body leaks the context error: %s", w.Body.String())
	}
}
