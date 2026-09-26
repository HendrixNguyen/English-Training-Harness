package quests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// outlineJSON marshals the 4x7x3 fixture integration_test.go already builds
// (integrationRoadmap), so fake-backed tests exercise the same shape the
// integration test seeds through onboarding.
func outlineJSON(t *testing.T) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(integrationRoadmap())
	if err != nil {
		t.Fatalf("marshal integrationRoadmap: %v", err)
	}
	return b
}

// newRoadmapHarness is newHarness plus a roadmap_json document on the fake
// quest repo, so Service.Roadmap has something to unmarshal.
func newRoadmapHarness(t *testing.T, now time.Time) *harness {
	t.Helper()
	h := newHarness(t, now)
	h.quests.roadmapJSON = outlineJSON(t)
	return h
}

func TestFakeProgressBetweenFiltersByDate(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))
	h.progress.rows["u1|2026-09-01"] = struct {
		minutes   int
		targetMet bool
	}{30, true}
	h.progress.rows["u1|2026-09-03"] = struct {
		minutes   int
		targetMet bool
	}{12, false}
	h.progress.rows["u1|2026-10-30"] = struct {
		minutes   int
		targetMet bool
	}{5, false}
	h.progress.rows["u2|2026-09-02"] = struct {
		minutes   int
		targetMet bool
	}{9, false}

	rows, err := h.progress.ProgressBetween(context.Background(), "u1", "2026-09-01", "2026-09-28")
	if err != nil {
		t.Fatalf("ProgressBetween: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 (rows = %+v)", len(rows), rows)
	}
	if rows["2026-09-01"] != (DayProgress{MinutesSpent: 30, IsTargetMet: true}) {
		t.Errorf("rows[2026-09-01] = %+v", rows["2026-09-01"])
	}
	if rows["2026-09-03"] != (DayProgress{MinutesSpent: 12, IsTargetMet: false}) {
		t.Errorf("rows[2026-09-03] = %+v", rows["2026-09-03"])
	}
}

func TestRoadmapReturnsFourModulesOfSevenNamedDaysWithDates(t *testing.T) {
	now := time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC)
	h := newRoadmapHarness(t, now)
	created := time.Date(2026, time.September, 1, 22, 0, 0, 0, time.UTC)
	h.quests.roadmap.CreatedAt = created

	out, err := h.svc.Roadmap(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Roadmap: %v", err)
	}
	if out.RoadmapID != "rm-1" {
		t.Errorf("RoadmapID = %q, want rm-1", out.RoadmapID)
	}
	if out.Title != "Integration" || out.CEFRLevel != "B1" {
		t.Errorf("Title/CEFRLevel = %q/%q, want Integration/B1", out.Title, out.CEFRLevel)
	}
	if !out.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", out.CreatedAt, created)
	}
	if out.DayNumber != 10 {
		t.Errorf("DayNumber = %d, want 10", out.DayNumber)
	}
	if len(out.Modules) != 4 {
		t.Fatalf("len(Modules) = %d, want 4", len(out.Modules))
	}
	if out.Modules[1].Week != 2 || out.Modules[1].Title != "Week 2" || out.Modules[1].Focus != "integration" {
		t.Errorf("Modules[1] = %+v, want week=2 title=Week 2 focus=integration", out.Modules[1])
	}
	d0 := out.Modules[0].Days[0]
	if d0.DayNumber != 1 || d0.Date != "2026-09-01" || d0.Title != "Day 1" {
		t.Errorf("Modules[0].Days[0] = %+v, want day=1 date=2026-09-01 title=Day 1", d0)
	}
	if out.Modules[0].Days[1].Date != "2026-09-02" {
		t.Errorf("Modules[0].Days[1].Date = %q, want 2026-09-02", out.Modules[0].Days[1].Date)
	}
	last := out.Modules[3].Days[6]
	if last.DayNumber != 28 || last.Date != "2026-09-28" {
		t.Errorf("Modules[3].Days[6] = %+v, want day=28 date=2026-09-28", last)
	}
	for _, m := range out.Modules {
		for _, d := range m.Days {
			if len(d.Tasks) != 3 {
				t.Fatalf("day %d has %d tasks, want 3", d.DayNumber, len(d.Tasks))
			}
			if d.Tasks[0] != (RoadmapTask{TaskType: "vocabulary", Title: "Day task: vocabulary", DurationMinutes: 10}) {
				t.Errorf("day %d tasks[0] = %+v, want the stored vocabulary task", d.DayNumber, d.Tasks[0])
			}
		}
	}
}

