package notify

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound means the users row is gone (the session outlived it).
var ErrUserNotFound = errors.New("notify: user not found")

// Subscription is a §3.2 push_subscriptions row — the PushSubscription the
// browser handed the PWA. ID is empty on input.
type Subscription struct {
	ID       string
	Endpoint string
	P256dh   string
	Auth     string
}

// Preferences is the slice of users this package reads.
type Preferences struct {
	NotificationTime string // "HH:MM:SS"
	Timezone         string // IANA name, 'UTC' when NULL
}

// Repo is the Postgres side. Everything is scoped by user id.
type Repo interface {
	// UpdatePreferences writes notification_time and, when timezone is
	// non-empty, timezone. ErrUserNotFound when no row matched.
	UpdatePreferences(ctx context.Context, userID, notificationTime, timezone string) error
	Preferences(ctx context.Context, userID string) (Preferences, error)
	// SaveSubscription stores s for userID exactly once per endpoint and
	// takes the endpoint away from any other user that had it.
	SaveSubscription(ctx context.Context, userID string, s Subscription) error
	Subscriptions(ctx context.Context, userID string) ([]Subscription, error)
	DeleteSubscription(ctx context.Context, id string) error
}

const (
	updatePreferencesSQL = `
UPDATE users
SET notification_time = $2::time,
    timezone = COALESCE(NULLIF($3, ''), timezone)
WHERE id = $1`

	preferencesSQL = `SELECT COALESCE(notification_time::text, '20:00:00'), COALESCE(timezone, 'UTC') FROM users WHERE id = $1`

	// §3.2 has no UNIQUE on endpoint, so no ON CONFLICT: the CTE moves the
	// endpoint away from other users, the INSERT adds it for this user only
	// if absent. One statement, so it is atomic.
	saveSubscriptionSQL = `
WITH moved AS (
    DELETE FROM push_subscriptions WHERE endpoint = $2 AND user_id <> $1
)
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
SELECT $1, $2, $3, $4
WHERE NOT EXISTS (SELECT 1 FROM push_subscriptions WHERE endpoint = $2 AND user_id = $1)`

	subscriptionsSQL = `SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at, id`

	deleteSubscriptionSQL = `DELETE FROM push_subscriptions WHERE id = $1`
)

// PgRepo is the real Repo.
type PgRepo struct{ Pool *pgxpool.Pool }

// NewPgRepo builds a repo over an existing pool.
func NewPgRepo(pool *pgxpool.Pool) *PgRepo { return &PgRepo{Pool: pool} }

func (r *PgRepo) UpdatePreferences(ctx context.Context, userID, notificationTime, timezone string) error {
	tag, err := r.Pool.Exec(ctx, updatePreferencesSQL, userID, notificationTime, timezone)
	if err != nil {
		return fmt.Errorf("notify: updating preferences: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PgRepo) Preferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.Pool.QueryRow(ctx, preferencesSQL, userID).Scan(&p.NotificationTime, &p.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrUserNotFound
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("notify: reading preferences: %w", err)
	}
	return p, nil
}

func (r *PgRepo) SaveSubscription(ctx context.Context, userID string, s Subscription) error {
	if _, err := r.Pool.Exec(ctx, saveSubscriptionSQL, userID, s.Endpoint, s.P256dh, s.Auth); err != nil {
		return fmt.Errorf("notify: saving subscription: %w", err)
	}
	return nil
}

func (r *PgRepo) Subscriptions(ctx context.Context, userID string) ([]Subscription, error) {
	rows, err := r.Pool.Query(ctx, subscriptionsSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("notify: reading subscriptions: %w", err)
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, fmt.Errorf("notify: scanning subscription: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PgRepo) DeleteSubscription(ctx context.Context, id string) error {
	if _, err := r.Pool.Exec(ctx, deleteSubscriptionSQL, id); err != nil {
		return fmt.Errorf("notify: deleting subscription: %w", err)
	}
	return nil
}

var _ Repo = (*PgRepo)(nil)
