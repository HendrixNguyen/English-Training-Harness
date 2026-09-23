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

// OnTargetMet is §8's success logic for the user's local day localDate:
// quests fires it when the day's total first reaches 1800s (backend spec
// §6.2), and may fire it again after a failure on the same call or after a
// lost Redis counter. The pet owns the once: Repo.SaveTargetMet's predicate
// on last_target_met_date refuses a second write for the same (or an
// earlier) local date, and that refusal is a silent no-op — quests logs hook
// errors, and "already counted" is not one. There is deliberately no Go-side
// pre-check: one mechanism, in the database, is what the tests pin.
func (s *Service) OnTargetMet(ctx context.Context, userID, localDate string) error {
	st, err := s.Ensure(ctx, userID)
	if err != nil {
		return err
	}
	_, err = s.repo.SaveTargetMet(ctx, userID, ApplyTargetMet(st, s.now(), localDate))
	return err
}

// Revive implements the 15-minute revival challenge behind POST /pet/revive.
//
// Only a wilted plant (health 0) may be revived; otherwise ErrNotWilted (409).
// The first call on a local day starts a challenge, recording the daily
// counter's current value; each later call the same day checks whether
// ReviveSeconds more have been recorded through POST /quests/progress. On pass
// the state becomes 50 / sprout / 0 (§6.3), the local day is resolved so that
// night's sweep applies no miss for it (judged_through = today), and the
// challenge is cleared. A challenge left over from an earlier local day is
// replaced.
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

// Sweep is the body of the §8 hourly cron. It runs at every :00 UTC and looks
// at every pet: for each, the local day that most recently ended is
// judged = PreviousDate(LocalDate(now, tz)), and there is work only while
// judged_through is before it. That replaces "local hour is 0": a zone whose
// clocks jump 23:59:59 → 01:00 is judged at 01:00, a zone whose hour 0
// happens twice is judged once, and a tick the process slept through is
// caught up at the next one. No midnight instant is ever constructed.
//
// A day is spared when the pet's own marker says its target was met
// (last_target_met_date == judged) or — leniency fallback only — the Redis
// counter for it reads >= 1800s; a spared day is recorded (MarkJudged) so it
// is never re-read. Otherwise PenaliseMiss applies §8's inactivity logic in
// one conditional UPDATE, so N concurrent sweepers penalise once and the
// count returned is the number of writes that applied.
//
// First contact: a pet with no judged_through yet is judged only for days it
// existed (LocalDate(updated_at) <= judged — updated_at is the creation stamp
// until something writes the row); otherwise its marker is initialised.
//
// Errors on one pet are collected and the rest are still processed.
func (s *Service) Sweep(ctx context.Context, now time.Time) (int, error) {
	cands, err := s.repo.SweepCandidates(ctx)
	if err != nil {
		return 0, err
	}

	penalised := 0
	var errs []error
	fail := func(userID string, err error) { errs = append(errs, fmt.Errorf("user %s: %w", userID, err)) }

	for _, c := range cands {
		loc := quests.Location(c.Timezone)
		judged := PreviousDate(quests.LocalDate(now, loc))

		switch {
		case c.State.JudgedThrough != nil && *c.State.JudgedThrough >= judged:
			continue // nothing has ended since the last judgement
		case c.State.JudgedThrough == nil && quests.LocalDate(c.State.UpdatedAt, loc) > judged:
			// Never judged and not yet alive on the judged day: nothing to
			// judge, but record the day so the row stops depending on updated_at.
			if _, err := s.repo.MarkJudged(ctx, c.UserID, judged); err != nil {
				fail(c.UserID, err)
			}
			continue
		}

		met := c.State.LastTargetMetDate != nil && *c.State.LastTargetMetDate == judged
		if !met {
			total, err := s.study.Total(ctx, c.UserID, judged)
			if err != nil {
				fail(c.UserID, err)
				continue
			}
			met = total >= quests.TargetSeconds
		}
		if met {
			if _, err := s.repo.MarkJudged(ctx, c.UserID, judged); err != nil {
				fail(c.UserID, err)
			}
			continue
		}

		applied, err := s.repo.PenaliseMiss(ctx, c.UserID, judged, now)
		if err != nil {
			fail(c.UserID, err)
			continue
		}
		if applied {
			penalised++
		}
	}
	return penalised, errors.Join(errs...)
}
