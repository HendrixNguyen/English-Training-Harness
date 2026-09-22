package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DemoRoadmapDays and DemoTaskTypes mirror spec §6.1: 4 modules of 7 days,
// three 10-minute tasks each.
const DemoRoadmapDays = 28

// DemoTaskTypes are the three task_category values (§3.2).
var DemoTaskTypes = []string{"vocabulary", "reading", "practice"}

// SeedDemoRoadmap inserts one active roadmap with 28 days x 3 exercises for a
// user and returns the roadmap id. Each content_json carries the `title` and
// `duration_minutes` that the backend spec §6.2 daily response exposes per
// task — §3.2 has no columns for them, so quests reads them from here.
//
// TEMPORARY. This exists only because the onboarding slice (which generates a
// real roadmap via the AI router, spec §5.1 steps 4-5) is not part of this MVP
// run. When onboarding lands, DELETE this file rather than extending it.
func SeedDemoRoadmap(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	var roadmapID string
	err := pool.QueryRow(ctx,
		`INSERT INTO roadmaps (user_id, roadmap_json, is_active) VALUES ($1, $2::jsonb, TRUE) RETURNING id`,
		userID, `{"source":"demo-seed","modules":4,"days":28}`,
	).Scan(&roadmapID)
	if err != nil {
		return "", fmt.Errorf("store: seeding roadmap: %w", err)
	}

	for day := 1; day <= DemoRoadmapDays; day++ {
		for _, taskType := range DemoTaskTypes {
			content := fmt.Sprintf(`{"title":"Day %d %s","duration_minutes":10,"day":%d,"task":"%s"}`, day, taskType, day, taskType)
			if _, err := pool.Exec(ctx,
				`INSERT INTO exercises (roadmap_id, day_number, task_type, content_json)
				 VALUES ($1, $2, $3::task_category, $4::jsonb)`,
				roadmapID, day, taskType, content,
			); err != nil {
				return "", fmt.Errorf("store: seeding day %d %s: %w", day, taskType, err)
			}
		}
	}
	return roadmapID, nil
}
