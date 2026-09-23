package google

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Result is the backend spec §6.4 response for POST /integrations/google/sync.
type Result struct {
	Status            string `json:"status"`
	CalendarEventID   string `json:"calendar_event_id"`
	TasksCreatedCount int    `json:"tasks_created_count"`
}

// Service performs the one-way sync. Every dependency is an interface so the
// tests never reach Google or Postgres.
type Service struct {
	tokens RefreshTokenSource
	oauth  TokenRefresher
	cal    CalendarClient
	tasks  TasksClient
	repo   Repo
	now    func() time.Time
}

// NewService wires the dependencies. now is injected so event times are testable.
func NewService(tokens RefreshTokenSource, oauth TokenRefresher, cal CalendarClient, tasks TasksClient, repo Repo, now func() time.Time) *Service {
	return &Service{tokens: tokens, oauth: oauth, cal: cal, tasks: tasks, repo: repo, now: now}
}

// Sync pushes the recurring practice event and the per-day task list.
// Idempotency: the Calendar event is inserted with a deterministic client id
// (PracticeEventID), so a repeat insert is a 409 that is patched, and a
// stored id is patched (re-inserted only when Google lost it). The Tasks list
// is rebuilt only when the active roadmap changed. State is persisted after
// the event, after the list is created and after the task inserts.
//
// What that does NOT protect: a failure of SaveSyncState itself right after
// tasklists.insert (Tasks has no client-supplied id) orphans an empty list
// the retry cannot find — it deletes only the stored id. The Calendar half
// is covered by the deterministic id.
func (s *Service) Sync(ctx context.Context, userID string) (Result, error) {
	refresh, err := s.tokens.RefreshToken(ctx, userID)
	if errors.Is(err, ErrNoRefreshToken) {
		return Result{}, fmt.Errorf("%w: %v", ErrReauthRequired, err)
	}
	if err != nil {
		return Result{}, err
	}
	access, err := s.oauth.AccessToken(ctx, refresh)
	if err != nil {
		return Result{}, err
	}

	prof, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	state, err := s.repo.SyncState(ctx, userID)
	if errors.Is(err, ErrNoSyncState) {
		state = SyncState{UserID: userID}
	} else if err != nil {
		return Result{}, err
	}

	// 1. Calendar event: patch what we have, insert when we have nothing or
	// Google no longer has it.
	ev, err := PracticeEvent(s.now(), prof.NotificationTime, prof.Timezone)
	if err != nil {
		return Result{}, err
	}
	if state.CalendarEventID != "" {
		err = s.cal.PatchEvent(ctx, access, state.CalendarEventID, ev)
		if errors.Is(err, ErrNotFound) {
			state.CalendarEventID = ""
			err = nil
		}
		if err != nil {
			return Result{}, err
		}
	}
	if state.CalendarEventID == "" {
		// Client-supplied id: if a previous attempt inserted and then failed to
		// save (pool hiccup, client disconnect cancelling the request context),
		// this insert is a 409 and we patch our own event instead of adding a
		// second one. status: confirmed in the payload also restores an event
		// the user deleted (Google keeps it as cancelled with the id reserved).
		ev.ID = PracticeEventID(userID)
		id, err := s.cal.InsertEvent(ctx, access, ev)
		if errors.Is(err, ErrAlreadyExists) {
			id, err = ev.ID, s.cal.PatchEvent(ctx, access, ev.ID, ev)
			if errors.Is(err, ErrNotFound) {
				// The id is reserved but the event is gone for good (410): one
				// insert with a Google-assigned id — today's non-idempotent path,
				// reachable only after a user deleted the event by hand.
				ev.ID = ""
				id, err = s.cal.InsertEvent(ctx, access, ev)
			}
		}
		if err != nil {
			return Result{}, err
		}
		state.CalendarEventID = id
	}
	if err := s.repo.SaveSyncState(ctx, state); err != nil {
		return Result{}, err
	}

	// 2. Tasks list: one task per roadmap day, rebuilt only for a new roadmap.
	rm, err := s.repo.ActiveRoadmap(ctx, userID)
	if errors.Is(err, ErrNoActiveRoadmap) {
		return Result{Status: "synced", CalendarEventID: state.CalendarEventID, TasksCreatedCount: 0}, nil
	}
	if err != nil {
		return Result{}, err
	}
	if state.TasklistID != "" && state.RoadmapID == rm.ID {
		return Result{Status: "synced", CalendarEventID: state.CalendarEventID, TasksCreatedCount: state.TasksCreatedCount}, nil
	}
	if state.TasklistID != "" {
		// An older roadmap's list, or one whose tasks never finished: replace it.
		if err := s.tasks.DeleteTaskList(ctx, access, state.TasklistID); err != nil {
			return Result{}, err
		}
		state.TasklistID, state.RoadmapID, state.TasksCreatedCount = "", "", 0
	}
	listID, err := s.tasks.InsertTaskList(ctx, access, TasklistTitle)
	if err != nil {
		return Result{}, err
	}
	state.TasklistID = listID
	if err := s.repo.SaveSyncState(ctx, state); err != nil {
		return Result{}, err
	}

	days, err := s.repo.DayTitles(ctx, rm.ID)
	if err != nil {
		return Result{}, err
	}
	loc := Location(prof.Timezone)
	created := 0
	for _, d := range days {
		task := Task{
			Title: TaskTitle(d.Day, d.Titles),
			Notes: fmt.Sprintf("%d quests · 30 minutes. Open the app to start.", len(d.Titles)),
			Due:   DayDue(rm.CreatedAt, d.Day, loc),
		}
		if err := s.tasks.InsertTask(ctx, access, listID, task); err != nil {
			return Result{}, err
		}
		created++
	}
	state.RoadmapID, state.TasksCreatedCount = rm.ID, created
	if err := s.repo.SaveSyncState(ctx, state); err != nil {
		return Result{}, err
	}
	return Result{Status: "synced", CalendarEventID: state.CalendarEventID, TasksCreatedCount: created}, nil
}
