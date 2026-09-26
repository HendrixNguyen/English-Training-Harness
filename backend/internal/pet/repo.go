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
// never an error — and compute on the live row: no writer takes a Go-side
// pre-image except Save (revive), whose two date columns are
// GREATEST-protected — and every right-hand side reads the pre-image, so the
// shield CASEs see the count before the write.
type Repo interface {
	// Ensure creates the 1:1 row idempotently: INSERT ... ON CONFLICT DO NOTHING.
	Ensure(ctx context.Context, userID string) error
	Get(ctx context.Context, userID string) (State, error)
	// Save writes every mutable column unconditionally except the two verdict
	// dates, which only ever move forward. Revive uses it; the verdict
	// writers do not. It never writes shields / last_shield_used_on (only the
	// verdict writers move them).
	Save(ctx context.Context, userID string, s State) error
	// SaveTargetMet applies §8's success arithmetic in SQL on the live row —
	// +TargetMetHealthBonus capped at MaxHealth, streak+1, stage from the new
	// streak, last_practiced_at = updated_at = now, last_target_met_date =
	// localDate — only while last_target_met_date is NULL or before
	// localDate. Pre-image and write are one statement, like PenaliseMiss: a
	// concurrent miss can no longer be overwritten by a stale Go-side read.
	// ApplyTargetMet is the Go reference this SQL is held to.
	SaveTargetMet(ctx context.Context, userID string, now time.Time, localDate string) (applied bool, err error)
	// PenaliseMiss applies §8's inactivity arithmetic in SQL and advances
	// judged_through to judged, only while judged_through is NULL or earlier.
	// While shields > 0 it spends one instead (health/streak/stage untouched,
	// last_shield_used_on = judged) and reports shielded; ApplyMiss is the Go
	// reference.
	PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (applied, shielded bool, err error)
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
	to_char(p.last_target_met_date, 'YYYY-MM-DD'), to_char(p.judged_through, 'YYYY-MM-DD'),
	p.shields, to_char(p.last_shield_used_on, 'YYYY-MM-DD')`

	getSQL = `SELECT ` + stateColumns + ` FROM pet_states p WHERE p.user_id = $1`

	// GREATEST ignores NULL on either side, so both verdict dates only ever
	// move forward: a revive built from a pre-image that predates a concurrent
	// OnTargetMet cannot erase that day's marker.
	saveSQL = `
UPDATE pet_states
SET health_points = $2, stage = $3::pet_stage, current_streak = $4, last_practiced_at = $5, updated_at = $6,
    last_target_met_date = GREATEST(last_target_met_date, $7::date), judged_through = GREATEST(judged_through, $8::date)
WHERE user_id = $1`

	// Pre-image and write in one statement: every right-hand side reads the
	// row's current values. health' is always > 0 (health ≥ 0 plus the
	// bonus), so the stage CASE needs only StageFor's streak thresholds;
	// integration section 4b pins it to ApplyTargetMet.
	saveTargetMetSQL = `
UPDATE pet_states
SET health_points = LEAST($4, COALESCE(health_points, $4) + $3),
    current_streak = COALESCE(current_streak, 0) + 1,
    stage = (CASE
               WHEN COALESCE(current_streak, 0) + 1 >= 14 THEN 'fruitful'
               WHEN COALESCE(current_streak, 0) + 1 >= 7  THEN 'flowering'
               WHEN COALESCE(current_streak, 0) + 1 >= 3  THEN 'sapling'
               ELSE 'sprout'
             END)::pet_stage,
    -- The award (decision 2): the 7th, 14th … consecutive met day adds a shield, capped.
    shields = LEAST($6, shields + CASE WHEN (COALESCE(current_streak, 0) + 1) % $7 = 0 THEN 1 ELSE 0 END),
    last_practiced_at = $5,
    updated_at = $5,
    last_target_met_date = $2::date
WHERE user_id = $1 AND (last_target_met_date IS NULL OR last_target_met_date < $2::date)`

	// Pre-image and write in one statement: every right-hand side reads the
	// row's current values, so `shields` in each CASE is the count before the
	// write. With a shield held the plant is untouched and the shield is spent
	// (last_shield_used_on = judged); otherwise the §8 arithmetic — the CASE is
	// StageFor(health, 0) for the two stages a streak of 0 can produce. Either
	// way judged_through advances. RETURNING tells the caller which branch ran:
	// last_shield_used_on can equal $2 only if this write set it, because the
	// predicate refuses a day already judged. The integration test pins it to
	// ApplyMiss.
	penaliseMissSQL = `
UPDATE pet_states
SET health_points = CASE WHEN shields > 0 THEN health_points ELSE GREATEST(0, COALESCE(health_points, 100) - $3) END,
    current_streak = CASE WHEN shields > 0 THEN current_streak ELSE 0 END,
    stage = CASE WHEN shields > 0 THEN stage
                 ELSE (CASE WHEN COALESCE(health_points, 100) - $3 <= 0 THEN 'wilted' ELSE 'sprout' END)::pet_stage
            END,
    last_shield_used_on = CASE WHEN shields > 0 THEN $2::date ELSE last_shield_used_on END,
    shields = CASE WHEN shields > 0 THEN shields - 1 ELSE shields END,
    judged_through = $2::date,
    updated_at = $4
WHERE user_id = $1 AND (judged_through IS NULL OR judged_through < $2::date)
RETURNING COALESCE(last_shield_used_on = $2::date, FALSE)`

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
	targets := append(dst, &s.PlantName, &s.HealthPoints, &s.Stage, &s.CurrentStreak, &last, &s.UpdatedAt, &s.LastTargetMetDate, &s.JudgedThrough, &s.Shields, &s.LastShieldUsedOn)
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

func (r *PgRepo) SaveTargetMet(ctx context.Context, userID string, now time.Time, localDate string) (bool, error) {
	tag, err := r.Pool.Exec(ctx, saveTargetMetSQL, userID, localDate, TargetMetHealthBonus, MaxHealth, now, MaxShields, ShieldEveryDays)
	if err != nil {
		return false, fmt.Errorf("pet: saving target met: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PgRepo) PenaliseMiss(ctx context.Context, userID, judged string, now time.Time) (bool, bool, error) {
	var shielded bool
	err := r.Pool.QueryRow(ctx, penaliseMissSQL, userID, judged, MissPenalty, now).Scan(&shielded)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil // already judged: the predicate refused the write
	}
	if err != nil {
		return false, false, fmt.Errorf("pet: penalising miss: %w", err)
	}
	return true, shielded, nil
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
