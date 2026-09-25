package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// router mounts the handler behind a stand-in for auth.Require() that sets
// auth.ContextUserID (the real middleware is covered in internal/auth).
func router(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/integrations/google/sync", func(c *gin.Context) {
		if userID != "" {
			c.Set(auth.ContextUserID, userID)
		}
		c.Next()
	}, SyncHandler(svc))
	return r
}

func post(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/google/sync", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestSyncHandlerAnswersTheSpec64Body(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "synced" || body["calendar_event_id"] != PracticeEventID("u1") || body["tasks_created_count"] != float64(28) {
		t.Errorf("body = %v", body)
	}
	if len(body) != 3 {
		t.Errorf("§6.4 body has exactly status, calendar_event_id, tasks_created_count; got %v", body)
	}
}

func TestSyncHandlerMapsReauthTo409(t *testing.T) {
	h := newHarness()
	h.oauth.err = ErrReauthRequired
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusConflict || w.Body.String() != `{"error":"reauth_required"}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

func TestSyncHandlerMapsUpstreamTo502(t *testing.T) {
	h := newHarness()
	h.oauth.err = &UpstreamError{Service: "oauth", Status: 503, Body: "down"}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

func TestSyncHandlerRequiresAUser(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestSyncHandlerMapsPlainErrorsTo500(t *testing.T) {
	h := newHarness()
	h.repo.errs = map[string]error{"Profile": errors.New("pg: connection reset")}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusInternalServerError || w.Body.String() != `{"error":"internal_error"}` {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestSyncHandlerMapsADeadlineTo502(t *testing.T) {
	h := newHarness()
	h.repo.errs = map[string]error{"Profile": context.DeadlineExceeded}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestSyncHandlerMapsAnUnconsumed409To502(t *testing.T) {
	h := newHarness()
	// A 409 from tasklists.insert is not the Calendar insert's "ours already";
	// nothing consumes it, so it must read as "Google is being difficult, retry".
	h.tasks.errs = map[string]error{"InsertTaskList": fmt.Errorf("%w: tasks returned 409", ErrAlreadyExists)}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Fatalf("status %d body %s, want 502 google_unavailable (CODEMAP: other Google failures → 502)", w.Code, w.Body.String())
	}
}

// captureLog routes the stdlib logger into a buffer for one test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func TestSyncHandlerLogsTheFailureServerSideOnly(t *testing.T) {
	buf := captureLog(t)
	h := newHarness()
	h.oauth.err = &UpstreamError{Service: "oauth", Status: 503, Body: `{"error":"backend_error"}`}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Fatalf("client body must stay opaque: %d %s", w.Code, w.Body)
	}
	got := buf.String()
	for _, want := range []string{"google: sync", "user=u1", "oauth", "503", "backend_error"} {
		if !strings.Contains(got, want) {
			t.Errorf("log %q missing %q", got, want)
		}
	}
	// The refresh token the harness hands out and the derived access token
	// must never be written — they are never in an error value; keep it so.
	for _, secret := range []string{"1//refresh", "access-for-"} {
		if strings.Contains(got, secret) {
			t.Errorf("log leaks a token: %q", got)
		}
	}
}

func TestSyncHandlerLogsA500WithTheCauseAndTruncatesLongBodies(t *testing.T) {
	buf := captureLog(t)
	h := newHarness()
	h.oauth.err = &UpstreamError{Service: "oauth", Status: 502, Body: strings.Repeat("x", 5000)}
	post(t, router(h.svc, "u1"))
	if n := strings.Count(buf.String(), "x"); n > 600 {
		t.Errorf("log carries %d bytes of upstream body, want it truncated to ~512", n)
	}

	buf.Reset()
	h = newHarness()
	h.repo.errs = map[string]error{"Profile": errors.New("pg: connection reset")}
	post(t, router(h.svc, "u1"))
	if !strings.Contains(buf.String(), "connection reset") || !strings.Contains(buf.String(), "user=u1") {
		t.Errorf("500 path must log the cause: %q", buf.String())
	}
}

func TestSyncHandlerLogsSuccessWithoutTokens(t *testing.T) {
	buf := captureLog(t)
	h := newHarness()
	post(t, router(h.svc, "u1"))
	got := buf.String()
	if !strings.Contains(got, "user=u1") || !strings.Contains(got, "tasks=28") {
		t.Errorf("success line missing user/tasks: %q", got)
	}
	if strings.Contains(got, "1//refresh") || strings.Contains(got, "access-for-") {
		t.Errorf("success line leaks a token: %q", got)
	}
}
