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
// question_id → selected_option and sets the TTL; StageLevel adds the graded
// level under levelField (same TTL) so a re-submit after a failed roadmap step
// is not graded again; StagedLevel returns it only for exactly the staged
// answers; Clear removes the hash after a successful assessment. A failed
// grading leaves the answers inspectable for the TTL.
type QuizStore interface {
	StageAnswers(ctx context.Context, userID string, answers []Answer, ttl time.Duration) error
	StageLevel(ctx context.Context, userID, level string, ttl time.Duration) error
	// StagedLevel is "" when nothing is staged or the answers differ.
	StagedLevel(ctx context.Context, userID string, answers []Answer) (string, error)
	Clear(ctx context.Context, userID string) error
}

// levelField holds the graded level beside the answers. Bank ids are q1..q10
// and validate() rejects anything else, so it can never collide with one.
const levelField = "_level"

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

func (s *RedisQuizStore) StageLevel(ctx context.Context, userID, level string, ttl time.Duration) error {
	key := store.PlacementQuizKey(userID)
	pipe := s.Client.TxPipeline()
	pipe.HSet(ctx, key, levelField, level)
	pipe.Expire(ctx, key, ttl) // HSET on an expired key would otherwise create one without a TTL
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("onboarding: staging level: %w", err)
	}
	return nil
}

func (s *RedisQuizStore) StagedLevel(ctx context.Context, userID string, answers []Answer) (string, error) {
	fields, err := s.Client.HGetAll(ctx, store.PlacementQuizKey(userID)).Result() // empty map for a missing key
	if err != nil {
		return "", fmt.Errorf("onboarding: reading staged level: %w", err)
	}
	level := fields[levelField]
	if level == "" || len(fields)-1 != len(answers) {
		return "", nil
	}
	for _, a := range answers {
		if fields[a.QuestionID] != a.SelectedOption {
			return "", nil
		}
	}
	return level, nil
}

func (s *RedisQuizStore) Clear(ctx context.Context, userID string) error {
	if err := s.Client.Del(ctx, store.PlacementQuizKey(userID)).Err(); err != nil {
		return fmt.Errorf("onboarding: clearing quiz: %w", err)
	}
	return nil
}

var _ QuizStore = (*RedisQuizStore)(nil)
