package pet

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
)

// ErrNotWilted means POST /pet/revive was called while health_points > 0.
var ErrNotWilted = errors.New("pet: revive requires a wilted plant")

// ReviveResult is what POST /pet/revive reports (backend spec §6.3).
type ReviveResult struct {
	Passed bool
	State  State
}

// Service is the plant engine over its collaborators.
type Service struct {
	repo       Repo
	challenges ChallengeStore
	study      StudyCounter
	now        func() time.Time
}

// NewService wires the collaborators; now is injectable for tests.
func NewService(repo Repo, challenges ChallengeStore, study StudyCounter, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, challenges: challenges, study: study, now: now}
}

// Ensure creates the user's pet_states row if it is missing (idempotent:
// INSERT ... ON CONFLICT (user_id) DO NOTHING) and returns the current state.
// It is what GET /pet/status and onboarding call first, which answers the
// "who creates the row" gap in the spec for the code.
func (s *Service) Ensure(ctx context.Context, userID string) (State, error) {
	if err := s.repo.Ensure(ctx, userID); err != nil {
		return State{}, err
	}
	return s.repo.Get(ctx, userID)
}

// OnTargetMet is §8's success logic, applied once per local day: quests fires
// it on the progress call that crosses 1800s (backend spec §6.2). localDate
// is informational — the row is not keyed by day.
func (s *Service) OnTargetMet(ctx context.Context, userID, localDate string) error {
	st, err := s.Ensure(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Save(ctx, userID, ApplyTargetMet(st, s.now(), localDate))
}

// Revive implements the 15-minute revival challenge behind POST /pet/revive.
//
// Only a wilted plant (health 0) may be revived; otherwise ErrNotWilted (409).
// The first call on a local day starts a challenge, recording the daily
// counter's current value; each later call the same day checks whether
// ReviveSeconds more have been recorded through POST /quests/progress. On pass
// the state becomes 50 / sprout / 0 (§6.3) and the challenge is cleared. A
// challenge left over from an earlier local day is replaced.
func (s *Service) Revive(ctx context.Context, userID string) (ReviveResult, error) {
	st, err := s.Ensure(ctx, userID)
	if err != nil {
		return ReviveResult{}, err
	}
	if st.HealthPoints > 0 {
		return ReviveResult{}, ErrNotWilted
	}

	tz, err := s.repo.Timezone(ctx, userID)
	if err != nil {
		return ReviveResult{}, err
	}
	now := s.now()
	today := quests.LocalDate(now, quests.Location(tz))

	c, ok, err := s.challenges.Get(ctx, userID)
	if err != nil {
		return ReviveResult{}, err
	}
	if !ok || c.LocalDate != today {
		start, err := s.study.Total(ctx, userID, today)
		if err != nil {
			return ReviveResult{}, err
		}
		if err := s.challenges.Start(ctx, userID, Challenge{StartedAt: now, LocalDate: today, StartSeconds: start}); err != nil {
			return ReviveResult{}, err
		}
		return ReviveResult{Passed: false, State: st}, nil
	}

	total, err := s.study.Total(ctx, userID, c.LocalDate)
	if err != nil {
		return ReviveResult{}, err
	}
	if total-c.StartSeconds < ReviveSeconds {
		return ReviveResult{Passed: false, State: st}, nil
	}

	st = ApplyRevive(st, now, today)
	if err := s.repo.Save(ctx, userID, st); err != nil {
		return ReviveResult{}, err
	}
	if err := s.challenges.Clear(ctx, userID); err != nil {
		// The pass is already persisted; a stale key only expires later.
		log.Printf("pet: clearing revive challenge for %s: %v", userID, err)
	}
	return ReviveResult{Passed: true, State: st}, nil
}

// Sweep is the body of the §8 hourly cron. It runs at :00 UTC; a user "hits
// local midnight" when now in their timezone is in hour 0. For each such pet
// whose previous local day recorded fewer than 1800s it applies ApplyMiss.
//
// Idempotency: a pet whose updated_at is already at or after that local
// midnight has been touched this local day (by an earlier run of this same
// sweep after a restart, or by a target met after midnight) and is skipped.
// Errors on one pet are collected and the rest are still processed; the
// count returned is the number of pets penalised.
func (s *Service) Sweep(ctx context.Context, now time.Time) (int, error) {
	cands, err := s.repo.SweepCandidates(ctx)
	if err != nil {
		return 0, err
	}

	penalised := 0
	var errs []error
	for _, c := range cands {
		loc := quests.Location(c.Timezone)
		local := now.In(loc)
		midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		if !c.State.UpdatedAt.Before(midnight) {
			continue // already handled this local day
		}
		yesterday := midnight.AddDate(0, 0, -1).Format("2006-01-02")
		total, err := s.study.Total(ctx, c.UserID, yesterday)
		if err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			continue
		}
		if total >= quests.TargetSeconds {
			continue
		}
		if _, err := s.repo.PenaliseMiss(ctx, c.UserID, yesterday, now); err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			continue
		}
		penalised++
	}
	return penalised, errors.Join(errs...)
}
