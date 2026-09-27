package notify

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// DueBatchSize caps one Tick. At a 30 s poll that is 200 users/minute; a
// bigger backlog simply drains over several ticks.
const DueBatchSize = 100

// Queue is notify's Redis side: the §4 ZSET plus the per-subscription
// failure counter, so NewService needs no extra wiring.
type Queue interface {
	// Schedule sets the user's next send time (ZADD; overwrites the score, so
	// a user is never in the set twice).
	Schedule(ctx context.Context, userID string, at time.Time) error
	// Due returns up to limit members whose score is <= now, oldest first.
	Due(ctx context.Context, now time.Time, limit int64) ([]string, error)
	// Remove drops the user from the set (no subscriptions left, or no user).
	Remove(ctx context.Context, userID string) error
	// RecordFailure increments the subscription's consecutive-failure
	// counter (store.PushFailKey), refreshes its TTL, and returns the new
	// count.
	RecordFailure(ctx context.Context, subscriptionID string) (int64, error)
	// ClearFailures resets the subscription's counter (a successful send, or
	// the row being pruned).
	ClearFailures(ctx context.Context, subscriptionID string) error
}

// RedisQueue is the real Queue.
type RedisQueue struct{ Client *redis.Client }

// NewRedisQueue builds a queue over an existing client.
func NewRedisQueue(r *store.Redis) *RedisQueue { return &RedisQueue{Client: r.Client} }

func (q *RedisQueue) Schedule(ctx context.Context, userID string, at time.Time) error {
	if err := q.Client.ZAdd(ctx, store.WebPushDelayQueueKey, redis.Z{Score: float64(at.Unix()), Member: userID}).Err(); err != nil {
		return fmt.Errorf("notify: scheduling reminder: %w", err)
	}
	return nil
}

func (q *RedisQueue) Due(ctx context.Context, now time.Time, limit int64) ([]string, error) {
	members, err := q.Client.ZRangeByScore(ctx, store.WebPushDelayQueueKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(now.Unix(), 10),
		Count: limit,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("notify: reading due reminders: %w", err)
	}
	return members, nil
}

func (q *RedisQueue) Remove(ctx context.Context, userID string) error {
	if err := q.Client.ZRem(ctx, store.WebPushDelayQueueKey, userID).Err(); err != nil {
		return fmt.Errorf("notify: removing reminder: %w", err)
	}
	return nil
}

func (q *RedisQueue) RecordFailure(ctx context.Context, subscriptionID string) (int64, error) {
	key := store.PushFailKey(subscriptionID)
	incr := q.Client.TxPipeline()
	res := incr.Incr(ctx, key)
	incr.Expire(ctx, key, store.PushFailTTL)
	if _, err := incr.Exec(ctx); err != nil {
		return 0, fmt.Errorf("notify: recording send failure: %w", err)
	}
	return res.Val(), nil
}

func (q *RedisQueue) ClearFailures(ctx context.Context, subscriptionID string) error {
	if err := q.Client.Del(ctx, store.PushFailKey(subscriptionID)).Err(); err != nil {
		return fmt.Errorf("notify: clearing send failures: %w", err)
	}
	return nil
}

var _ Queue = (*RedisQueue)(nil)
