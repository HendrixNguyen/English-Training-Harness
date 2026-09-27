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
	// SaveContent writes content + content_schema=1 only if the row is not yet
	// typed (a conditional UPDATE), and returns the row's content_json either
	// way: on a lost race, the first writer's bytes come back, not this
	// caller's own content.
	SaveContent(ctx context.Context, exerciseID string, content json.RawMessage) (json.RawMessage, error)
}

const (
	// contentTargetSQL: $1 user, $2 exercise id as text — comparing id::text
	// means a non-UUID string matches nothing (ErrExerciseNotFound) instead of
	// erroring Postgres 22P02 (invalid input syntax for type uuid).
	contentTargetSQL = `
SELECT e.id, e.day_number, e.task_type, e.content_json, e.is_completed, r.created_at,
       COALESCE(u.timezone, 'UTC'), COALESCE(u.cefr_current::text, 'B1'), u.target_goal,
       COALESCE(r.roadmap_json->'modules'->((e.day_number-1)/7)->>'focus', ''),
       COALESCE(r.roadmap_json->'modules'->((e.day_number-1)/7)->'days'->((e.day_number-1)%7)->>'title', '')
FROM exercises e JOIN roadmaps r ON r.id = e.roadmap_id JOIN users u ON u.id = r.user_id
WHERE e.id::text = $2 AND r.user_id = $1 AND r.is_active`

	// saveContentSQL writes only when the row is not yet typed. Zero rows
	// affected means either the id does not exist or another writer already
	// won; saveContentByIDSQL tells the two apart.
	saveContentSQL = `
UPDATE exercises
SET content_json = content_json || jsonb_build_object('content', $2::jsonb, 'content_schema', 1)
WHERE id::text = $1 AND COALESCE(content_json->>'content_schema', '') <> '1'
RETURNING content_json`

	saveContentByIDSQL = `SELECT content_json FROM exercises WHERE id::text = $1`
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

func (r *PgRepo) SaveContent(ctx context.Context, exerciseID string, content json.RawMessage) (json.RawMessage, error) {
	var out json.RawMessage
	err := r.Pool.QueryRow(ctx, saveContentSQL, exerciseID, content).Scan(&out)
	if errors.Is(err, pgx.ErrNoRows) {
		// Zero rows: either another writer already won this race, or the id
		// does not exist at all. Re-read decides which — the first writer's
		// content_json either way, ErrExerciseNotFound if there is none.
		err2 := r.Pool.QueryRow(ctx, saveContentByIDSQL, exerciseID).Scan(&out)
		if errors.Is(err2, pgx.ErrNoRows) {
			return nil, ErrExerciseNotFound
		}
		if err2 != nil {
			return nil, fmt.Errorf("quests: re-reading content after a lost race: %w", err2)
		}
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("quests: saving content: %w", err)
	}
	return out, nil
}

var _ ContentRepo = (*PgRepo)(nil)
