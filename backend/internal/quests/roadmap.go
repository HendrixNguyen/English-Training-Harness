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
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
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

// RoadmapTask is one entry of a day's tasks in the GET /api/v1/roadmap body
// (backend spec §6.2): the outline only — no content_json.
type RoadmapTask struct {
	TaskType        string `json:"task_type"`
	Title           string `json:"title"`
	DurationMinutes int    `json:"duration_minutes"`
}

// RoadmapDay is one of the 28 days: its plan and what daily_progress says
// happened on its date.
type RoadmapDay struct {
	DayNumber    int           `json:"day_number"`
	Date         string        `json:"date"` // YYYY-MM-DD in the user's timezone (DayDate)
	Title        string        `json:"title"`
	Tasks        []RoadmapTask `json:"tasks"`
	MinutesSpent int           `json:"minutes_spent"`
	IsTargetMet  bool          `json:"is_target_met"`
}

// RoadmapModule is one of the four weekly modules (§6.1).
type RoadmapModule struct {
	Week  int          `json:"week"`
	Title string       `json:"title"`
	Focus string       `json:"focus"`
	Days  []RoadmapDay `json:"days"`
}

// RoadmapOutline is the GET /api/v1/roadmap 200 body — backend spec §6.2,
// field for field.
type RoadmapOutline struct {
	RoadmapID string          `json:"roadmap_id"`
	Title     string          `json:"title"`
	CEFRLevel string          `json:"cefr_level"`
	CreatedAt time.Time       `json:"created_at"`
	DayNumber int             `json:"day_number"`
	Modules   []RoadmapModule `json:"modules"`
}

// Roadmap resolves the active roadmap document, joins it with daily_progress
// over its 28 days and reports the learner's real progress against it. Three
// rules govern the join: dates are DayDate in the user's timezone (the same
// rule DayNumber uses, so the tree and the daily suite can never disagree
// about which date a day is); a day with no daily_progress row is
// minutes_spent 0 / is_target_met false; and the flag is exactly what the pet
// was told — a day past 30 minutes whose OnTargetMet hook failed still reads
// 30/false until the next progress call retries it. The tree does not know
// better than the pet.
func (s *Service) Roadmap(ctx context.Context, userID string) (RoadmapOutline, error) {
	profile, err := s.quests.Profile(ctx, userID)
	if err != nil {
		return RoadmapOutline{}, err
	}
	doc, err := s.quests.ActiveRoadmapDoc(ctx, userID)
	if err != nil {
		return RoadmapOutline{}, err
	}

	var parsed airouter.Roadmap
	if err := json.Unmarshal(doc.JSON, &parsed); err != nil {
		return RoadmapOutline{}, fmt.Errorf("quests: roadmap_json for roadmap %s: %w", doc.ID, err)
	}
	if len(parsed.Modules) != airouter.Modules {
		return RoadmapOutline{}, fmt.Errorf("quests: roadmap %s has %d modules, want %d", doc.ID, len(parsed.Modules), airouter.Modules)
	}
	for mi, m := range parsed.Modules {
		if len(m.Days) != airouter.DaysPerModule {
			return RoadmapOutline{}, fmt.Errorf("quests: roadmap %s module %d has %d days, want %d", doc.ID, mi+1, len(m.Days), airouter.DaysPerModule)
		}
	}

	loc := Location(profile.Timezone)
	now := s.now()
	day := DayNumber(doc.CreatedAt, now, loc)

	rows, err := s.progress.ProgressBetween(ctx, userID, DayDate(doc.CreatedAt, 1, loc), DayDate(doc.CreatedAt, RoadmapDays, loc))
	if err != nil {
		return RoadmapOutline{}, err
	}

	modules := make([]RoadmapModule, 0, airouter.Modules)
	for mi, m := range parsed.Modules {
		days := make([]RoadmapDay, 0, airouter.DaysPerModule)
		for di, d := range m.Days {
			n := mi*airouter.DaysPerModule + di + 1
			date := DayDate(doc.CreatedAt, n, loc)
			p := rows[date]

			tasks := make([]RoadmapTask, 0, len(d.Tasks))
			for _, task := range d.Tasks {
				duration := task.DurationMinutes
				if duration <= 0 {
					duration = DefaultTaskMinutes
				}
				tasks = append(tasks, RoadmapTask{
					TaskType:        task.Type,
					Title:           task.Title,
					DurationMinutes: duration,
				})
			}

			days = append(days, RoadmapDay{
				DayNumber:    n,
				Date:         date,
				Title:        d.Title,
				Tasks:        tasks,
				MinutesSpent: p.MinutesSpent,
				IsTargetMet:  p.IsTargetMet,
			})
		}
		modules = append(modules, RoadmapModule{
			Week:  m.Week,
			Title: m.Title,
			Focus: m.Focus,
			Days:  days,
		})
	}

	return RoadmapOutline{
		RoadmapID: doc.ID,
		Title:     parsed.Title,
		CEFRLevel: parsed.CEFRLevel,
		CreatedAt: doc.CreatedAt,
		DayNumber: day,
		Modules:   modules,
	}, nil
}

// RoadmapHandler serves GET /api/v1/roadmap (backend spec §6.2). Mount behind
// auth.Require().
func RoadmapHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := auth.UserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		outline, err := svc.Roadmap(c.Request.Context(), userID)
		switch {
		case errors.Is(err, ErrNoActiveRoadmap):
			c.JSON(http.StatusNotFound, gin.H{"error": "no_active_roadmap"})
		case err != nil:
			log.Printf("quests: roadmap outline for user %s: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, outline)
		}
	}
}
