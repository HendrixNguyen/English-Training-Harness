package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
)

// ErrUnknownUser means the authenticated id has no users row.
var ErrUnknownUser = errors.New("onboarding: unknown user")

// Profile is the slice of users the idempotent path and Regenerate report back.
type Profile struct {
	CEFRCurrent string
	TargetGoal  string
}

// Assessment is everything one successful onboarding writes, in one tx.
type Assessment struct {
	CEFRLevel        string
	TargetGoal       string
	Timezone         string
	NotificationTime string // HH:MM:SS
	Roadmap          airouter.Roadmap
}

// Repo is the Postgres side. onboarding owns roadmaps/exercises writes and
// the users columns onboarding fills (cefr_current, target_goal, timezone,
// notification_time); quests reads roadmaps/exercises afterwards.
type Repo interface {
	// ActiveRoadmapID returns ok=false when the user has no active roadmap.
	ActiveRoadmapID(ctx context.Context, userID string) (id string, ok bool, err error)
	Profile(ctx context.Context, userID string) (Profile, error)
	// SaveAssessment updates the user, deactivates previous roadmaps, inserts
	// the new active roadmap and its 84 exercises, atomically. Returns the
	// roadmap id.
	SaveAssessment(ctx context.Context, userID string, a Assessment) (string, error)
	// ReplaceRoadmap sets cefr_current, deactivates active roadmaps and
	// inserts the new active roadmap with its 84 exercises, atomically.
	// Returns the roadmap id.
	ReplaceRoadmap(ctx context.Context, userID, cefrLevel string, roadmap airouter.Roadmap) (string, error)
}

const (
	activeRoadmapSQL = `SELECT id FROM roadmaps WHERE user_id = $1 AND is_active = TRUE ORDER BY created_at DESC LIMIT 1`

	profileSQL = `SELECT COALESCE(cefr_current::text, 'A1'), COALESCE(target_goal, '') FROM users WHERE id = $1`

	updateUserSQL = `
UPDATE users
SET cefr_current = $2::cefr_level, target_goal = $3, timezone = $4, notification_time = $5::time
WHERE id = $1`

	updateLevelSQL = `UPDATE users SET cefr_current = $2::cefr_level WHERE id = $1`

	deactivateSQL = `UPDATE roadmaps SET is_active = FALSE WHERE user_id = $1 AND is_active = TRUE`

	insertRoadmapSQL = `INSERT INTO roadmaps (user_id, roadmap_json, is_active) VALUES ($1, $2::jsonb, TRUE) RETURNING id`

	insertExerciseSQL = `
INSERT INTO exercises (roadmap_id, day_number, task_type, content_json)
VALUES ($1, $2, $3::task_category, $4::jsonb)`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) ActiveRoadmapID(ctx context.Context, userID string) (string, bool, error) {
	var id string
	err := r.Pool.QueryRow(ctx, activeRoadmapSQL, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("onboarding: reading active roadmap: %w", err)
	}
	return id, true, nil
}

func (r *PgRepo) Profile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	err := r.Pool.QueryRow(ctx, profileSQL, userID).Scan(&p.CEFRCurrent, &p.TargetGoal)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrUnknownUser
	}
	if err != nil {
		return Profile{}, fmt.Errorf("onboarding: reading profile: %w", err)
	}
	return p, nil
}

func (r *PgRepo) SaveAssessment(ctx context.Context, userID string, a Assessment) (string, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("onboarding: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, updateUserSQL, userID, a.CEFRLevel, a.TargetGoal, a.Timezone, a.NotificationTime)
	if err != nil {
		return "", fmt.Errorf("onboarding: updating user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", ErrUnknownUser
	}

	roadmapID, err := insertActiveRoadmap(ctx, tx, userID, a.Roadmap)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("onboarding: commit: %w", err)
	}
	return roadmapID, nil
}

// ReplaceRoadmap sets cefr_current (row lock — see the plan's Review Focus:
// two concurrent regenerates serialise here, and the second sees the first's
// roadmap already deactivated), then deactivates the active roadmap and
// inserts the new one with its 84 exercises, in the same transaction.
func (r *PgRepo) ReplaceRoadmap(ctx context.Context, userID, cefrLevel string, roadmap airouter.Roadmap) (string, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("onboarding: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, updateLevelSQL, userID, cefrLevel)
	if err != nil {
		return "", fmt.Errorf("onboarding: updating cefr_current: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", ErrUnknownUser
	}

	roadmapID, err := insertActiveRoadmap(ctx, tx, userID, roadmap)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("onboarding: commit: %w", err)
	}
	return roadmapID, nil
}

// insertActiveRoadmap deactivates any active roadmap for userID and inserts
// roadmap as the new active one with its 84 exercises, all within tx. Shared
// by SaveAssessment and ReplaceRoadmap.
func insertActiveRoadmap(ctx context.Context, tx pgx.Tx, userID string, roadmap airouter.Roadmap) (string, error) {
	roadmapJSON, err := json.Marshal(roadmap)
	if err != nil {
		return "", fmt.Errorf("onboarding: encoding roadmap: %w", err)
	}
	if _, err := tx.Exec(ctx, deactivateSQL, userID); err != nil {
		return "", fmt.Errorf("onboarding: deactivating roadmaps: %w", err)
	}
	var roadmapID string
	if err := tx.QueryRow(ctx, insertRoadmapSQL, userID, roadmapJSON).Scan(&roadmapID); err != nil {
		return "", fmt.Errorf("onboarding: inserting roadmap: %w", err)
	}

	batch := &pgx.Batch{}
	exercises := roadmap.Exercises()
	for _, e := range exercises {
		batch.Queue(insertExerciseSQL, roadmapID, e.DayNumber, e.TaskType, e.ContentJSON)
	}
	results := tx.SendBatch(ctx, batch)
	for i := range exercises {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return "", fmt.Errorf("onboarding: inserting exercise %d: %w", i, err)
		}
	}
	if err := results.Close(); err != nil {
		return "", fmt.Errorf("onboarding: closing batch: %w", err)
	}
	return roadmapID, nil
}

var _ Repo = (*PgRepo)(nil)
