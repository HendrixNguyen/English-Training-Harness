package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// TokenTTL is both the JWT exp window and the Redis session TTL, so the two can
// never drift (spec §4: sess:{user_id}:token, 24 hours).
const TokenTTL = store.SessionTTL

// TokenIssuer signs and verifies session JWTs (HS256).
type TokenIssuer struct {
	secret []byte
	now    func() time.Time
}

// NewTokenIssuer builds an issuer. now is injectable so expiry is testable.
func NewTokenIssuer(secret string, now func() time.Time) *TokenIssuer {
	if now == nil {
		now = time.Now
	}
	return &TokenIssuer{secret: []byte(secret), now: now}
}

// Issue returns a signed token whose subject is userID.
func (t *TokenIssuer) Issue(userID string) (string, error) {
	now := t.now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("auth: signing token: %w", err)
	}
	return signed, nil
}

// Verify checks the signature and expiry and returns the subject.
func (t *TokenIssuer) Verify(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{},
		func(tok *jwt.Token) (any, error) {
			if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("auth: unexpected signing method %v", tok.Header["alg"])
			}
			return t.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return "", fmt.Errorf("auth: verifying token: %w", err)
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", fmt.Errorf("auth: token has no subject")
	}
	return claims.Subject, nil
}
