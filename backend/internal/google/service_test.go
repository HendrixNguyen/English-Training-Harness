package google

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSyncFirstTimeInsertsEventListAnd28Tasks(t *testing.T) {
	h := newHarness()

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.Status != "synced" || res.CalendarEventID != "evt_new" || res.TasksCreatedCount != 28 {
		t.Errorf("result = %+v", res)
	}
	if len(h.cal.inserted) != 1 || len(h.cal.patched) != 0 {
		t.Errorf("calendar inserts/patches = %d/%d, want 1/0", len(h.cal.inserted), len(h.cal.patched))
	}
	if got := h.cal.inserted[0].Start.Format("2006-01-02T15:04:05Z07:00"); got != "2026-09-22T20:00:00+07:00" {
		t.Errorf("event start = %s", got)
	}
	tasks := h.tasks.tasks["list_new"]
	if len(tasks) != 28 || tasks[0].Title != "Day 1: Vocab 1 · Read 1 · Practice 1" || tasks[27].Due.Format("2006-01-02") != "2026-09-29" {
		t.Errorf("tasks = %d, first %q, last due %v", len(tasks), tasks[0].Title, tasks[27].Due)
	}
	// Access token reached every Google call; the refresh token never did.
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "calendar.") || strings.HasPrefix(c, "tasks.") {
			if !strings.Contains(c, "access-for-1//refresh") {
				t.Errorf("call %q did not carry the access token", c)
			}
		}
	}
	final := h.repo.saved[len(h.repo.saved)-1]
	if final != (SyncState{UserID: "u1", CalendarEventID: "evt_new", TasklistID: "list_new", RoadmapID: "roadmap-A", TasksCreatedCount: 28}) {
		t.Errorf("final state = %+v", final)
	}
}

func TestSyncPersistsStateAfterTheEventAndAfterTheListBeforeTasks(t *testing.T) {
	h := newHarness()
	if _, err := h.svc.Sync(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	if len(h.repo.saved) != 3 {
		t.Fatalf("saved %d times, want 3 (after event, after list, final): %v", len(h.repo.saved), h.log.calls)
	}
	if s := h.repo.saved[0]; s.CalendarEventID != "evt_new" || s.TasklistID != "" {
		t.Errorf("first save = %+v, want only the event id", s)
	}
	if s := h.repo.saved[1]; s.TasklistID != "list_new" || s.RoadmapID != "" || s.TasksCreatedCount != 0 {
		t.Errorf("second save = %+v, want list id with no roadmap yet", s)
	}
	// Ordering: event saved before the list is created; list saved before any task.
	idx := func(prefix string) int {
		for i, c := range h.log.calls {
			if strings.HasPrefix(c, prefix) {
				return i
			}
		}
		return -1
	}
	if !(idx("repo.SaveSyncState(evt=evt_new,list=,") < idx("tasks.InsertTaskList") && idx("repo.SaveSyncState(evt=evt_new,list=list_new,roadmap=,") < idx("tasks.InsertTask(")) {
		t.Errorf("save points out of order: %v", h.log.calls)
	}
}

func TestSyncFailureMidTasksKeepsTheListIDSoRetryDeletesIt(t *testing.T) {
	h := newHarness()
	h.tasks.insertErr = &UpstreamError{Service: "tasks", Status: 500}
	h.tasks.failAfter = 5

	_, err := h.svc.Sync(context.Background(), "u1")
	var up *UpstreamError
	if !errors.As(err, &up) {
		t.Fatalf("err = %v, want *UpstreamError", err)
	}
	if h.repo.state.TasklistID != "list_new" || h.repo.state.RoadmapID != "" {
		t.Fatalf("state after failure = %+v, want list_new with no roadmap", h.repo.state)
	}

	// Retry: the half-built list is deleted and rebuilt, not appended to.
	h.tasks.insertErr = nil
	h.tasks.nextList = "list_retry"
	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(h.tasks.deleted) != 1 || h.tasks.deleted[0] != "list_new" || res.TasksCreatedCount != 28 || len(h.tasks.tasks["list_retry"]) != 28 {
		t.Errorf("deleted = %v, result = %+v", h.tasks.deleted, res)
	}
}

func TestResyncSameRoadmapPatchesEventAndCreatesNothing(t *testing.T) {
	h := newHarness()
	h.repo.noState = false
	h.repo.state = SyncState{UserID: "u1", CalendarEventID: "evt_old", TasklistID: "list_old", RoadmapID: "roadmap-A", TasksCreatedCount: 28}

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.CalendarEventID != "evt_old" || res.TasksCreatedCount != 28 {
		t.Errorf("result = %+v", res)
	}
	if len(h.cal.inserted) != 0 || len(h.cal.patched) != 1 || h.cal.patched[0] != "evt_old" {
		t.Errorf("calendar inserts/patches = %v/%v", h.cal.inserted, h.cal.patched)
	}
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "tasks.") {
			t.Errorf("same roadmap must not touch Tasks, but called %s", c)
		}
	}
}

func TestResyncNewRoadmapDeletesOldListAndBuildsANewOne(t *testing.T) {
	h := newHarness()
	h.repo.noState = false
	h.repo.state = SyncState{UserID: "u1", CalendarEventID: "evt_old", TasklistID: "list_old", RoadmapID: "roadmap-OLD", TasksCreatedCount: 28}

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(h.tasks.deleted) != 1 || h.tasks.deleted[0] != "list_old" {
		t.Errorf("deleted = %v, want [list_old]", h.tasks.deleted)
	}
	if res.TasksCreatedCount != 28 || h.repo.state.TasklistID != "list_new" || h.repo.state.RoadmapID != "roadmap-A" {
		t.Errorf("result = %+v, state = %+v", res, h.repo.state)
	}
}

func TestResyncReinsertsTheEventWhenGoogleLostIt(t *testing.T) {
	h := newHarness()
	h.repo.noState = false
	h.repo.state = SyncState{UserID: "u1", CalendarEventID: "evt_deleted_by_user", TasklistID: "list_old", RoadmapID: "roadmap-A", TasksCreatedCount: 28}
	h.cal.patchErr = ErrNotFound

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.CalendarEventID != "evt_new" || len(h.cal.inserted) != 1 {
		t.Errorf("result = %+v, inserted = %d", res, len(h.cal.inserted))
	}
}

func TestSyncWithoutARoadmapPushesOnlyTheEvent(t *testing.T) {
	h := newHarness()
	h.repo.noRoad = true

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.Status != "synced" || res.CalendarEventID != "evt_new" || res.TasksCreatedCount != 0 {
		t.Errorf("result = %+v", res)
	}
	for _, c := range h.log.calls {
		if strings.HasPrefix(c, "tasks.") {
			t.Errorf("no roadmap must not touch Tasks, but called %s", c)
		}
	}
}

func TestSyncNeedsReauthWithoutARefreshTokenOrOnInvalidGrant(t *testing.T) {
	h := newHarness()
	h.tokens.err = ErrNoRefreshToken
	if _, err := h.svc.Sync(context.Background(), "u1"); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("no refresh token: err = %v, want ErrReauthRequired", err)
	}
	if len(h.cal.inserted) != 0 {
		t.Error("nothing may reach Google without a token")
	}

	h = newHarness()
	h.oauth.err = ErrReauthRequired // what OAuthClient returns on invalid_grant
	if _, err := h.svc.Sync(context.Background(), "u1"); !errors.Is(err, ErrReauthRequired) {
		t.Errorf("invalid_grant: err = %v, want ErrReauthRequired", err)
	}
	if len(h.repo.saved) != 0 {
		t.Error("no state may be written when the token refresh fails")
	}
}
