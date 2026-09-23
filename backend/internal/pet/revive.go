package pet

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// Challenge is the active revival attempt: the user must record ReviveSeconds
// of study (via POST /quests/progress) on LocalDate, counted from StartSeconds
// — the daily counter's value when the challenge began.
type Challenge struct {
	StartedAt    time.Time
	LocalDate    string
	StartSeconds int64
}

// ChallengeStore keeps the one active challenge per user under
// store.PetReviveKey with store.PetReviveTTL.
type ChallengeStore interface {
	Start(ctx context.Context, userID string, c Challenge) error
	// Get returns ok=false when there is no active challenge.
	Get(ctx context.Context, userID string) (c Challenge, ok bool, err error)
	Clear(ctx context.Context, userID string) error
}

// StudyCounter is pet's read-only view of the daily study counter. It is
// satisfied by *quests.RedisCounter (Total), so pet never builds quests' key
// or reads its tables — packages talk via interfaces (CODEMAP).
type StudyCounter interface {
	Total(ctx context.Context, userID, localDate string) (int64, error)
}

// RedisChallengeStore is the real ChallengeStore: a Hash with three fields.
type RedisChallengeStore struct{ Client *redis.Client }

// NewRedisChallengeStore builds a store over an existing client.
func NewRedisChallengeStore(r *store.Redis) *RedisChallengeStore {
	return &RedisChallengeStore{Client: r.Client}
}

func (s *RedisChallengeStore) Start(ctx context.Context, userID string, c Challenge) error {
	key := store.PetReviveKey(userID)
	pipe := s.Client.TxPipeline()
	pipe.HSet(ctx, key,
		"started_at", c.StartedAt.UTC().Format(time.RFC3339),
		"local_date", c.LocalDate,
		"start_seconds", strconv.FormatInt(c.StartSeconds, 10),
	)
	pipe.Expire(ctx, key, store.PetReviveTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("pet: starting revive challenge: %w", err)
	}
	return nil
}

func (s *RedisChallengeStore) Get(ctx context.Context, userID string) (Challenge, bool, error) {
	m, err := s.Client.HGetAll(ctx, store.PetReviveKey(userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return Challenge{}, false, fmt.Errorf("pet: reading revive challenge: %w", err)
	}
	if len(m) == 0 {
		return Challenge{}, false, nil
	}
	started, err := time.Parse(time.RFC3339, m["started_at"])
	if err != nil {
		return Challenge{}, false, fmt.Errorf("pet: malformed started_at %q: %w", m["started_at"], err)
	}
	start, err := strconv.ParseInt(m["start_seconds"], 10, 64)
	if err != nil {
		return Challenge{}, false, fmt.Errorf("pet: malformed start_seconds %q: %w", m["start_seconds"], err)
	}
	return Challenge{StartedAt: started, LocalDate: m["local_date"], StartSeconds: start}, true, nil
}

func (s *RedisChallengeStore) Clear(ctx context.Context, userID string) error {
	if err := s.Client.Del(ctx, store.PetReviveKey(userID)).Err(); err != nil {
		return fmt.Errorf("pet: clearing revive challenge: %w", err)
	}
	return nil
}

var _ ChallengeStore = (*RedisChallengeStore)(nil)
