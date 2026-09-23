package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	if body["status"] != "synced" || body["calendar_event_id"] != "evt_new" || body["tasks_created_count"] != float64(28) {
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
