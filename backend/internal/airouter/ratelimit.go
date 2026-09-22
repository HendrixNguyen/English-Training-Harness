package airouter

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// AILimitPerMinute is the §4 cap on ratelimit:ai:{user_id}.
const AILimitPerMinute = 5

// RateLimiter gates AI calls per user. Allow returns nil when the call may
// proceed and ErrRateLimited when the user is over the limit.
type RateLimiter interface {
	Allow(ctx context.Context, userID string) error
}

// RedisRateLimiter is the real RateLimiter: INCR the §4 key and set its TTL
// only if it has none (EXPIRE NX, Redis ≥ 7), both in one pipeline, so a
// crash between the two commands can never leave a key that never expires.
type RedisRateLimiter struct {
	Client *redis.Client
	Limit  int64
}

// NewRedisRateLimiter builds a limiter over an existing client at the §4 limit.
func NewRedisRateLimiter(r *store.Redis) *RedisRateLimiter {
	return &RedisRateLimiter{Client: r.Client, Limit: AILimitPerMinute}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, userID string) error {
	key := store.AIRateLimitKey(userID)
	pipe := l.Client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, store.AIRateLimitTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("airouter: rate limit: %w", err)
	}
	if incr.Val() > l.Limit {
		return ErrRateLimited
	}
	return nil
}

var _ RateLimiter = (*RedisRateLimiter)(nil)
