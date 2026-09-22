package quests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
)

// newQuestRouter mounts the routes with a stub that injects the authenticated
// user under auth.ContextUserID, standing in for auth.Require() (covered in the
// auth slice, middleware_test.go).
func newQuestRouter(svc *Service, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	inject := func(c *gin.Context) {
		c.Set(auth.ContextUserID, userID)
		c.Next()
	}
	g := r.Group("/api/v1", inject)
	g.GET("/quests/daily", DailyHandler(svc))
	g.POST("/quests/progress", ProgressHandler(svc))
	return r
}

func postJSON(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDailyHandlerReturnsTheSpec62Body(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/quests/daily", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	// Exactly the backend spec §6.2 GET /quests/daily shape.
	var body struct {
		Date                 string `json:"date"`
		DayNumber            int    `json:"day_number"`
		TotalMinutesRequired int    `json:"total_minutes_required"`
		AccumulatedSeconds   int64  `json:"accumulated_seconds"`
		IsTargetMet          bool   `json:"is_target_met"`
		Tasks                []struct {
			ID              string          `json:"id"`
			TaskType        string          `json:"task_type"`
			Title           string          `json:"title"`
			DurationMinutes int             `json:"duration_minutes"`
			IsCompleted     bool            `json:"is_completed"`
			ContentJSON     json.RawMessage `json:"content_json"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding: %v (%s)", err, w.Body.String())
	}
	if body.Date != "2026-09-22" || body.DayNumber != 2 || body.TotalMinutesRequired != 30 || len(body.Tasks) != 3 {
		t.Errorf("body = %+v", body)
	}
	if body.Tasks[0].DurationMinutes != DefaultTaskMinutes || len(body.Tasks[0].ContentJSON) == 0 {
		t.Errorf("task[0] = %+v, want duration_minutes=%d and content_json present", body.Tasks[0], DefaultTaskMinutes)
	}
	for _, stale := range []string{`"exercises"`, `"total_seconds"`, `"target_met"`} {
		if strings.Contains(w.Body.String(), stale) {
			t.Errorf("body still uses pre-reconciliation field %s: %s", stale, w.Body.String())
		}
	}
}

func TestDailyHandlerReturns404WithoutARoadmap(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	h.quests.roadmap = nil

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/quests/daily", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"error":"no_active_roadmap"`) {
		t.Errorf("body = %s, want the no_active_roadmap error", w.Body.String())
	}
}

func TestProgressHandlerReturnsTheSpec62Body(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	h := newHarness(t, now)

	// The full §6.2 request, user_answers included.
	w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress",
		`{"exercise_id":"ex-2-reading","duration_seconds":1800,"user_answers":{"q1":"A"}}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	for _, want := range []string{
		`"daily_seconds_spent":1800`, `"daily_minutes_spent":30`, `"is_target_met":true`,
		`"pet_health":100`, `"streak_count":5`,
	} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("body = %s, missing %s", w.Body.String(), want)
		}
	}
	for _, stale := range []string{`newly_met`, `"total_seconds"`, `"target_met"`} {
		if strings.Contains(w.Body.String(), stale) {
			t.Errorf("body leaks non-§6.2 field %s: %s", stale, w.Body.String())
		}
	}
}

func TestProgressHandlerAcceptsABodyWithoutUserAnswers(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))

	w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress",
		`{"exercise_id":"ex-2-reading","duration_seconds":600}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — user_answers is optional; body = %s", w.Code, w.Body.String())
	}
}

func TestProgressHandlerRejectsABadBody(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	r := newQuestRouter(h.svc, "u1")

	for _, body := range []string{
		`{}`,
		`{"exercise_id":"ex-2-reading"}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":0}`,
		`{"exercise_id":"ex-2-reading","duration_seconds":-5}`,
		`{"exercise_id":"ex-2-reading","seconds":600}`, // the pre-§6.2 field name is not an alias
		`nonsense`,
	} {
		if w := postJSON(r, "/api/v1/quests/progress", body); w.Code != http.StatusBadRequest {
			t.Errorf("body %q → status %d, want 400", body, w.Code)
		}
	}
}

func TestProgressHandlerReturns404ForAnUnknownExercise(t *testing.T) {
	h := newHarness(t, time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC))
	h.quests.markErr = ErrExerciseNotFound

	w := postJSON(newQuestRouter(h.svc, "u1"), "/api/v1/quests/progress", `{"exercise_id":"nope","duration_seconds":600}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"error":"exercise_not_found"`) {
		t.Errorf("body = %s, want the exercise_not_found error", w.Body.String())
	}
}
