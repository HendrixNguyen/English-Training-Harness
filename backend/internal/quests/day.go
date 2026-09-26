// Package quests serves the daily 30-minute exercise suite and records progress
// against it (1st-thinking doc §5.2, §6.1; backend spec §6.2 for the wire DTOs).
package quests

import "time"

// RoadmapDays is the fixed roadmap length: 4 modules of 7 daily quests (§6.1).
const RoadmapDays = 28

// TargetSeconds is the daily goal from §1: at least 30 minutes.
const TargetSeconds = 1800

// MaxDurationSeconds bounds one POST /quests/progress report. A §6.2 task is
// 10 minutes and the whole day is 30; one hour is a learner who left a task
// open, not a plausible single sitting. Anything larger is a client bug
// (milliseconds, an overflowed Number) or abuse, and is answered 400 before
// the INCRBY. Chosen so that even the per-day ceiling below cannot be crossed
// by less than 24 max-size reports.
const MaxDurationSeconds = 3600

// MaxDailySeconds bounds the counter for one local day: nobody studies more
// than a day in a day. RecordProgress rejects, before the INCRBY, any report
// that would push the running total past it. daily_progress.minutes_spent is
// INT (§3.2); without a ceiling ~1.29e11 accumulated seconds overflow the
// upsert and every later call for that user fails for the 48h TTL (reviewer
// 2026-09-22). The check races with concurrent calls, but each can overshoot
// by at most MaxDurationSeconds, so the INT limit stays ~10^6 concurrent
// max-size requests away.
const MaxDailySeconds = 86400

// Location resolves users.timezone (§3.2, default 'UTC'). An unknown name falls
// back to UTC rather than erroring — a bad timezone string must never lock a
// learner out of their quests.
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

// LocalDate is the YYYY-MM-DD the user is currently living in. It is the date
// component of the daily:accumulated key and of daily_progress.date.
func LocalDate(now time.Time, loc *time.Location) string {
	return now.In(loc).Format("2006-01-02")
}

// DayNumber is the 1-based day of the roadmap, counted in calendar days in the
// user's own timezone and clamped to 1..RoadmapDays. Clamping at the top means a
// learner past day 28 keeps seeing day 28 — see the plan's open questions.
//
// It counts calendar days, not elapsed hours: the two local midnights are
// re-expressed as UTC dates before subtracting, so a 23-hour spring-forward day
// or a 25-hour fall-back day is still exactly one day. Dividing time.Sub by 24h
// lost a day at every spring-forward (reviewer 2026-09-22).
func DayNumber(createdAt, now time.Time, loc *time.Location) int {
	start := startOfDay(createdAt, loc)
	today := startOfDay(now, loc)

	su := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	tu := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	days := int(tu.Sub(su)/(24*time.Hour)) + 1
	if days < 1 {
		return 1
	}
	if days > RoadmapDays {
		return RoadmapDays
	}
	return days
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

// DayDate is the inverse of DayNumber: the YYYY-MM-DD of roadmap day n
// (1-based) in the user's timezone — the local date of created_at plus n-1
// calendar days. It is the key GET /roadmap joins daily_progress on, and the
// same rule google.DayDue applies to Google Tasks `due`. Calendar days again,
// not 24-hour multiples: AddDate re-resolves the wall-clock date, so a
// spring-forward or fall-back day is still one day.
func DayDate(createdAt time.Time, n int, loc *time.Location) string {
	return startOfDay(createdAt, loc).AddDate(0, 0, n-1).Format("2006-01-02")
}
