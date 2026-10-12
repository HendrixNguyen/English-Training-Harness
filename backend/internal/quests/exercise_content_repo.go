package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ContentTarget is what ContentService needs to generate or return one
// exercise's typed content: the exercise row itself plus enough of its
// roadmap and the learner's profile to brief the AI (level, goal, where in
// the curriculum the task sits).
type ContentTarget struct {
	Exercise         Exercise
	RoadmapCreatedAt time.Time
	Timezone         string
	Level            string
	Goal             string
	ModuleFocus      string
	DayTitle         string
}

// ContentRepo is the read/write side ContentService needs for one exercise's
// typed content. It is deliberately narrower than QuestRepo: it never lists a
// whole day and it owns the one conditional write content generation makes.
type ContentRepo interface {
	// ContentTarget finds exerciseID on userID's ACTIVE roadmap; anything else
	// (unknown id, another user's exercise, an inactive roadmap, or an id that
	// is not a UUID) is ErrExerciseNotFound. The cases are deliberately
	// indistinguishable to the caller, same as CheckExercise.
	ContentTarget(ctx context.Context, userID, exerciseID string) (ContentTarget, error)
	// SaveContent stores generated content and its actual level when the original
	// snapshot still matches an owned, active, incomplete exercise. False is a
	// CAS loss, including deleted/inactive/foreign/completed rows; callers reread
	// once through ContentTarget. A winner receives the persisted PostgreSQL
	// bytes; a loser receives nil content and never overwrites a winner.
	SaveContent(ctx context.Context, userID, exerciseID string, original, content json.RawMessage, level string) (json.RawMessage, bool, error)
}

const (
	// contentTargetSQL: $1 user, $2 exercise id as text — comparing id::text
	// means a non-UUID string matches nothing (ErrExerciseNotFound) instead of
	// erroring Postgres 22P02 (invalid input syntax for type uuid).
	contentTargetSQL = `
SELECT e.id, e.day_number, e.task_type, e.content_json, e.is_completed, r.created_at,
       COALESCE(u.timezone, 'UTC'), COALESCE(u.cefr_current::text, 'B1'), u.target_goal,
       COALESCE(NULLIF(btrim(r.roadmap_json->'modules'->((e.day_number-1)/7)->>'focus'), ''), r.roadmap_json->'modules'->((e.day_number-1)/7)->>'title', ''),
       COALESCE(r.roadmap_json->'modules'->((e.day_number-1)/7)->'days'->((e.day_number-1)%7)->>'title', '')
FROM exercises e JOIN roadmaps r ON r.id = e.roadmap_id JOIN users u ON u.id = r.user_id
WHERE e.id::text = $2 AND r.user_id = $1 AND r.is_active`

	// CAS includes ownership and active/incomplete status at the write, including
	// first generation. A loser is reread by the service through ContentTarget.
	saveContentSQL = `
UPDATE exercises e
SET content_json = e.content_json || jsonb_build_object('content', $4::jsonb, 'content_schema', 1, 'content_level', $5::text)
FROM roadmaps r
WHERE e.id::text = $2 AND r.id = e.roadmap_id AND r.user_id = $1 AND r.is_active
  AND NOT e.is_completed AND e.content_json = $3::jsonb
RETURNING e.content_json`
)

func (r *PgRepo) ContentTarget(ctx context.Context, userID, exerciseID string) (ContentTarget, error) {
	var t ContentTarget
	err := r.Pool.QueryRow(ctx, contentTargetSQL, userID, exerciseID).Scan(
		&t.Exercise.ID, &t.Exercise.DayNumber, &t.Exercise.TaskType, &t.Exercise.ContentJSON, &t.Exercise.IsCompleted,
		&t.RoadmapCreatedAt, &t.Timezone, &t.Level, &t.Goal, &t.ModuleFocus, &t.DayTitle,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentTarget{}, ErrExerciseNotFound
	}
	if err != nil {
		return ContentTarget{}, fmt.Errorf("quests: reading content target: %w", err)
	}
	return t, nil
}

func (r *PgRepo) SaveContent(ctx context.Context, userID, exerciseID string, original, content json.RawMessage, level string) (json.RawMessage, bool, error) {
	var saved json.RawMessage
	err := r.Pool.QueryRow(ctx, saveContentSQL, userID, exerciseID, original, content, level).Scan(&saved)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("quests: saving content: %w", err)
	}
	return saved, true, nil
}

var _ ContentRepo = (*PgRepo)(nil)
