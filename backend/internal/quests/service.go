package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

// ErrInvalidDuration means duration_seconds is outside 1..MaxDurationSeconds
// or would push the day past MaxDailySeconds. The handler maps it to 400.
var ErrInvalidDuration = errors.New("quests: invalid duration_seconds")

// ProgressResult is the POST /api/v1/quests/progress 200 body — backend spec
// §6.2, field for field.
type ProgressResult struct {
	DailySecondsSpent int64 `json:"daily_seconds_spent"`
	DailyMinutesSpent int   `json:"daily_minutes_spent"`
	IsTargetMet       bool  `json:"is_target_met"`
	PetHealth         int   `json:"pet_health"`
	StreakCount       int   `json:"streak_count"`

	// NewlyMet is true only on a call that fired Pet.OnTargetMet — the total is
	// at or past TargetSeconds and daily_progress did not yet say the pet was
	// told. It is asserted by tests; §6.2 has no such field, so it never
	// reaches the wire.
	NewlyMet bool `json:"-"`
}

// Service implements the daily loop (1st-thinking §5.2; backend spec §6.2).
type Service struct {
	counter  Counter
	quests   QuestRepo
	progress ProgressRepo
	pet      Pet
	now      func() time.Time
}

// NewService wires the collaborators. now is injectable so day boundaries are
// testable; pet may be NopPet{} until the pet slice registers the real one.
func NewService(counter Counter, quests QuestRepo, progress ProgressRepo, pet Pet, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	if pet == nil {
		pet = NopPet{}
	}
	return &Service{counter: counter, quests: quests, progress: progress, pet: pet, now: now}
}

// RecordProgress validates, then implements §5.2 steps 2-4 in that order:
//
//  0. Reads only — resolve the profile and active roadmap, then CheckExercise:
//     the exercise must be on the caller's active roadmap and on today's
//     day_number, or the call is rejected with ErrExerciseNotFound before a
//     single write. A rejected request leaves Redis and daily_progress
//     untouched (reviewer 2026-09-22). Then read the counter and reject with
//     ErrInvalidDuration if seconds is outside 1..MaxDurationSeconds or the
//     day would pass MaxDailySeconds.
//  1. INCRBY the Redis counter (+ EXPIRE) and read the running total back.
//  2. Upsert daily_progress.minutes_spent from that total (never lowering it)
//     and learn whether the day's is_target_met is already set.
//  3. If the total is at or past 1800s and the day is not yet flagged: fire
//     Pet.OnTargetMet, and only when it returns nil flag the day with
//     MarkTargetMet. A hook failure leaves the day unflagged so the next
//     progress call fires again; pet's own last_target_met_date makes a
//     re-fire a no-op once it has landed (at-least-once here, exactly-once
//     there). A failed flag is logged, not returned: the session is recorded
//     and a 500 would make the client retry and INCRBY again.
//  4. Mark the exercise complete, then read Pet.State so the §6.2 response
//     carries pet_health / streak_count.
//
// The ordering is a contract, not an implementation detail: see the call-log
// test in service_test.go.
func (s *Service) RecordProgress(ctx context.Context, userID, exerciseID string, seconds int64) (ProgressResult, error) {
	if seconds <= 0 || seconds > MaxDurationSeconds {
		return ProgressResult{}, fmt.Errorf("%w: must be in 1..%d, got %d", ErrInvalidDuration, MaxDurationSeconds, seconds)
	}

	profile, err := s.quests.Profile(ctx, userID)
	if err != nil {
		return ProgressResult{}, err
	}
	roadmap, err := s.quests.ActiveRoadmap(ctx, userID)
	if err != nil {
		return ProgressResult{}, err
	}

	loc := Location(profile.Timezone)
	now := s.now()
	date := LocalDate(now, loc)
	day := DayNumber(roadmap.CreatedAt, now, loc)

	if err := s.quests.CheckExercise(ctx, roadmap.ID, exerciseID, day); err != nil {
		return ProgressResult{}, err
	}

	// Still a read: the running total decides whether this report fits under
	// the daily ceiling. Only then does the §5.2 write sequence start.
	total, err := s.counter.Total(ctx, userID, date)
	if err != nil {
		return ProgressResult{}, err
	}
	if total+seconds > MaxDailySeconds {
		return ProgressResult{}, fmt.Errorf("%w: %d + %d would exceed the daily ceiling of %d", ErrInvalidDuration, total, seconds, MaxDailySeconds)
	}

	total, err = s.counter.Add(ctx, userID, date, seconds)
	if err != nil {
		return ProgressResult{}, err
	}

	alreadyMet, err := s.progress.Upsert(ctx, userID, date, int(total/60))
	if err != nil {
		return ProgressResult{}, err
	}
	// The durable row wins over the counter for the response: a counter lost
	// mid-day does not un-meet a met day.
	targetMet := total >= TargetSeconds || alreadyMet
	newlyMet := total >= TargetSeconds && !alreadyMet

	if newlyMet {
		if err := s.pet.OnTargetMet(ctx, userID, date); err != nil {
			// Best-effort for this call: the study session is already recorded
			// and must not be rolled back by a pet failure. The day stays
			// unflagged, so the next call retries the hook.
			log.Printf("quests: pet target-met hook failed for user %s on %s (will retry on the next progress call): %v", userID, date, err)
		} else if err := s.progress.MarkTargetMet(ctx, userID, date); err != nil {
			log.Printf("quests: flagging daily_progress.is_target_met for user %s on %s: %v", userID, date, err)
		}
	}

	// MarkComplete keeps its own ErrExerciseNotFound for the race where the
	// row vanished between CheckExercise and here; the handler still maps it
	// to 404. It runs after the hook so a vanished exercise cannot cost the
	// user the day's +20.
	if err := s.quests.MarkComplete(ctx, roadmap.ID, exerciseID); err != nil {
		return ProgressResult{}, err
	}

	// Also best-effort: the write is done, and a 500 here would make the client
	// retry and double-count. GET /pet/status (pet slice) is the authoritative read.
	pet, err := s.pet.State(ctx, userID)
	if err != nil {
		log.Printf("quests: reading pet state for user %s: %v", userID, err)
		pet = PetState{}
	}

	return ProgressResult{
		DailySecondsSpent: total,
		DailyMinutesSpent: int(total / 60),
		IsTargetMet:       targetMet,
		PetHealth:         pet.Health,
		StreakCount:       pet.Streak,
		NewlyMet:          newlyMet,
	}, nil
}

