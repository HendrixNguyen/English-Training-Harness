package quests

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Counter is the Redis side of §5.2 step 2. Add returns the running total for
// the day *after* adding, which is what decides whether the 30-minute target has
// just been crossed.
type Counter interface {
	Add(ctx context.Context, userID, localDate string, seconds int64) (total int64, err error)
	Total(ctx context.Context, userID, localDate string) (int64, error)
}

// counterKey builds the §4 daily:accumulated key from an already-localised date,
// so the counter can never fall back to server time by accident.
func counterKey(userID, localDate string) string {
	d, err := time.Parse("2006-01-02", localDate)
	if err != nil {
		// Callers always pass LocalDate output; a bad value would silently
		// merge days, so fail loudly rather than guess.
		panic(fmt.Sprintf("quests: counterKey got a malformed date %q", localDate))
	}
	return store.DailyAccumulatedKey(userID, d)
}

// RedisCounter is the real Counter.
type RedisCounter struct{ Client *redis.Client }

// NewRedisCounter builds a counter over an existing client.
func NewRedisCounter(r *store.Redis) *RedisCounter { return &RedisCounter{Client: r.Client} }

// Add does INCRBY then EXPIRE, in that order, in one pipeline. EXPIRE is
// re-applied on every write so the 48h window slides with activity.
func (c *RedisCounter) Add(ctx context.Context, userID, localDate string, seconds int64) (int64, error) {
	key := counterKey(userID, localDate)

	pipe := c.Client.TxPipeline()
	incr := pipe.IncrBy(ctx, key, seconds)
	pipe.Expire(ctx, key, store.DailyAccumulatedTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("quests: incrementing daily counter: %w", err)
	}
	return incr.Val(), nil
}

// Total reads the counter without changing it. A missing key means zero.
func (c *RedisCounter) Total(ctx context.Context, userID, localDate string) (int64, error) {
	v, err := c.Client.Get(ctx, counterKey(userID, localDate)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("quests: reading daily counter: %w", err)
	}
	return v, nil
}

var _ Counter = (*RedisCounter)(nil)
