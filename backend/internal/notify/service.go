package notify

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrInvalidRequest means the settings body failed validation; nothing was written.
var ErrInvalidRequest = errors.New("notify: invalid request")

// StudyCounter is notify's read-only view of the §4 daily:accumulated
// counter, keyed by the user's LOCAL date. *quests.RedisCounter satisfies it
// (cmd/api/main.go registers it), so notify never touches quests' key.
type StudyCounter interface {
	Total(ctx context.Context, userID, localDate string) (int64, error)
}

// SettingsRequest is the validated form of the §6.4 body. Subscription is
// nil when the client only moved the time.
type SettingsRequest struct {
	NotificationTime string
	Timezone         string // optional IANA name; "" keeps users.timezone
	Subscription     *Subscription
}

// SettingsResult is the §6.4 response plus next_reminder_at (additive).
type SettingsResult struct {
	Status           string `json:"status"`
	NotificationTime string `json:"notification_time"`
	NextReminderAt   string `json:"next_reminder_at"`
}

// TickStats is one worker pass, for logs and tests.
type TickStats struct {
	Due, Sent, Skipped, Pruned, Failed int
}

// Service owns the settings write path and the reminder pass.
type Service struct {
	repo    Repo
	queue   Queue
	sender  Sender
	counter StudyCounter
	now     func() time.Time
	payload Payload
}

// NewService wires the dependencies. sender may be nil when VAPID keys are
// absent: UpdateSettings still works; Tick must not be called (main.go does
// not start the worker).
func NewService(repo Repo, queue Queue, sender Sender, counter StudyCounter, now func() time.Time) *Service {
	return &Service{repo: repo, queue: queue, sender: sender, counter: counter, now: now, payload: DefaultPayload}
}

// UpdateSettings validates, writes users.notification_time (+timezone),
// stores the subscription, and ZADDs the next local send time.
func (s *Service) UpdateSettings(ctx context.Context, userID string, req SettingsRequest) (SettingsResult, error) {
	clock, err := NormalizeClock(req.NotificationTime)
	if err != nil {
		return SettingsResult{}, fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			return SettingsResult{}, fmt.Errorf("%w: unknown timezone %q", ErrInvalidRequest, req.Timezone)
		}
	}
	if sub := req.Subscription; sub != nil && (sub.Endpoint == "" || sub.P256dh == "" || sub.Auth == "") {
		return SettingsResult{}, fmt.Errorf("%w: push_subscription needs endpoint, p256dh and auth", ErrInvalidRequest)
	}
	if req.Subscription != nil {
		if err := ValidateEndpoint(req.Subscription.Endpoint); err != nil {
			return SettingsResult{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
		}
	}

	if err := s.repo.UpdatePreferences(ctx, userID, clock, req.Timezone); err != nil {
		return SettingsResult{}, err
	}
	if req.Subscription != nil {
		if err := s.repo.SaveSubscription(ctx, userID, *req.Subscription); err != nil {
			return SettingsResult{}, err
		}
	}
	prefs, err := s.repo.Preferences(ctx, userID)
	if err != nil {
		return SettingsResult{}, err
	}
	next, err := NextSendTime(s.now(), clock, Location(prefs.Timezone))
	if err != nil {
		return SettingsResult{}, err
	}
	if err := s.queue.Schedule(ctx, userID, next); err != nil {
		return SettingsResult{}, err
	}
	return SettingsResult{Status: "updated", NotificationTime: clock, NextReminderAt: next.UTC().Format(time.RFC3339)}, nil
}

// Tick is one worker pass at now: pop due users, re-slot each for tomorrow
// FIRST (a crash mid-send then costs one reminder, not one every 30 s), skip
// those who already met today's target, send to every subscription, prune
// 404/410 ones, and drop users with nothing to send to. Per-user failures
// are collected and returned joined; the pass never stops early.
func (s *Service) Tick(ctx context.Context, now time.Time) (TickStats, error) {
	var stats TickStats
	due, err := s.queue.Due(ctx, now, DueBatchSize)
	if err != nil {
		return stats, err
	}
	stats.Due = len(due)

	var errs []error
	for _, userID := range due {
		prefs, err := s.repo.Preferences(ctx, userID)
		if errors.Is(err, ErrUserNotFound) {
			errs = appendIf(errs, s.queue.Remove(ctx, userID))
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		loc := Location(prefs.Timezone)

		next, err := NextSendTime(now, prefs.NotificationTime, loc)
		if err != nil {
			// Unparseable column value: keep the user out of a hot loop.
			next = now.Add(24 * time.Hour)
			errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
		}
		if err := s.queue.Schedule(ctx, userID, next); err != nil {
			errs = append(errs, err)
			continue
		}

		total, err := s.counter.Total(ctx, userID, LocalDate(now, loc))
		if err != nil {
			// Unknown is not "met": send rather than silently skip.
			errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
			total = 0
		}
		if total >= TargetSeconds {
			stats.Skipped++
			continue
		}

		subs, err := s.repo.Subscriptions(ctx, userID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(subs) == 0 {
			errs = appendIf(errs, s.queue.Remove(ctx, userID))
			continue
		}
		for _, sub := range subs {
			err := s.sender.Send(ctx, sub, s.payload)
			switch {
			case errors.Is(err, ErrSubscriptionGone):
				stats.Pruned++
				errs = appendIf(errs, s.repo.DeleteSubscription(ctx, sub.ID))
			case err != nil:
				stats.Failed++
				errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
			default:
				stats.Sent++
			}
		}
	}
	return stats, errors.Join(errs...)
}

func appendIf(errs []error, err error) []error {
	if err != nil {
		return append(errs, err)
	}
	return errs
}
