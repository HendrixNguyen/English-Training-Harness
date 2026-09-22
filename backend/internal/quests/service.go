package quests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ProgressResult is the POST /api/v1/quests/progress 200 body — backend spec
// §6.2, field for field.
type ProgressResult struct {
	DailySecondsSpent int64 `json:"daily_seconds_spent"`
	DailyMinutesSpent int   `json:"daily_minutes_spent"`
	IsTargetMet       bool  `json:"is_target_met"`
	PetHealth         int   `json:"pet_health"`
	StreakCount       int   `json:"streak_count"`

	// NewlyMet is true only on the call that crossed TargetSeconds. It is the
	// once-only trigger for Pet.OnTargetMet and is asserted by tests; §6.2 has
	// no such field, so it never reaches the wire.
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
//     untouched (reviewer 2026-09-22).
//  1. INCRBY the Redis counter (+ EXPIRE) and read the running total back.
//  2. Upsert daily_progress from that total — Redis is the single source of
//     truth for the day, so bursts of calls cannot disagree.
//  3. Mark the exercise complete.
//  4. Fire Pet.OnTargetMet, but only on the call that crossed 1800s, then read
//     Pet.State so the §6.2 response carries pet_health / streak_count.
//
// The ordering is a contract, not an implementation detail: see the call-log
// test in service_test.go.
func (s *Service) RecordProgress(ctx context.Context, userID, exerciseID string, seconds int64) (ProgressResult, error) {
	if seconds <= 0 {
		return ProgressResult{}, fmt.Errorf("quests: duration_seconds must be positive, got %d", seconds)
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

	total, err := s.counter.Add(ctx, userID, date, seconds)
	if err != nil {
		return ProgressResult{}, err
	}

	// newly_met is derived from the counter alone: this call crossed the target
	// iff the new total is at or past it and the previous total was not.
	targetMet := total >= TargetSeconds
	newlyMet := targetMet && total-seconds < TargetSeconds

	if err := s.progress.Upsert(ctx, userID, date, int(total/60), targetMet); err != nil {
		return ProgressResult{}, err
	}
	// MarkComplete keeps its own ErrExerciseNotFound for the race where the
	// row vanished between CheckExercise and here; the handler still maps it
	// to 404.
	if err := s.quests.MarkComplete(ctx, roadmap.ID, exerciseID); err != nil {
		return ProgressResult{}, err
	}

	if newlyMet {
		// Best-effort: the study session is already recorded and must not be
		// rolled back by a pet failure.
		if err := s.pet.OnTargetMet(ctx, userID, date); err != nil {
			log.Printf("quests: pet target-met hook failed for user %s on %s: %v", userID, date, err)
		}
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
// timezone and returns that day's tasks plus today's running total.
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

	tasks := make([]Task, 0, len(exercises)) // never nil: serialises as []
	for _, e := range exercises {
		tasks = append(tasks, toTask(e))
	}

	return DailySuite{
		Date:                 date,
		DayNumber:            day,
		TotalMinutesRequired: TargetSeconds / 60,
		AccumulatedSeconds:   total,
		IsTargetMet:          total >= TargetSeconds,
		Tasks:                tasks,
	}, nil
}
