package quests

// GET /api/v1/roadmap — the 28-day outline joined with per-day progress.
// Everything for that read lives in this file (design: harness/designs/
// roadmap-tree.md). It shares Service, QuestRepo and ProgressRepo with the
// daily loop and adds no write.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// RoadmapDoc is the §3.2 roadmaps row with its roadmap_json: the
// airouter.Roadmap onboarding validated and stored.
type RoadmapDoc struct {
	ID        string
	CreatedAt time.Time
	JSON      json.RawMessage
}

// DayProgress is one daily_progress row as GET /roadmap reports it.
type DayProgress struct {
	MinutesSpent int
	IsTargetMet  bool
}

const (
	activeRoadmapDocSQL = `
SELECT id, created_at, roadmap_json
FROM roadmaps
WHERE user_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT 1`

	// date::text is YYYY-MM-DD under the default ISO DateStyle — the same
	// string LocalDate/DayDate produce, so the map key needs no formatting.
	progressBetweenSQL = `
SELECT date::text, COALESCE(minutes_spent, 0), COALESCE(is_target_met, FALSE)
FROM daily_progress
WHERE user_id = $1 AND date BETWEEN $2::date AND $3::date`
)

func (r *PgRepo) ActiveRoadmapDoc(ctx context.Context, userID string) (RoadmapDoc, error) {
	var doc RoadmapDoc
	err := r.Pool.QueryRow(ctx, activeRoadmapDocSQL, userID).Scan(&doc.ID, &doc.CreatedAt, &doc.JSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoadmapDoc{}, ErrNoActiveRoadmap
	}
	if err != nil {
		return RoadmapDoc{}, fmt.Errorf("quests: reading active roadmap document: %w", err)
	}
	return doc, nil
}

func (r *PgRepo) ProgressBetween(ctx context.Context, userID, fromDate, toDate string) (map[string]DayProgress, error) {
	rows, err := r.Pool.Query(ctx, progressBetweenSQL, userID, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("quests: reading daily_progress range: %w", err)
	}
	defer rows.Close()

	out := map[string]DayProgress{}
	for rows.Next() {
		var date string
		var p DayProgress
		if err := rows.Scan(&date, &p.MinutesSpent, &p.IsTargetMet); err != nil {
			return nil, fmt.Errorf("quests: scanning daily_progress row: %w", err)
		}
		out[date] = p
	}
	return out, rows.Err()
}
