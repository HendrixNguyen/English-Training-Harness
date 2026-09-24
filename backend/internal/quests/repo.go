package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoActiveRoadmap means the user has no roadmaps row with is_active = TRUE.
// Until the onboarding slice exists, this is the normal state for a new user.
var ErrNoActiveRoadmap = errors.New("quests: no active roadmap")

// ErrExerciseNotFound means the exercise id is unknown or belongs to another
// user's roadmap. The two are deliberately indistinguishable to the client.
var ErrExerciseNotFound = errors.New("quests: exercise not found")

// ErrNoProgressRow means MarkTargetMet found no daily_progress row for that
// local date. RecordProgress cannot hit it (Upsert creates the row on the
// same call); it exists so no caller can mistake "nothing there" for
// "flagged" — every verdict writer in pet already reports what it did.
var ErrNoProgressRow = errors.New("quests: no daily_progress row for that date")

// Roadmap is the slice of spec §3.2 `roadmaps` this package reads.
type Roadmap struct {
	ID        string
	CreatedAt time.Time
}

// Exercise is a §3.2 `exercises` row. It is the storage model; the §6.2 wire
// DTO is `Task` in service.go (see toTask).
type Exercise struct {
	ID          string
	DayNumber   int
	TaskType    string // vocabulary | reading | practice
	ContentJSON json.RawMessage
	IsCompleted bool
}

// Profile is the slice of `users` quests needs: the timezone decides which day
// a progress call belongs to.
type Profile struct {
	Timezone string
}

// QuestRepo is the Postgres read side plus the one exercise write.
type QuestRepo interface {
	Profile(ctx context.Context, userID string) (Profile, error)
	ActiveRoadmap(ctx context.Context, userID string) (Roadmap, error)
	ExercisesForDay(ctx context.Context, roadmapID string, day int) ([]Exercise, error)
	// CheckExercise is the read-only ownership check RecordProgress runs
	// before it writes anything: nil when exerciseID is on roadmapID for
	// day, ErrExerciseNotFound otherwise. Unknown ids and other users' ids
	// are deliberately indistinguishable.
	CheckExercise(ctx context.Context, roadmapID, exerciseID string, day int) error
	// MarkComplete sets is_completed and returns ErrExerciseNotFound when the
	// exercise is not on roadmapID.
	MarkComplete(ctx context.Context, roadmapID, exerciseID string) error
}

// ProgressRepo owns daily_progress. The row is monotonic: minutes_spent never
// lowers and is_target_met never returns to FALSE, so a Redis counter lost
// mid-day cannot rewrite the durable record from a restarted total.
// is_target_met means "the pet has been told about this day": Upsert reports
// it, and MarkTargetMet sets it after Pet.OnTargetMet has returned nil.
type ProgressRepo interface {
	// Upsert writes minutes_spent for (userID, localDate) — never lowering it —
	// and reports whether is_target_met is already TRUE on that row.
	Upsert(ctx context.Context, userID, localDate string, minutes int) (alreadyMet bool, err error)
	// MarkTargetMet flips is_target_met to TRUE. Idempotent; ErrNoProgressRow
	// when the row does not exist.
	MarkTargetMet(ctx context.Context, userID, localDate string) error
	// TargetMet reports the row's is_target_met for (userID, localDate); no
	// row → false. GET /quests/daily reads it so both endpoints answer the
	// same is_target_met for the same local day, counter or no counter.
	TargetMet(ctx context.Context, userID, localDate string) (bool, error)
}

