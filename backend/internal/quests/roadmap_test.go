package quests

import (
	"context"
	"encoding/json"
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
