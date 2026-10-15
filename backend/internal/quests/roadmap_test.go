package quests

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type outlineFake struct {
	snapshot RoadmapSnapshot
	err      error
	owner    string
}

func (f *outlineFake) ReadRoadmap(_ context.Context, owner string) (RoadmapSnapshot, error) {
	f.owner = owner
	return f.snapshot, f.err
}
func TestRoadmapCalendarAndRecordedProgress(t *testing.T) {
	for _, tc := range []struct {
		zone, start, now string
		day              int
		today            string
	}{
		{"America/New_York", "2026-03-07T15:00:00Z", "2026-03-09T04:30:00Z", 3, "2026-03-09"},
		{"America/New_York", "2026-10-31T15:00:00Z", "2026-11-02T05:30:00Z", 3, "2026-11-02"},
		{"Asia/Ho_Chi_Minh", "2026-09-01T18:00:00Z", "2026-10-01T18:00:00Z", 31, "2026-10-02"},
	} {
		t.Run(tc.now, func(t *testing.T) {
			start, _ := time.Parse(time.RFC3339, tc.start)
			now, _ := time.Parse(time.RFC3339, tc.now)
			raw, _ := json.Marshal(integrationRoadmap())
			yes := true
			no := false
			f := &outlineFake{snapshot: RoadmapSnapshot{ID: "owned", CreatedAt: start, Timezone: tc.zone, Outline: raw, Tasks: []RoadmapStoredTask{{DayNumber: 1, RoadmapTask: RoadmapTask{ID: "done", IsCompleted: &yes}}, {DayNumber: 2, RoadmapTask: RoadmapTask{ID: "pending", IsCompleted: &no}}}, Progress: map[string]RoadmapProgress{}}}
			f.snapshot.Progress[LocalDate(start, Location(tc.zone))] = RoadmapProgress{MinutesSpent: 30, IsTargetMet: true}
			f.snapshot.Progress[start.In(Location(tc.zone)).AddDate(0, 0, 1).Format("2006-01-02")] = RoadmapProgress{MinutesSpent: 12}
			out, err := NewRoadmapService(f, func() time.Time { return now }).Read(context.Background(), "learner")
			if err != nil {
				t.Fatal(err)
			}
			if f.owner != "learner" || out.DayNumber != tc.day || out.Today != tc.today || len(out.Modules) != 4 {
				t.Fatalf("%+v owner=%s", out, f.owner)
			}
			d := out.Modules[0].Days
			if len(d) != 7 || !d[0].IsTargetMet || d[0].MinutesSpent != 30 || d[1].IsTargetMet || d[1].MinutesSpent != 12 || d[2].IsTargetMet || d[2].MinutesSpent != 0 {
				t.Fatalf("days=%+v", d)
			}
			if d[0].Tasks[0].IsCompleted == nil || !*d[0].Tasks[0].IsCompleted || d[1].Tasks[0].IsCompleted == nil || *d[1].Tasks[0].IsCompleted {
				t.Fatal("completion not recorded")
			}
		})
	}
}
func TestRoadmapHandlerErrorsAndOwner(t *testing.T) {
	for _, tc := range []struct {
		owner  string
		err    error
		status int
		code   string
	}{{"", nil, 401, "unauthorized"}, {"u", ErrNoActiveRoadmap, 404, "no_active_roadmap"}, {"u", errors.New("db down"), 500, "internal_error"}} {
		f := &outlineFake{err: tc.err}
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/api/v1/roadmap", func(c *gin.Context) { c.Set(auth.ContextUserID, tc.owner) }, RoadmapHandler(NewRoadmapService(f, time.Now)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/roadmap?user_id=other", nil))
		if w.Code != tc.status || w.Body.String() != `{"error":"`+tc.code+`"}` {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
		if tc.owner != "" && f.owner != tc.owner {
			t.Fatal("wrong owner")
		}
	}
}

func TestRoadmapHandlerMetadataOnlyAndUnknownCompletion(t *testing.T) {
	raw := json.RawMessage(`{"title":"Real plan","cefr_level":"B1","modules":[{"week":1,"title":"Real week","focus":"Real focus","days":[{"title":"Real day","tasks":[{"content":{"answer_key":"NEVER_EXPOSE"}}]},{"title":"2"},{"title":"3"},{"title":"4"},{"title":"5"},{"title":"6"},{"title":"7"}]},{"week":2,"days":[{},{},{},{},{},{},{}]},{"week":3,"days":[{},{},{},{},{},{},{}]},{"week":4,"days":[{},{},{},{},{},{},{}]}]}`)
	f := &outlineFake{snapshot: RoadmapSnapshot{ID: "owned", CreatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Timezone: "UTC", Outline: raw, Tasks: []RoadmapStoredTask{{DayNumber: 1, RoadmapTask: RoadmapTask{ID: "real-task", TaskType: "reading", Title: "Read", DurationMinutes: 10}}}}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/roadmap", func(c *gin.Context) { c.Set(auth.ContextUserID, "owner") }, RoadmapHandler(NewRoadmapService(f, func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) })))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/roadmap", nil))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var got RoadmapOutline
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.DayNumber != 32 || got.Modules[0].Days[0].Tasks[0].IsCompleted != nil || got.Modules[0].Title != "Real week" || got.Modules[0].Focus != "Real focus" {
		t.Fatalf("%+v", got)
	}
	var body map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body) != 6 {
		t.Fatalf("unexpected wire fields %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "NEVER_EXPOSE") || strings.Contains(w.Body.String(), "content") || strings.Contains(w.Body.String(), "answer_key") {
		t.Fatal("content leaked")
	}
}
