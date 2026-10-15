package notify

import (
	"context"
	"errors"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadSettingsReloadAndNoWrites(t *testing.T) {
	log := &callLog{}
	repo := newFakeRepo(log)
	repo.prefs["u"] = Preferences{"07:30:00", "America/New_York"}
	now := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	svc := NewService(repo, newFakeQueue(log), nil, nil, func() time.Time { return now })
	out, err := svc.ReadSettings(context.Background(), "u")
	if err != nil || out.NotificationTime != "07:30:00" || out.Timezone != "America/New_York" || out.NextReminderAt != "2026-03-08T11:30:00Z" || out.PushConfigured {
		t.Fatalf("%+v %v", out, err)
	}
	if len(log.calls) != 1 {
		t.Fatalf("read side effects: %v", log.calls)
	}
	repo.prefs["u"] = Preferences{"08:00:00", "Asia/Ho_Chi_Minh"}
	reload := NewService(repo, nil, nil, nil, func() time.Time { return now })
	out, err = reload.ReadSettings(context.Background(), "u")
	if err != nil || out.NotificationTime != "08:00:00" || out.Timezone != "Asia/Ho_Chi_Minh" {
		t.Fatalf("reload=%+v %v", out, err)
	}
	if _, err = svc.ReadSettings(context.Background(), "missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatal(err)
	}
}
func TestReadSettingsHandlerAuthAndMissing(t *testing.T) {
	for _, owner := range []string{"", "missing"} {
		log := &callLog{}
		svc := NewService(newFakeRepo(log), nil, nil, nil, time.Now)
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/settings", func(c *gin.Context) { c.Set(auth.ContextUserID, owner) }, ReadSettingsHandler(svc))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/settings?user_id=other", nil))
		want := 401
		if owner != "" {
			want = 404
		}
		if w.Code != want {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
}

type failingPreferences struct{ Repo }

func (f failingPreferences) Preferences(context.Context, string) (Preferences, error) {
	return Preferences{}, errors.New("private database detail")
}
func TestReadSettingsHandlerSuccessCapabilityAndReadFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		log := &callLog{}
		repo := newFakeRepo(log)
		repo.prefs["owner"] = Preferences{"09:15:00", "Asia/Ho_Chi_Minh"}
		repo.prefs["other"] = Preferences{"22:00:00", "UTC"}
		var dependency Repo = repo
		if fail {
			dependency = failingPreferences{repo}
		}
		svc := NewService(dependency, nil, newFakeSender(log), nil, func() time.Time { return time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC) })
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/settings", func(c *gin.Context) { c.Set(auth.ContextUserID, "owner") }, ReadSettingsHandler(svc))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/settings?user_id=other", nil))
		if fail {
			if w.Code != 500 || w.Body.String() != `{"error":"internal_error"}` {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			continue
		}
		if w.Code != 200 || w.Body.String() != `{"notification_time":"09:15:00","timezone":"Asia/Ho_Chi_Minh","next_reminder_at":"2026-10-13T02:15:00Z","push_configured":true}` {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
		if len(log.calls) != 1 {
			t.Fatalf("unexpected side effects %v", log.calls)
		}
	}
}
