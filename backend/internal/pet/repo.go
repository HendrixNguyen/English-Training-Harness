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

// Repo is the Postgres side. Ensure/Get/Save touch pet_states (this package's
// table). Timezone/Timezones/SweepCandidates read users.timezone — the shared
// root table, read the same way quests reads it for day_number.
type Repo interface {
	// Ensure creates the 1:1 row idempotently: INSERT ... ON CONFLICT DO NOTHING.
	Ensure(ctx context.Context, userID string) error
	Get(ctx context.Context, userID string) (State, error)
	// Save writes health, stage, streak, last_practiced_at and updated_at.
	Save(ctx context.Context, userID string, s State) error
	Timezone(ctx context.Context, userID string) (string, error)
	// Timezones lists the distinct users.timezone values of users who have a pet.
	Timezones(ctx context.Context) ([]string, error)
	// SweepCandidates returns every pet whose user is in one of the timezones.
	SweepCandidates(ctx context.Context, timezones []string) ([]Candidate, error)
}

const (
	ensureSQL = `INSERT INTO pet_states (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`

	// Every pet_states column except user_id is nullable in §3.2 (DEFAULT
	// without NOT NULL), so COALESCE to the DDL defaults.
	stateColumns = `COALESCE(p.plant_name, 'My Green Buddy'), COALESCE(p.health_points, 100), COALESCE(p.stage::text, 'sprout'),
	COALESCE(p.current_streak, 0), p.last_practiced_at, COALESCE(p.updated_at, CURRENT_TIMESTAMP)`

	getSQL = `SELECT ` + stateColumns + ` FROM pet_states p WHERE p.user_id = $1`

	saveSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6
WHERE user_id = $1`

	timezoneSQL = `SELECT COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	timezonesSQL = `
SELECT DISTINCT COALESCE(u.timezone, 'UTC')
FROM users u JOIN pet_states p ON p.user_id = u.id`

	candidatesSQL = `
SELECT p.user_id, COALESCE(u.timezone, 'UTC'), ` + stateColumns + `
FROM pet_states p JOIN users u ON u.id = p.user_id
WHERE COALESCE(u.timezone, 'UTC') = ANY($1)`
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
	targets := append(dst, &s.PlantName, &s.HealthPoints, &s.Stage, &s.CurrentStreak, &last, &s.UpdatedAt)
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
	tag, err := r.Pool.Exec(ctx, saveSQL, userID, s.HealthPoints, s.Stage, s.CurrentStreak, s.LastPracticedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("pet: saving pet_states: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoPet
	}
	return nil
}

func (r *PgRepo) Timezone(ctx context.Context, userID string) (string, error) {
	var tz string
	if err := r.Pool.QueryRow(ctx, timezoneSQL, userID).Scan(&tz); err != nil {
		return "", fmt.Errorf("pet: reading timezone: %w", err)
	}
	return tz, nil
}

func (r *PgRepo) Timezones(ctx context.Context) ([]string, error) {
	rows, err := r.Pool.Query(ctx, timezonesSQL)
	if err != nil {
		return nil, fmt.Errorf("pet: listing timezones: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var tz string
		if err := rows.Scan(&tz); err != nil {
			return nil, fmt.Errorf("pet: scanning timezone: %w", err)
		}
		out = append(out, tz)
	}
	return out, rows.Err()
}

func (r *PgRepo) SweepCandidates(ctx context.Context, timezones []string) ([]Candidate, error) {
	if len(timezones) == 0 {
		return nil, nil
	}
	rows, err := r.Pool.Query(ctx, candidatesSQL, timezones)
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
