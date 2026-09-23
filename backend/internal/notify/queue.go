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

// Queue is the §4 queue:webpush:delay ZSET: member = user id, score = the
// UNIX time of the next reminder. It is persistent (no TTL).
type Queue interface {
	// Schedule sets the user's next send time (ZADD; overwrites the score, so
	// a user is never in the set twice).
	Schedule(ctx context.Context, userID string, at time.Time) error
	// Due returns up to limit members whose score is <= now, oldest first.
	Due(ctx context.Context, now time.Time, limit int64) ([]string, error)
	// Remove drops the user from the set (no subscriptions left, or no user).
	Remove(ctx context.Context, userID string) error
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

var _ Queue = (*RedisQueue)(nil)
