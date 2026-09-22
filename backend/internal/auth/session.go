package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// ErrNoSession means the sess:{user_id}:token key is absent — the session was
// revoked or has expired.
var ErrNoSession = errors.New("auth: no active session")

// SessionStore is the Redis side of sign-in. Deleting a user's key revokes
// their session even though the JWT itself is still within its exp window.
type SessionStore interface {
	Put(ctx context.Context, userID, token string, ttl time.Duration) error
	Get(ctx context.Context, userID string) (string, error)
	Delete(ctx context.Context, userID string) error
}

// RedisSessionStore is the real SessionStore, keyed by store.SessionKey (§4).
type RedisSessionStore struct{ Client *redis.Client }

// NewRedisSessionStore builds a store over an existing client.
func NewRedisSessionStore(r *store.Redis) *RedisSessionStore {
	return &RedisSessionStore{Client: r.Client}
}

func (s *RedisSessionStore) Put(ctx context.Context, userID, token string, ttl time.Duration) error {
	if err := s.Client.Set(ctx, store.SessionKey(userID), token, ttl).Err(); err != nil {
		return fmt.Errorf("auth: storing session: %w", err)
	}
	return nil
}

func (s *RedisSessionStore) Get(ctx context.Context, userID string) (string, error) {
	v, err := s.Client.Get(ctx, store.SessionKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNoSession
	}
	if err != nil {
		return "", fmt.Errorf("auth: reading session: %w", err)
	}
	return v, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, userID string) error {
	if err := s.Client.Del(ctx, store.SessionKey(userID)).Err(); err != nil {
		return fmt.Errorf("auth: deleting session: %w", err)
	}
	return nil
}

var _ SessionStore = (*RedisSessionStore)(nil)