func TestRoadmapJoinsDailyProgressByLocalDate(t *testing.T) {
	now := time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC)
	h := newRoadmapHarness(t, now)
	created := time.Date(2026, time.September, 1, 22, 0, 0, 0, time.UTC)
	h.quests.roadmap.CreatedAt = created

	set := func(user, date string, minutes int, met bool) {
		h.progress.rows[user+"|"+date] = struct {
			minutes   int
			targetMet bool
		}{minutes, met}
	}
	set("u1", "2026-09-01", 32, true)
	set("u1", "2026-09-03", 12, false)
	set("u1", "2026-09-10", 10, false)
	set("u1", "2026-08-30", 7, true)  // before the roadmap: must be ignored
	set("u2", "2026-09-02", 30, true) // another user

	out, err := h.svc.Roadmap(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Roadmap: %v", err)
	}
	byDay := map[int]RoadmapDay{}
	metCount := 0
	for _, m := range out.Modules {
		for _, d := range m.Days {
			byDay[d.DayNumber] = d
			if d.IsTargetMet {
				metCount++
			}
		}
	}
	wantMinutes := map[int]int{1: 32, 2: 0, 3: 12, 10: 10, 11: 0}
	for day, want := range wantMinutes {
		if got := byDay[day].MinutesSpent; got != want {
			t.Errorf("day %d minutes = %d, want %d", day, got, want)
		}
	}
	if metCount != 1 || !byDay[1].IsTargetMet {
		t.Errorf("exactly one day should be met (day 1); got %d met, day1=%+v", metCount, byDay[1])
	}
}

func TestRoadmapDatesFollowTheUsersTimezone(t *testing.T) {
	created := time.Date(2026, time.September, 1, 18, 30, 0, 0, time.UTC) // 09-02 01:30 local
	now := time.Date(2026, time.September, 1, 19, 0, 0, 0, time.UTC)
	h := newRoadmapHarness(t, now)
	h.quests.timezone = "Asia/Ho_Chi_Minh"
	h.quests.roadmap.CreatedAt = created
	h.progress.rows["u1|2026-09-02"] = struct {
		minutes   int
		targetMet bool
	}{5, false}

	out, err := h.svc.Roadmap(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Roadmap: %v", err)
	}
	if out.DayNumber != 1 {
		t.Errorf("DayNumber = %d, want 1", out.DayNumber)
	}
	d0 := out.Modules[0].Days[0]
	if d0.Date != "2026-09-02" {
		t.Errorf("Days[0].Date = %q, want 2026-09-02", d0.Date)
	}
	if d0.MinutesSpent != 5 {
		t.Errorf("Days[0].MinutesSpent = %d, want 5 (keyed by the local date)", d0.MinutesSpent)
	}
}

func TestRoadmapDatesAgreeWithDayNumberAcrossDST(t *testing.T) {
	loc := Location("Europe/London")
	if loc == time.UTC {
		t.Fatalf("Location(%q) fell back to UTC — tzdata missing on this machine", "Europe/London")
	}
	at := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 9, 0, 0, 0, loc) }
	created := at(2026, time.March, 25)
	now := at(2026, time.April, 21)

	h := newRoadmapHarness(t, now)
	h.quests.timezone = "Europe/London"
	h.quests.roadmap.CreatedAt = created

	out, err := h.svc.Roadmap(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Roadmap: %v", err)
	}
	if out.DayNumber != 28 {
		t.Errorf("DayNumber = %d, want 28", out.DayNumber)
	}
	if got := out.Modules[3].Days[6].Date; got != "2026-04-21" {
		t.Errorf("Modules[3].Days[6].Date = %q, want 2026-04-21", got)
	}
	if got := out.Modules[0].Days[4].Date; got != "2026-03-29" {
		t.Errorf("Modules[0].Days[4].Date = %q, want 2026-03-29 (the spring-forward day is day 5)", got)
	}
}

func TestRoadmapDefaultsAMissingDuration(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))
	r := integrationRoadmap()
	r.Modules[0].Days[0].Tasks[0].DurationMinutes = 0
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	h.quests.roadmapJSON = b

	out, err := h.svc.Roadmap(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Roadmap: %v", err)
	}
	if got := out.Modules[0].Days[0].Tasks[0].DurationMinutes; got != DefaultTaskMinutes {
		t.Errorf("DurationMinutes = %d, want %d", got, DefaultTaskMinutes)
	}
}

