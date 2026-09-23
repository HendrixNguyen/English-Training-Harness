package google

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoRefreshToken means users.google_refresh_token is NULL or empty: the
// user signed in before offline access was requested, or Google omitted it.
// Service maps it to ErrReauthRequired.
var ErrNoRefreshToken = errors.New("google: no refresh token on file")

// RefreshTokenSource is the ONLY way this package reads
// users.google_refresh_token. Today's single implementation returns the
// column as stored (plaintext — what the merged auth slice writes). Backend
// spec §7 wants AES-256-GCM at rest via ENCRYPTION_SECRET_KEY; that fix
// (inbox bug "google_refresh_token is stored in plaintext") replaces this
// implementation with a decrypting one and changes nothing else here.
type RefreshTokenSource interface {
	RefreshToken(ctx context.Context, userID string) (string, error)
}

// PgRefreshTokenSource reads the column verbatim.
type PgRefreshTokenSource struct{ Pool *pgxpool.Pool }

// NewPgRefreshTokenSource builds the source over an existing pool.
func NewPgRefreshTokenSource(pool *pgxpool.Pool) *PgRefreshTokenSource {
	return &PgRefreshTokenSource{Pool: pool}
}

const refreshTokenSQL = `SELECT COALESCE(google_refresh_token, '') FROM users WHERE id = $1`

func (s *PgRefreshTokenSource) RefreshToken(ctx context.Context, userID string) (string, error) {
	var tok string
	if err := s.Pool.QueryRow(ctx, refreshTokenSQL, userID).Scan(&tok); err != nil {
		return "", fmt.Errorf("google: reading refresh token: %w", err)
	}
	if tok == "" {
		return "", ErrNoRefreshToken
	}
	return tok, nil
}

var _ RefreshTokenSource = (*PgRefreshTokenSource)(nil)
