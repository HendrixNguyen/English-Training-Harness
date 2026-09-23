package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

func router(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/settings/notifications", func(c *gin.Context) {
		if userID != "" {
			c.Set(auth.ContextUserID, userID)
		}
		c.Next()
	}, SettingsHandler(svc))
	return r
}

func post(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// spec64Body is the §6.4 request verbatim.
const spec64Body = `{"notification_time": "20:00:00", "push_subscription": {"endpoint": "push_subscription_endpoint_string", "p256dh": "BNc5T...", "auth": "aX8v..."}}`

func TestSettingsHandlerAcceptsTheSpec64BodyAndAnswersTheSpec64Response(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, "u1"), spec64Body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "updated" || body["notification_time"] != "20:00:00" {
		t.Errorf("body = %v", body)
	}
	if _, ok := body["next_reminder_at"].(string); !ok {
		t.Errorf("next_reminder_at missing: %v", body)
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].Endpoint != "push_subscription_endpoint_string" {
		t.Errorf("subscription not stored: %+v", subs)
	}
}

func TestSettingsHandlerAcceptsTimeOnlyAndOptionalTimezone(t *testing.T) {
	h := newHarness()
	w := post(t, router(h.svc, "u1"), `{"notification_time":"07:15","timezone":"Europe/London"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	if p := h.repo.prefs["u1"]; p.Timezone != "Europe/London" || p.NotificationTime != "07:15:00" {
		t.Errorf("prefs = %+v", p)
	}
}

func TestSettingsHandlerRejectsBadBodies(t *testing.T) {
	for name, body := range map[string]string{
		"not json":                `{`,
		"no notification_time":    `{"push_subscription":{"endpoint":"e","p256dh":"p","auth":"a"}}`,
		"bad clock":               `{"notification_time":"25:99"}`,
		"incomplete subscription": `{"notification_time":"20:00","push_subscription":{"endpoint":"e"}}`,
		"nested keys (not §6.4)":  `{"notification_time":"20:00","push_subscription":{"endpoint":"e","keys":{"p256dh":"p","auth":"a"}}}`,
	} {
		h := newHarness()
		w := post(t, router(h.svc, "u1"), body)
		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"invalid_request"}` {
			t.Errorf("%s: status = %d, body = %s", name, w.Code, w.Body)
		}
		if len(h.queue.scores) != 0 {
			t.Errorf("%s: a rejected request scheduled a reminder", name)
		}
	}
}

func TestSettingsHandlerRequiresAUserAndMapsMissingRowTo404(t *testing.T) {
	h := newHarness()
	if w := post(t, router(h.svc, ""), spec64Body); w.Code != http.StatusUnauthorized {
		t.Errorf("no user: status = %d, want 401", w.Code)
	}
	if w := post(t, router(h.svc, "ghost"), spec64Body); w.Code != http.StatusNotFound || w.Body.String() != `{"error":"user_not_found"}` {
		t.Errorf("missing row: status = %d, body = %s", w.Code, w.Body)
	}
}
