package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"net/http"
	"sort"
	"time"
)

// RoadmapOutline exposes only owned plan metadata and durable progress.
// Exercise content, answer keys and quiz results never enter this DTO.
type RoadmapOutline struct {
	RoadmapID string          `json:"roadmap_id"`
	Title     string          `json:"title"`
	CEFRLevel string          `json:"cefr_level"`
	Today     string          `json:"today"`
	DayNumber int             `json:"day_number"`
	Modules   []RoadmapModule `json:"modules"`
}
type RoadmapModule struct {
	Week  int          `json:"week"`
	Title string       `json:"title"`
	Focus string       `json:"focus"`
	Days  []RoadmapDay `json:"days"`
}
type RoadmapDay struct {
	DayNumber    int           `json:"day_number"`
	Date         string        `json:"date"`
	Title        string        `json:"title"`
	MinutesSpent int           `json:"minutes_spent"`
	IsTargetMet  bool          `json:"is_target_met"`
	Tasks        []RoadmapTask `json:"tasks"`
}
type RoadmapTask struct {
	ID              string `json:"id"`
	TaskType        string `json:"task_type"`
	Title           string `json:"title"`
	DurationMinutes int    `json:"duration_minutes"`
	IsCompleted     *bool  `json:"is_completed"` // NULL means historical completion unknown.
}
type RoadmapStoredTask struct {
	RoadmapTask
	DayNumber int `json:"day_number"`
}
type RoadmapProgress struct {
	MinutesSpent int  `json:"minutes_spent"`
	IsTargetMet  bool `json:"is_target_met"`
}
type RoadmapSnapshot struct {
	ID        string
	CreatedAt time.Time
	Timezone  string
	Outline   json.RawMessage
	Tasks     []RoadmapStoredTask
	Progress  map[string]RoadmapProgress
}
type RoadmapReader interface {
	ReadRoadmap(context.Context, string) (RoadmapSnapshot, error)
}
type RoadmapService struct {
	repo RoadmapReader
	now  func() time.Time
}

func NewRoadmapService(repo RoadmapReader, now func() time.Time) *RoadmapService {
	return &RoadmapService{repo: repo, now: now}
}

func (s *RoadmapService) Read(ctx context.Context, owner string) (RoadmapOutline, error) {
	snapshot, err := s.repo.ReadRoadmap(ctx, owner)
	if err != nil {
		return RoadmapOutline{}, err
	}
	// Deliberately decode only presentation metadata, never generated content.
	var plan struct {
		Title     string `json:"title"`
		CEFRLevel string `json:"cefr_level"`
		Modules   []struct {
			Week  int    `json:"week"`
			Title string `json:"title"`
			Focus string `json:"focus"`
			Days  []struct {
				Title string `json:"title"`
			} `json:"days"`
		} `json:"modules"`
	}
	if err = json.Unmarshal(snapshot.Outline, &plan); err != nil {
		return RoadmapOutline{}, fmt.Errorf("quests: decoding roadmap metadata: %w", err)
	}
	if len(plan.Modules) != 4 || plan.Title == "" || plan.CEFRLevel == "" {
		return RoadmapOutline{}, errors.New("quests: invalid roadmap metadata")
	}
	loc := Location(snapshot.Timezone)
	start := snapshot.CreatedAt.In(loc)
	now := s.now().In(loc)
	// Re-express calendar components as UTC, so DST never changes the day count.
	startDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	day := int(todayDate.Sub(startDate)/(24*time.Hour)) + 1
	if day < 1 {
		day = 1
	}
	out := RoadmapOutline{RoadmapID: snapshot.ID, Title: plan.Title, CEFRLevel: plan.CEFRLevel, Today: LocalDate(now, loc), DayNumber: day, Modules: make([]RoadmapModule, 0, 4)}
	tasks := map[int][]RoadmapTask{}
	for _, task := range snapshot.Tasks {
		tasks[task.DayNumber] = append(tasks[task.DayNumber], task.RoadmapTask)
	}
	sort.Slice(plan.Modules, func(i, j int) bool { return plan.Modules[i].Week < plan.Modules[j].Week })
	for mi, m := range plan.Modules {
		if m.Week != mi+1 || len(m.Days) != 7 {
			return RoadmapOutline{}, errors.New("quests: invalid roadmap calendar")
		}
		module := RoadmapModule{Week: m.Week, Title: m.Title, Focus: m.Focus, Days: make([]RoadmapDay, 0, 7)}
		for di, d := range m.Days {
			n := (m.Week-1)*7 + di + 1
			date := startDate.AddDate(0, 0, n-1).Format("2006-01-02")
			p := snapshot.Progress[date]
			dailyTasks := tasks[n]
			if dailyTasks == nil {
				dailyTasks = []RoadmapTask{}
			}
			sort.SliceStable(dailyTasks, func(i, j int) bool {
				order := map[string]int{"vocabulary": 0, "reading": 1, "practice": 2}
				return order[dailyTasks[i].TaskType] < order[dailyTasks[j].TaskType]
			})
			module.Days = append(module.Days, RoadmapDay{DayNumber: n, Date: date, Title: d.Title, MinutesSpent: p.MinutesSpent, IsTargetMet: p.IsTargetMet, Tasks: dailyTasks})
		}
		out.Modules = append(out.Modules, module)
	}
	return out, nil
}