func TestRoadmapWithoutAnActiveRoadmap(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))
	h.quests.roadmap = nil

	_, err := h.svc.Roadmap(context.Background(), "u1")
	if !errors.Is(err, ErrNoActiveRoadmap) {
		t.Errorf("err = %v, want ErrNoActiveRoadmap", err)
	}
}

func TestRoadmapWithUnreadableJSONFails(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))
	h.quests.roadmapJSON = json.RawMessage("{not json")

	_, err := h.svc.Roadmap(context.Background(), "u1")
	if err == nil {
		t.Fatal("err = nil, want an error")
	}
	if !strings.Contains(err.Error(), "roadmap_json") {
		t.Errorf("err = %v, want it to mention roadmap_json", err)
	}
}

func TestRoadmapWithTheWrongShapeFails(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))
	h.quests.roadmapJSON = json.RawMessage(`{"title":"x","cefr_level":"B1","modules":[]}`)

	_, err := h.svc.Roadmap(context.Background(), "u1")
	if err == nil {
		t.Fatal("err = nil, want an error")
	}
	if !strings.Contains(err.Error(), "modules") {
		t.Errorf("err = %v, want it to mention modules", err)
	}
}

func TestRoadmapHandlerReturnsTheSpec62Body(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/roadmap", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding: %v (%s)", err, w.Body.String())
	}
	wantTopKeys := []string{"roadmap_id", "title", "cefr_level", "created_at", "day_number", "modules"}
	if len(body) != len(wantTopKeys) {
		t.Errorf("top-level keys = %v, want exactly %v", keysOf(body), wantTopKeys)
	}
	for _, k := range wantTopKeys {
		if _, ok := body[k]; !ok {
			t.Errorf("missing top-level key %q", k)
		}
	}
	modules, ok := body["modules"].([]any)
	if !ok || len(modules) != 4 {
		t.Fatalf("modules = %+v, want 4 entries", body["modules"])
	}
	wantModuleKeys := []string{"week", "title", "focus", "days"}
	wantDayKeys := []string{"day_number", "date", "title", "tasks", "minutes_spent", "is_target_met"}
	wantTaskKeys := []string{"task_type", "title", "duration_minutes"}
	for _, mRaw := range modules {
		m := mRaw.(map[string]any)
		if len(m) != len(wantModuleKeys) {
			t.Errorf("module keys = %v, want %v", keysOf(m), wantModuleKeys)
		}
		days, ok := m["days"].([]any)
		if !ok || len(days) != 7 {
			t.Fatalf("days = %+v, want 7 entries", m["days"])
		}
		for _, dRaw := range days {
			d := dRaw.(map[string]any)
			if len(d) != len(wantDayKeys) {
				t.Errorf("day keys = %v, want %v", keysOf(d), wantDayKeys)
			}
			tasks, ok := d["tasks"].([]any)
			if !ok || len(tasks) != 3 {
				t.Fatalf("tasks = %+v, want 3 entries", d["tasks"])
			}
			for _, tRaw := range tasks {
				task := tRaw.(map[string]any)
				if len(task) != len(wantTaskKeys) {
					t.Errorf("task keys = %v, want %v", keysOf(task), wantTaskKeys)
				}
			}
		}
	}
	if strings.Contains(w.Body.String(), `"content`) {
		t.Errorf("body leaks exercise content: %s", w.Body.String())
	}
}

// keysOf is a test-only helper for readable failure messages.
func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestRoadmapHandlerReturns404WithoutARoadmap(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))
	h.quests.roadmap = nil

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/roadmap", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"error":"no_active_roadmap"`) {
		t.Errorf("body = %s, want the no_active_roadmap error", w.Body.String())
	}
}

func TestRoadmapHandlerReturns500OnAnUnreadableDocument(t *testing.T) {
	h := newRoadmapHarness(t, time.Date(2026, time.September, 10, 10, 0, 0, 0, time.UTC))
	h.quests.roadmapJSON = json.RawMessage(`{`)

	w := httptest.NewRecorder()
	newQuestRouter(h.svc, "u1").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/roadmap", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"error":"internal_error"`) {
		t.Errorf("body = %s, want the internal_error error", w.Body.String())
	}
}