const (
	profileSQL = `SELECT COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	activeRoadmapSQL = `
SELECT id, created_at
FROM roadmaps
WHERE user_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT 1`

	exercisesForDaySQL = `
SELECT id, day_number, task_type, content_json, is_completed
FROM exercises
WHERE roadmap_id = $1 AND day_number = $2
ORDER BY task_type`

	checkExerciseSQL = `
SELECT 1
FROM exercises
WHERE id = $1 AND roadmap_id = $2 AND day_number = $3`

	markCompleteSQL = `
UPDATE exercises
SET is_completed = TRUE
WHERE id = $1 AND roadmap_id = $2`

	// daily_progress.date defaults to CURRENT_DATE, which is the *server's*
	// date — always pass the user's local date explicitly. A new row starts
	// is_target_met = FALSE whatever the total: only MarkTargetMet sets it,
	// after the pet hook, so the flag never gets ahead of the pet.
	upsertProgressSQL = `
INSERT INTO daily_progress (user_id, date, minutes_spent, is_target_met)
VALUES ($1, $2::date, $3, FALSE)
ON CONFLICT (user_id, date) DO UPDATE SET
    minutes_spent = GREATEST(COALESCE(daily_progress.minutes_spent, 0), EXCLUDED.minutes_spent)
RETURNING COALESCE(is_target_met, FALSE)`

	markTargetMetSQL = `
UPDATE daily_progress
SET is_target_met = TRUE
WHERE user_id = $1 AND date = $2::date`

	targetMetSQL = `
SELECT COALESCE(is_target_met, FALSE)
FROM daily_progress
WHERE user_id = $1 AND date = $2::date`
)

// PgRepo implements both QuestRepo and ProgressRepo over one pool.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) Profile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	if err := r.Pool.QueryRow(ctx, profileSQL, userID).Scan(&p.Timezone); err != nil {
		return Profile{}, fmt.Errorf("quests: reading profile: %w", err)
	}
	return p, nil
}

func (r *PgRepo) ActiveRoadmap(ctx context.Context, userID string) (Roadmap, error) {
	var rm Roadmap
	err := r.Pool.QueryRow(ctx, activeRoadmapSQL, userID).Scan(&rm.ID, &rm.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Roadmap{}, ErrNoActiveRoadmap
	}
	if err != nil {
		return Roadmap{}, fmt.Errorf("quests: reading active roadmap: %w", err)
	}
	return rm, nil
}

func (r *PgRepo) ExercisesForDay(ctx context.Context, roadmapID string, day int) ([]Exercise, error) {
	rows, err := r.Pool.Query(ctx, exercisesForDaySQL, roadmapID, day)
	if err != nil {
		return nil, fmt.Errorf("quests: reading exercises: %w", err)
	}
	defer rows.Close()

	var out []Exercise
	for rows.Next() {
		var e Exercise
		if err := rows.Scan(&e.ID, &e.DayNumber, &e.TaskType, &e.ContentJSON, &e.IsCompleted); err != nil {
			return nil, fmt.Errorf("quests: scanning exercise: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PgRepo) CheckExercise(ctx context.Context, roadmapID, exerciseID string, day int) error {
	var one int
	err := r.Pool.QueryRow(ctx, checkExerciseSQL, exerciseID, roadmapID, day).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExerciseNotFound
	}
	if err != nil {
		return fmt.Errorf("quests: checking exercise: %w", err)
	}
	return nil
}

func (r *PgRepo) MarkComplete(ctx context.Context, roadmapID, exerciseID string) error {
	tag, err := r.Pool.Exec(ctx, markCompleteSQL, exerciseID, roadmapID)
	if err != nil {
		return fmt.Errorf("quests: marking exercise complete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrExerciseNotFound
	}
	return nil
}

func (r *PgRepo) Upsert(ctx context.Context, userID, localDate string, minutes int) (bool, error) {
	var alreadyMet bool
	if err := r.Pool.QueryRow(ctx, upsertProgressSQL, userID, localDate, minutes).Scan(&alreadyMet); err != nil {
		return false, fmt.Errorf("quests: upserting daily_progress: %w", err)
	}
	return alreadyMet, nil
}

func (r *PgRepo) MarkTargetMet(ctx context.Context, userID, localDate string) error {
	tag, err := r.Pool.Exec(ctx, markTargetMetSQL, userID, localDate)
	if err != nil {
		return fmt.Errorf("quests: marking target met: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoProgressRow
	}
	return nil
}

func (r *PgRepo) TargetMet(ctx context.Context, userID, localDate string) (bool, error) {
	var met bool
	err := r.Pool.QueryRow(ctx, targetMetSQL, userID, localDate).Scan(&met)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("quests: reading daily_progress flag: %w", err)
	}
	return met, nil
}

var (
	_ QuestRepo    = (*PgRepo)(nil)
	_ ProgressRepo = (*PgRepo)(nil)
)
