package google

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoActiveRoadmap means the user has no roadmaps row with is_active = TRUE;
// the sync then pushes only the Calendar event.
var ErrNoActiveRoadmap = errors.New("google: no active roadmap")

// ErrNoSyncState means this user has never synced (no google_sync row).
var ErrNoSyncState = errors.New("google: no sync state")

// Profile is the slice of §3.2 users the event needs.
type Profile struct {
	NotificationTime string // "HH:MM:SS" as Postgres renders TIME
	Timezone         string // IANA name, 'UTC' when NULL
}

// Roadmap is the slice of §3.2 roadmaps this package reads.
type Roadmap struct {
	ID        string
	CreatedAt time.Time
}

// DayTitles is one roadmap day's exercise titles (content_json->>'title'),
// in task_type order.
type DayTitles struct {
	Day    int
	Titles []string
}

// SyncState is the google_sync row (migration 0002). Empty strings mean NULL.
type SyncState struct {
	UserID            string
	CalendarEventID   string
	TasklistID        string
	RoadmapID         string
	TasksCreatedCount int
}

// Repo is the Postgres side of Sync. Everything but SaveSyncState is read-only.
type Repo interface {
	Profile(ctx context.Context, userID string) (Profile, error)
	ActiveRoadmap(ctx context.Context, userID string) (Roadmap, error)
	DayTitles(ctx context.Context, roadmapID string) ([]DayTitles, error)
	SyncState(ctx context.Context, userID string) (SyncState, error)
	SaveSyncState(ctx context.Context, s SyncState) error
}

const (
	profileSQL = `SELECT COALESCE(notification_time::text, '20:00:00'), COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	activeRoadmapSQL = `
SELECT id, created_at
FROM roadmaps
WHERE user_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT 1`

	dayTitlesSQL = `
SELECT day_number, COALESCE(content_json->>'title', '')
FROM exercises
WHERE roadmap_id = $1
ORDER BY day_number, task_type`

	syncStateSQL = `
SELECT COALESCE(calendar_event_id, ''), COALESCE(tasklist_id, ''), COALESCE(roadmap_id::text, ''), tasks_created_count
FROM google_sync
WHERE user_id = $1`

	// NULLIF turns the empty-string convention back into NULL; the FK on
	// roadmap_id is ON DELETE SET NULL, so a vanished roadmap reads as ''.
	saveSyncStateSQL = `
INSERT INTO google_sync (user_id, calendar_event_id, tasklist_id, roadmap_id, tasks_created_count, synced_at)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, '')::uuid, $5, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET
    calendar_event_id = EXCLUDED.calendar_event_id,
    tasklist_id = EXCLUDED.tasklist_id,
    roadmap_id = EXCLUDED.roadmap_id,
    tasks_created_count = EXCLUDED.tasks_created_count,
    synced_at = EXCLUDED.synced_at`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) Profile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	if err := r.Pool.QueryRow(ctx, profileSQL, userID).Scan(&p.NotificationTime, &p.Timezone); err != nil {
		return Profile{}, fmt.Errorf("google: reading profile: %w", err)
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
		return Roadmap{}, fmt.Errorf("google: reading active roadmap: %w", err)
	}
	return rm, nil
}

func (r *PgRepo) DayTitles(ctx context.Context, roadmapID string) ([]DayTitles, error) {
	rows, err := r.Pool.Query(ctx, dayTitlesSQL, roadmapID)
	if err != nil {
		return nil, fmt.Errorf("google: reading exercise titles: %w", err)
	}
	defer rows.Close()

	var out []DayTitles
	for rows.Next() {
		var (
			day   int
			title string
		)
		if err := rows.Scan(&day, &title); err != nil {
			return nil, fmt.Errorf("google: scanning exercise title: %w", err)
		}
		if n := len(out); n > 0 && out[n-1].Day == day {
			out[n-1].Titles = append(out[n-1].Titles, title)
			continue
		}
		out = append(out, DayTitles{Day: day, Titles: []string{title}})
	}
	return out, rows.Err()
}

func (r *PgRepo) SyncState(ctx context.Context, userID string) (SyncState, error) {
	s := SyncState{UserID: userID}
	err := r.Pool.QueryRow(ctx, syncStateSQL, userID).Scan(&s.CalendarEventID, &s.TasklistID, &s.RoadmapID, &s.TasksCreatedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return SyncState{}, ErrNoSyncState
	}
	if err != nil {
		return SyncState{}, fmt.Errorf("google: reading sync state: %w", err)
	}
	return s, nil
}

func (r *PgRepo) SaveSyncState(ctx context.Context, s SyncState) error {
	if _, err := r.Pool.Exec(ctx, saveSyncStateSQL, s.UserID, s.CalendarEventID, s.TasklistID, s.RoadmapID, s.TasksCreatedCount); err != nil {
		return fmt.Errorf("google: saving sync state: %w", err)
	}
	return nil
}

var _ Repo = (*PgRepo)(nil)
