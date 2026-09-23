// Package notify stores Web Push subscriptions and the learner's preferred
// practice time (backend spec §6.4 POST /settings/notifications), and runs
// the in-process reminder worker over the §4 queue:webpush:delay ZSET.
package notify

import (
	"fmt"
	"time"
)

// PollInterval is how often the worker asks the ZSET for due reminders.
const PollInterval = 30 * time.Second

// TargetSeconds mirrors quests.TargetSeconds (the §1 30-minute goal). Not
// imported: notify sees the counter only through StudyCounter.
const TargetSeconds = 1800

// Location resolves users.timezone (§3.2, default 'UTC'); an unknown name
// falls back to UTC. Same rule as quests.Location — a bad timezone must never
// stop a reminder.
func Location(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// NormalizeClock accepts "HH:MM" or "HH:MM:SS" and returns "HH:MM:SS" — the
// form Postgres renders TIME in and the form §6.4 shows.
func NormalizeClock(s string) (string, error) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("15:04:05"), nil
		}
	}
	return "", fmt.Errorf("notify: notification_time %q is not HH:MM[:SS]", s)
}

// NextSendTime is the next wall-clock hhmmss in loc strictly after now:
// today if still ahead, otherwise tomorrow. Built with time.Date in loc, so a
// DST change between now and then keeps the wall-clock time (a 23- or
// 25-hour gap) rather than adding a fixed 24 h.
func NextSendTime(now time.Time, hhmmss string, loc *time.Location) (time.Time, error) {
	norm, err := NormalizeClock(hhmmss)
	if err != nil {
		return time.Time{}, err
	}
	clock, _ := time.Parse("15:04:05", norm)
	l := now.In(loc)
	next := time.Date(l.Year(), l.Month(), l.Day(), clock.Hour(), clock.Minute(), clock.Second(), 0, loc)
	if !next.After(now) {
		next = time.Date(l.Year(), l.Month(), l.Day()+1, clock.Hour(), clock.Minute(), clock.Second(), 0, loc)
	}
	return next, nil
}

// LocalDate is the YYYY-MM-DD the user is living in — the date component of
// quests' daily:accumulated key, which StudyCounter.Total is keyed by.
func LocalDate(now time.Time, loc *time.Location) string {
	return now.In(loc).Format("2006-01-02")
}