// DefaultTaskMinutes is the §6.2 task length ("3x 10-min tasks"), used when an
// exercise's content_json carries no duration_minutes.
const DefaultTaskMinutes = 10

// Task is one entry of the GET /api/v1/quests/daily `tasks` array (backend spec
// §6.2). title and duration_minutes are not §3.2 columns; toTask reads them
// from content_json.
type Task struct {
	ID              string          `json:"id"`
	TaskType        string          `json:"task_type"`
	Title           string          `json:"title"`
	DurationMinutes int             `json:"duration_minutes"`
	IsCompleted     bool            `json:"is_completed"`
	ContentJSON     json.RawMessage `json:"content_json"`
}

// DailySuite is the GET /api/v1/quests/daily 200 body — backend spec §6.2,
// field for field.
type DailySuite struct {
	Date                 string `json:"date"`
	DayNumber            int    `json:"day_number"`
	TotalMinutesRequired int    `json:"total_minutes_required"`
	AccumulatedSeconds   int64  `json:"accumulated_seconds"`
	IsTargetMet          bool   `json:"is_target_met"`
	Tasks                []Task `json:"tasks"`
}

// toTask maps a §3.2 exercises row onto the §6.2 task DTO. A missing title is
// "", a missing or non-positive duration is DefaultTaskMinutes. Malformed
// content_json is the generator's bug, not a reason to 500 the whole day, so the
// unmarshal error is deliberately ignored and the defaults apply.
func toTask(e Exercise) Task {
	var meta struct {
		Title           string `json:"title"`
		DurationMinutes int    `json:"duration_minutes"`
	}
	_ = json.Unmarshal(e.ContentJSON, &meta)
	if meta.DurationMinutes <= 0 {
		meta.DurationMinutes = DefaultTaskMinutes
	}
	return Task{
		ID:              e.ID,
		TaskType:        e.TaskType,
		Title:           meta.Title,
		DurationMinutes: meta.DurationMinutes,
		IsCompleted:     e.IsCompleted,
		ContentJSON:     e.ContentJSON,
	}
}

// Daily resolves the active roadmap, computes today's day_number in the user's
// timezone and returns that day's tasks, today's running total and whether the
// day is met (counter or durable flag).
func (s *Service) Daily(ctx context.Context, userID string) (DailySuite, error) {
	profile, err := s.quests.Profile(ctx, userID)
	if err != nil {
		return DailySuite{}, err
	}
	roadmap, err := s.quests.ActiveRoadmap(ctx, userID)
	if err != nil {
		return DailySuite{}, err
	}

	loc := Location(profile.Timezone)
	now := s.now()
	date := LocalDate(now, loc)
	day := DayNumber(roadmap.CreatedAt, now, loc)

	exercises, err := s.quests.ExercisesForDay(ctx, roadmap.ID, day)
	if err != nil {
		return DailySuite{}, err
	}
	total, err := s.counter.Total(ctx, userID, date)
	if err != nil {
		return DailySuite{}, err
	}
	flagged, err := s.progress.TargetMet(ctx, userID, date)
	if err != nil {
		return DailySuite{}, err
	}

	tasks := make([]Task, 0, len(exercises)) // never nil: serialises as []
	for _, e := range exercises {
		tasks = append(tasks, toTask(e))
	}

	return DailySuite{
		Date:                 date,
		DayNumber:            day,
		TotalMinutesRequired: TargetSeconds / 60,
		AccumulatedSeconds:   total,
		// The durable row wins over a lost counter — the same rule
		// RecordProgress answers with, so the two endpoints never disagree
		// about one local day. AccumulatedSeconds stays live (the progress bar).
		IsTargetMet: total >= TargetSeconds || flagged,
		Tasks:       tasks,
	}, nil
}
