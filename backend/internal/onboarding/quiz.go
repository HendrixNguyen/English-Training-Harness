package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// QuizStore is the §4 quiz:placement:{user_id} Hash — "transient storage for
// active placement test answers before grading". StageAnswers writes
// question_id → selected_option and sets the TTL; Clear removes the hash
// after a successful assessment. A failed grading leaves the answers
// inspectable for the TTL.
type QuizStore interface {
	StageAnswers(ctx context.Context, userID string, answers []Answer, ttl time.Duration) error
	Clear(ctx context.Context, userID string) error
}

// RedisQuizStore is the real QuizStore, keyed by store.PlacementQuizKey.
type RedisQuizStore struct{ Client *redis.Client }

// NewRedisQuizStore builds a store over an existing client.
func NewRedisQuizStore(r *store.Redis) *RedisQuizStore { return &RedisQuizStore{Client: r.Client} }

func (s *RedisQuizStore) StageAnswers(ctx context.Context, userID string, answers []Answer, ttl time.Duration) error {
	key := store.PlacementQuizKey(userID)
	fields := make([]any, 0, 2*len(answers))
	for _, a := range answers {
		fields = append(fields, a.QuestionID, a.SelectedOption)
	}
	pipe := s.Client.TxPipeline()
	pipe.Del(ctx, key) // a re-take replaces, never merges with, stale answers
	pipe.HSet(ctx, key, fields...)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("onboarding: staging answers: %w", err)
	}
	return nil
}

func (s *RedisQuizStore) Clear(ctx context.Context, userID string) error {
	if err := s.Client.Del(ctx, store.PlacementQuizKey(userID)).Err(); err != nil {
		return fmt.Errorf("onboarding: clearing quiz: %w", err)
	}
	return nil
}

var _ QuizStore = (*RedisQuizStore)(nil)