// One SQL statement supplies a consistent active-roadmap snapshot, including
// owner-scoped progress. Concurrent replacement cannot mix old/new task IDs.
const readRoadmapSQL = `
SELECT r.id, r.created_at, COALESCE(u.timezone,'UTC'),
 jsonb_build_object('title',r.roadmap_json->'title','cefr_level',r.roadmap_json->'cefr_level','modules',
   (SELECT jsonb_agg(jsonb_build_object('week',m->'week','title',m->'title','focus',m->'focus','days',
     (SELECT jsonb_agg(jsonb_build_object('title',d->'title') ORDER BY dn)
      FROM jsonb_array_elements(m->'days') WITH ORDINALITY AS ds(d,dn))) ORDER BY mn)
    FROM jsonb_array_elements(r.roadmap_json->'modules') WITH ORDINALITY AS ms(m,mn))),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('id',e.id,'day_number',e.day_number,'task_type',e.task_type,
 'title',e.content_json->'title','duration_minutes',e.content_json->'duration_minutes','is_completed',e.is_completed)
 ORDER BY e.day_number,e.task_type) FROM exercises e WHERE e.roadmap_id=r.id),'[]'::jsonb),
 COALESCE((SELECT jsonb_object_agg(p.date::text,jsonb_build_object('minutes_spent',COALESCE(p.minutes_spent,0),'is_target_met',COALESCE(p.is_target_met,false)))
 FROM daily_progress p WHERE p.user_id=u.id),'{}'::jsonb)
FROM roadmaps r JOIN users u ON u.id=r.user_id
WHERE r.user_id=$1 AND r.is_active=TRUE ORDER BY r.created_at DESC,r.id DESC LIMIT 1`

func (r *PgRepo) ReadRoadmap(ctx context.Context, owner string) (RoadmapSnapshot, error) {
	var s RoadmapSnapshot
	var tasks, progress []byte
	err := r.Pool.QueryRow(ctx, readRoadmapSQL, owner).Scan(&s.ID, &s.CreatedAt, &s.Timezone, &s.Outline, &tasks, &progress)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNoActiveRoadmap
	}
	if err != nil {
		return s, fmt.Errorf("quests: reading roadmap: %w", err)
	}
	if err = json.Unmarshal(tasks, &s.Tasks); err != nil {
		return s, fmt.Errorf("quests: reading task summaries: %w", err)
	}
	if err = json.Unmarshal(progress, &s.Progress); err != nil {
		return s, fmt.Errorf("quests: reading roadmap progress: %w", err)
	}
	return s, nil
}
func RoadmapHandler(svc *RoadmapService) gin.HandlerFunc {
	return func(c *gin.Context) {
		owner := auth.UserID(c)
		if owner == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		out, err := svc.Read(c.Request.Context(), owner)
		switch {
		case errors.Is(err, ErrNoActiveRoadmap):
			c.JSON(http.StatusNotFound, gin.H{"error": "no_active_roadmap"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, out)
		}
	}
}
