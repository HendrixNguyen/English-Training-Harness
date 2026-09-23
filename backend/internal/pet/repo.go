package pet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoPet means Get ran for a user with no pet_states row. Service.Ensure
// always inserts first, so callers only see this on a deleted user.
var ErrNoPet = errors.New("pet: no pet_states row")

// Candidate is one pet the hourly sweep may decay: the row plus the timezone
// that decides which local day just ended.
type Candidate struct {
	UserID   string
	Timezone string
	State    State
}

// Repo is the Postgres side. Everything but Timezone/SweepCandidates' join
// touches pet_states (this package's table); users.timezone is the shared
// root table, read the same way quests reads it for day_number.
//
// The three verdict writers are conditional UPDATEs: the predicate on the
// verdict date is what makes OnTargetMet and Sweep safe to run twice, from
// two processes, for the same local day. applied == false is "already done",
// never an error.
type Repo interface {
	// Ensure creates the 1:1 row idempotently: INSERT ... ON CONFLICT DO NOTHING.
	Ensure(ctx context.Context, userID string) error
	Get(ctx context.Context, userID string) (State, error)
	// Save writes every mutable column unconditionally (judged_through only
	// ever forwards). Revive uses it; the two verdict writers below do not.
	Save(ctx context.Context, userID string, s State) error
	// SaveTargetMet writes s only while the row's last_target_met_date is
	// NULL or before s.LastTargetMetDate.
	SaveTargetMet(ctx context.Context, userID string, s State) (applied bool, err error)
	// PenaliseMiss applies §8's inactivity arithmetic in SQL and advances
	// judged_through to judged, only while judged_through is NULL or earlier.
	PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (applied bool, err error)
	// MarkJudged advances judged_through to judged without touching health —
	// the spared day. Same predicate as PenaliseMiss.
	MarkJudged(ctx context.Context, userID, judged string) (applied bool, err error)
	Timezone(ctx context.Context, userID string) (string, error)
	// SweepCandidates returns every pet with its user's timezone; the sweep
	// decides per pet, by civil date, whether there is anything to judge.
	SweepCandidates(ctx context.Context) ([]Candidate, error)
}

