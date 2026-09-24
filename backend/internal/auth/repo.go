package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// User is the slice of spec §3.2 `users` that sign-in reads back.
type User struct {
	ID          string
	Email       string
	FullName    string
	CEFRCurrent string
}

// defaultTargetGoal is what a brand-new user gets. §3.2 makes target_goal
// NOT NULL with no default, but the goal is only known after onboarding, so
// sign-in writes the empty string and onboarding fills it in later.
const defaultTargetGoal = ""

// UserRepo is the Postgres side of sign-in, so Service can be tested with a fake.
type UserRepo interface {
	// UpsertByGoogleID creates or refreshes the user and returns the stored row.
	UpsertByGoogleID(ctx context.Context, googleID, email, fullName, refreshToken string) (User, error)
}

// upsertUserSQL creates the user on first sign-in and refreshes the Google-owned
// fields afterwards. cefr_current and target_goal are deliberately absent from
// the UPDATE: re-login must not reset a learner's level or goal. An empty
// refresh token (Google omits it on silent re-consent) keeps the stored value.
const upsertUserSQL = `
INSERT INTO users (email, full_name, google_id, google_refresh_token, target_goal)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (google_id) DO UPDATE SET
    email = EXCLUDED.email,
    full_name = EXCLUDED.full_name,
    google_refresh_token = COALESCE(NULLIF(EXCLUDED.google_refresh_token, ''), users.google_refresh_token)
RETURNING id, email, full_name, cefr_current`

// Sealer encrypts a refresh token before it is stored (backend spec §6.1,
// §7: AES-256-GCM under ENCRYPTION_SECRET_KEY). *secrets.Box satisfies it.
type Sealer interface {
	Seal(plain string) (string, error)
}

// PgUserRepo is the real UserRepo. It seals google_refresh_token on the way
// in; google.PgRefreshTokenSource opens it on the way out.
type PgUserRepo struct {
	Pool   *pgxpool.Pool
	sealer Sealer
}

// NewPgUserRepo builds a repo over an existing pool and the process's sealer.
func NewPgUserRepo(pool *pgxpool.Pool, sealer Sealer) *PgUserRepo {
	return &PgUserRepo{Pool: pool, sealer: sealer}
}

func (r *PgUserRepo) UpsertByGoogleID(ctx context.Context, googleID, email, fullName, refreshToken string) (User, error) {
	// Google omits the refresh token on silent re-consent; "" must reach the
	// SQL unsealed so COALESCE(NULLIF(…, ''), users.google_refresh_token)
	// keeps the stored (sealed) value. Anything else is sealed here — the
	// plaintext never reaches Postgres.
	stored := ""
	if refreshToken != "" {
		sealed, err := r.sealer.Seal(refreshToken)
		if err != nil {
			return User{}, fmt.Errorf("auth: sealing refresh token: %w", err)
		}
		stored = sealed
	}

	// full_name and cefr_current are nullable in §3.2, so scan through pointers
	// and flatten NULL to the zero value.
	var (
		u        User
		nullName *string
		nullCEFR *string
	)
	err := r.Pool.QueryRow(ctx, upsertUserSQL,
		email, fullName, googleID, stored, defaultTargetGoal,
	).Scan(&u.ID, &u.Email, &nullName, &nullCEFR)
	if err != nil {
		return User{}, fmt.Errorf("auth: upserting user: %w", err)
	}
	if nullName != nil {
		u.FullName = *nullName
	}
	if nullCEFR != nil {
		u.CEFRCurrent = *nullCEFR
	}
	return u, nil
}

var _ UserRepo = (*PgUserRepo)(nil)
