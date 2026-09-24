package google

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoRefreshToken means there is no usable refresh token on file: the
// column is NULL/empty, or it holds a value this process cannot open (a
// pre-encryption plaintext row, a tampered value, another key). Service maps
// it to ErrReauthRequired: the PWA sends the user back through consent and
// auth stores a fresh sealed token — the cutover needs no data migration.
var ErrNoRefreshToken = errors.New("google: no refresh token on file")

// Opener decrypts what auth sealed (backend spec §7). *secrets.Box satisfies it.
type Opener interface {
	Open(sealed string) (string, error)
}

// RefreshTokenSource is the ONLY way this package reads
// users.google_refresh_token.
type RefreshTokenSource interface {
	RefreshToken(ctx context.Context, userID string) (string, error)
}

// PgRefreshTokenSource reads the sealed column and opens it.
type PgRefreshTokenSource struct {
	Pool   *pgxpool.Pool
	opener Opener
}

// NewPgRefreshTokenSource builds the source over an existing pool and the
// process's opener (the same Box auth seals with).
func NewPgRefreshTokenSource(pool *pgxpool.Pool, opener Opener) *PgRefreshTokenSource {
	return &PgRefreshTokenSource{Pool: pool, opener: opener}
}

const refreshTokenSQL = `SELECT COALESCE(google_refresh_token, '') FROM users WHERE id = $1`

func (s *PgRefreshTokenSource) RefreshToken(ctx context.Context, userID string) (string, error) {
	var stored string
	if err := s.Pool.QueryRow(ctx, refreshTokenSQL, userID).Scan(&stored); err != nil {
		return "", fmt.Errorf("google: reading refresh token: %w", err)
	}
	return openStored(s.opener, stored)
}

// openStored maps the raw column to a usable token or ErrNoRefreshToken.
func openStored(opener Opener, stored string) (string, error) {
	if stored == "" {
		return "", ErrNoRefreshToken
	}
	plain, err := opener.Open(stored)
	if err != nil {
		return "", fmt.Errorf("%w: stored value unusable (%v)", ErrNoRefreshToken, err)
	}
	return plain, nil
}

var _ RefreshTokenSource = (*PgRefreshTokenSource)(nil)