const (
	ensureSQL = `INSERT INTO pet_states (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`

	// Every pet_states column except user_id is nullable in §3.2 (DEFAULT
	// without NOT NULL), so COALESCE to the DDL defaults. The two DATE
	// columns are read as YYYY-MM-DD text: the same string quests.LocalDate
	// produces and the service compares.
	stateColumns = `COALESCE(p.plant_name, 'My Green Buddy'), COALESCE(p.health_points, 100), COALESCE(p.stage::text, 'sprout'),
	COALESCE(p.current_streak, 0), p.last_practiced_at, COALESCE(p.updated_at, CURRENT_TIMESTAMP),
	to_char(p.last_target_met_date, 'YYYY-MM-DD'), to_char(p.judged_through, 'YYYY-MM-DD')`

	getSQL = `SELECT ` + stateColumns + ` FROM pet_states p WHERE p.user_id = $1`

	// GREATEST ignores NULL, so a NULL judged_through takes $8 and a later one is kept.
	saveSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6,
    last_target_met_date = $7::date, judged_through = GREATEST(judged_through, $8::date)
WHERE user_id = $1`

	saveTargetMetSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6,
    last_target_met_date = $7::date
WHERE user_id = $1 AND (last_target_met_date IS NULL OR last_target_met_date < $7::date)`

	// Pre-image and write in one statement: health_points on the right-hand
	// side is the row's current value. The CASE is StageFor(health, 0) for
	// the two stages a streak of 0 can produce; the integration test pins it
	// to ApplyMiss.
	penaliseMissSQL = `
UPDATE pet_states
SET health_points = GREATEST(0, COALESCE(health_points, 100) - $3),
    current_streak = 0,
    stage = (CASE WHEN COALESCE(health_points, 100) - $3 <= 0 THEN 'wilted' ELSE 'sprout' END)::pet_stage,
    judged_through = $2::date,
    updated_at = $4
WHERE user_id = $1 AND (judged_through IS NULL OR judged_through < $2::date)`

	markJudgedSQL = `
UPDATE pet_states
SET judged_through = $2::date
WHERE user_id = $1 AND (judged_through IS NULL OR judged_through < $2::date)`

	timezoneSQL = `SELECT COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	candidatesSQL = `
SELECT p.user_id, COALESCE(u.timezone, 'UTC'), ` + stateColumns + `
FROM pet_states p JOIN users u ON u.id = p.user_id`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) Ensure(ctx context.Context, userID string) error {
	if _, err := r.Pool.Exec(ctx, ensureSQL, userID); err != nil {
		return fmt.Errorf("pet: ensuring pet_states row: %w", err)
	}
	return nil
}

func scanState(row pgx.Row, dst ...any) (State, error) {
	var s State
	var last *time.Time
	targets := append(dst, &s.PlantName, &s.HealthPoints, &s.Stage, &s.CurrentStreak, &last, &s.UpdatedAt, &s.LastTargetMetDate, &s.JudgedThrough)
	if err := row.Scan(targets...); err != nil {
		return State{}, err
	}
	if last != nil {
		u := last.UTC()
		s.LastPracticedAt = &u
	}
	s.UpdatedAt = s.UpdatedAt.UTC()
	return s, nil
}

func (r *PgRepo) Get(ctx context.Context, userID string) (State, error) {
	s, err := scanState(r.Pool.QueryRow(ctx, getSQL, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, ErrNoPet
	}
	if err != nil {
		return State{}, fmt.Errorf("pet: reading pet_states: %w", err)
	}
	return s, nil
}

func (r *PgRepo) Save(ctx context.Context, userID string, s State) error {
	tag, err := r.Pool.Exec(ctx, saveSQL, userID, s.HealthPoints, s.Stage, s.CurrentStreak, s.LastPracticedAt, s.UpdatedAt, s.LastTargetMetDate, s.JudgedThrough)
	if err != nil {
		return fmt.Errorf("pet: saving pet_states: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoPet
	}
	return nil
}

func (r *PgRepo) SaveTargetMet(ctx context.Context, userID string, s State) (bool, error) {
	if s.LastTargetMetDate == nil {
		return false, errors.New("pet: SaveTargetMet needs LastTargetMetDate — build the state with ApplyTargetMet")
	}
	tag, err := r.Pool.Exec(ctx, saveTargetMetSQL, userID, s.HealthPoints, s.Stage, s.CurrentStreak, s.LastPracticedAt, s.UpdatedAt, *s.LastTargetMetDate)
	if err != nil {
		return false, fmt.Errorf("pet: saving target met: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepo) PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (bool, error) {
	tag, err := r.Pool.Exec(ctx, penaliseMissSQL, userID, judged, MissPenalty, now)
	if err != nil {
		return false, fmt.Errorf("pet: penalising miss: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepo) MarkJudged(ctx context.Context, userID, judged string) (bool, error) {
	tag, err := r.Pool.Exec(ctx, markJudgedSQL, userID, judged)
	if err != nil {
		return false, fmt.Errorf("pet: marking day judged: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepo) Timezone(ctx context.Context, userID string) (string, error) {
	var tz string
	if err := r.Pool.QueryRow(ctx, timezoneSQL, userID).Scan(&tz); err != nil {
		return "", fmt.Errorf("pet: reading timezone: %w", err)
	}
	return tz, nil
}

func (r *PgRepo) SweepCandidates(ctx context.Context) ([]Candidate, error) {
	rows, err := r.Pool.Query(ctx, candidatesSQL)
	if err != nil {
		return nil, fmt.Errorf("pet: listing sweep candidates: %w", err)
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		s, err := scanState(rows, &c.UserID, &c.Timezone)
		if err != nil {
			return nil, fmt.Errorf("pet: scanning candidate: %w", err)
		}
		c.State = s
		out = append(out, c)
	}
	return out, rows.Err()
}

var _ Repo = (*PgRepo)(nil)
