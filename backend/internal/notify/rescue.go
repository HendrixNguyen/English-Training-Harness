package notify

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// RescueLocalHour is the local hour (T-2h before midnight) in which the
// pre-decay rescue push may fire — once per user per local day.
const RescueLocalHour = 22

// RescueCandidate is a user with at least one push subscription and the
// timezone that decides whether their local hour is RescueLocalHour.
type RescueCandidate struct{ UserID, Timezone string }

// RescueRepo is the Postgres side of the rescue job. *PgRepo satisfies it;
// Subscriptions/DeleteSubscription are the ones Repo already has.
type RescueRepo interface {
	RescueCandidates(ctx context.Context) ([]RescueCandidate, error)
	Subscriptions(ctx context.Context, userID string) ([]Subscription, error)
	DeleteSubscription(ctx context.Context, id string) error
}

const rescueCandidatesSQL = `
SELECT DISTINCT u.id::text, COALESCE(u.timezone, 'UTC')
FROM users u JOIN push_subscriptions p ON p.user_id = u.id
ORDER BY u.id::text`

// RescueCandidates lists every user with at least one push subscription.
func (r *PgRepo) RescueCandidates(ctx context.Context) ([]RescueCandidate, error) {
	rows, err := r.Pool.Query(ctx, rescueCandidatesSQL)
	if err != nil {
		return nil, fmt.Errorf("notify: reading rescue candidates: %w", err)
	}
	defer rows.Close()

	var out []RescueCandidate
	for rows.Next() {
		var c RescueCandidate
		if err := rows.Scan(&c.UserID, &c.Timezone); err != nil {
			return nil, fmt.Errorf("notify: scanning rescue candidate: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var _ RescueRepo = (*PgRepo)(nil)

// RescueFlags is the once-per-local-day guard: Claim is SET NX EX on
// store.RescueKey and answers true only for the first caller that day.
type RescueFlags interface {
	Claim(ctx context.Context, userID, localDate string) (bool, error)
}

// RedisRescueFlags is the real RescueFlags.
type RedisRescueFlags struct{ Client *redis.Client }

// NewRedisRescueFlags builds the flag store over an existing client.
func NewRedisRescueFlags(r *store.Redis) *RedisRescueFlags {
	return &RedisRescueFlags{Client: r.Client}
}

func (f *RedisRescueFlags) Claim(ctx context.Context, userID, localDate string) (bool, error) {
	ok, err := f.Client.SetNX(ctx, store.RescueKey(userID, localDate), "1", store.RescueTTL).Result()
	if err != nil {
		return false, fmt.Errorf("notify: claiming rescue flag: %w", err)
	}
	return ok, nil
}

// RescueMinutes is how many whole minutes are still missing from the
// TargetSeconds goal, never below 1 (the push is only sent when total < TargetSeconds).
func RescueMinutes(total int64) int {
	missing := int64(TargetSeconds) - total
	if missing <= 0 {
		return 1
	}
	m := int((missing + 59) / 60)
	if m < 1 {
		m = 1
	}
	return m
}

// RescuePayload is the rescue copy; the plant speaks as "tớ".
func RescuePayload(minutes int) Payload {
	return Payload{
		Title: fmt.Sprintf("Tớ cần cậu thêm %d phút nữa", minutes),
		Body:  fmt.Sprintf("Cây của cậu cần thêm %d phút hôm nay", minutes),
		URL:   "/",
	}
}

// RescueStats is one sweep, for logs and tests.
type RescueStats struct {
	Candidates, InWindow, Met, AlreadySent, Skipped, Sent, Pruned, Failed int
}

// Rescue is the pre-decay rescue job: one Web Push at local RescueLocalHour
// on a day whose daily:accumulated total is below TargetSeconds.
type Rescue struct {
	repo    RescueRepo
	counter StudyCounter
	flags   RescueFlags
	sender  Sender
	now     func() time.Time
}

// NewRescue wires the dependencies; sender must be non-nil (main.go builds
// it only when VAPID keys are set, like RunWorker).
func NewRescue(repo RescueRepo, counter StudyCounter, flags RescueFlags, sender Sender, now func() time.Time) *Rescue {
	return &Rescue{repo: repo, counter: counter, flags: flags, sender: sender, now: now}
}

// Sweep is one hourly pass at now. Per candidate: skip unless the local
// hour is RescueLocalHour; read the local day's seconds (an error SKIPS —
// a rescue must never fire twice, so unknown is not "unmet" here, unlike
// Tick); skip a met day without touching the flag; claim the flag (false →
// already sent today; error → skip); send RescuePayload to every
// subscription with Tick's prune rules. Per-user errors are joined; the
// pass never stops early.
func (r *Rescue) Sweep(ctx context.Context, now time.Time) (RescueStats, error) {
	var stats RescueStats
	candidates, err := r.repo.RescueCandidates(ctx)
	if err != nil {
		return stats, err
	}
	stats.Candidates = len(candidates)

	var errs []error
	for _, c := range candidates {
		loc := Location(c.Timezone)
		if now.In(loc).Hour() != RescueLocalHour {
			continue
		}
		stats.InWindow++
		date := LocalDate(now, loc)

		total, err := r.counter.Total(ctx, c.UserID, date)
		if err != nil {
			stats.Skipped++
			errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			continue
		}
		if total >= TargetSeconds {
			stats.Met++
			continue
		}

		claimed, err := r.flags.Claim(ctx, c.UserID, date)
		if err != nil {
			stats.Skipped++
			errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			continue
		}
		if !claimed {
			stats.AlreadySent++
			continue
		}

		subs, err := r.repo.Subscriptions(ctx, c.UserID)
		if err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			continue
		}
		// len(subs) == 0: unsubscribed since the candidates query — silent.
		payload := RescuePayload(RescueMinutes(total))
		for _, sub := range subs {
			err := r.sender.Send(ctx, sub, payload)
			switch {
			case errors.Is(err, ErrSubscriptionGone):
				stats.Pruned++
				errs = appendIf(errs, r.repo.DeleteSubscription(ctx, sub.ID))
			case errors.Is(err, ErrForbiddenEndpoint):
				stats.Pruned++
				errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
				errs = appendIf(errs, r.repo.DeleteSubscription(ctx, sub.ID))
			case err != nil:
				stats.Failed++
				errs = append(errs, fmt.Errorf("user %s: %w", c.UserID, err))
			default:
				stats.Sent++
			}
		}
	}
	return stats, errors.Join(errs...)
}
