// Package google is the one-way Calendar + Tasks push of 1st-thinking §5.1
// steps 6-7 (wire contract: backend spec §6.4). Nothing is read back from
// Google and nothing subscribes to it.
package google

import (
	"fmt"
	"strings"
	"time"
)

// EventSummary and EventDescription name the recurring Calendar block.
const (
	EventSummary     = "English practice"
	EventDescription = "Your daily 30-minute English session. Open the app to start today's quests."
	// EventDuration is the §1 daily target.
	EventDuration = 30 * time.Minute
	// Recurrence is one event per day for the roadmap's 28 days (§6.1: 4 x 7).
	Recurrence = "RRULE:FREQ=DAILY;COUNT=28"
	// TasklistTitle names the Google Tasks list holding one task per roadmap day.
	TasklistTitle = "English daily quests"
	// RoadmapDays mirrors quests.RoadmapDays (not imported: package boundary).
	RoadmapDays = 28
)

// Location resolves users.timezone (§3.2, default 'UTC'); an unknown name
// falls back to UTC. Same rule as quests.Location — a bad timezone must never
// make the sync fail.
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

// parseClock accepts users.notification_time as Postgres renders TIME
// ("20:00:00") or as "HH:MM".
func parseClock(s string) (h, m, sec int, err error) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, perr := time.Parse(layout, s); perr == nil {
			return t.Hour(), t.Minute(), t.Second(), nil
		}
	}
	return 0, 0, 0, fmt.Errorf("google: notification_time %q is not HH:MM[:SS]", s)
}

// NextOccurrence is the next wall-clock hhmmss in loc strictly after now:
// today if that moment is still ahead, otherwise tomorrow. It goes through
// time.Date in loc, so a DST transition between now and then keeps the
// wall-clock time rather than shifting it by an hour.
func NextOccurrence(now time.Time, hhmmss string, loc *time.Location) (time.Time, error) {
	h, m, s, err := parseClock(hhmmss)
	if err != nil {
		return time.Time{}, err
	}
	l := now.In(loc)
	candidate := time.Date(l.Year(), l.Month(), l.Day(), h, m, s, 0, loc)
	if !candidate.After(now) {
		candidate = time.Date(l.Year(), l.Month(), l.Day()+1, h, m, s, 0, loc)
	}
	return candidate, nil
}

// Event is the Calendar event this package pushes (calendar.go sends it).
type Event struct {
	Summary     string
	Description string
	Start, End  time.Time
	TimeZone    string // IANA name, sent alongside dateTime so Google recurs in the user's zone
	ID          string // client-supplied Calendar id; empty lets Google assign one
}

// payload is the Calendar v3 events resource body for insert and patch.
func (e Event) payload() map[string]any {
	p := map[string]any{
		"summary":     e.Summary,
		"description": e.Description,
		"start":       map[string]string{"dateTime": e.Start.Format(time.RFC3339), "timeZone": e.TimeZone},
		"end":         map[string]string{"dateTime": e.End.Format(time.RFC3339), "timeZone": e.TimeZone},
		"recurrence":  []string{Recurrence},
		// A PATCH with status confirmed restores an event the user deleted
		// (Google keeps it as "cancelled" and reserves its id).
		"status": "confirmed",
	}
	if e.ID != "" {
		p["id"] = e.ID
	}
	return p
}

// PracticeEventID is the client-supplied Calendar id of a user's recurring
// practice block: "aelp" + the user id lower-cased with every character
// outside base32hex ([a-v0-9]) dropped — a UUID's 32 hex digits are a subset.
// Calendar v3 events.insert accepts 5–1024 such characters. Deterministic per
// user, so a repeat insert (after a SaveSyncState failure or a client
// disconnect) is a 409 from Google rather than a second event: that is what
// makes the insert idempotent.
func PracticeEventID(userID string) string {
	var b strings.Builder
	b.WriteString("aelp")
	for _, r := range strings.ToLower(userID) {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'v') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// PracticeEvent builds the 30-minute daily block starting at the next
// notificationTime in timezone.
func PracticeEvent(now time.Time, notificationTime, timezone string) (Event, error) {
	loc := Location(timezone)
	start, err := NextOccurrence(now, notificationTime, loc)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Summary:     EventSummary,
		Description: EventDescription,
		Start:       start,
		End:         start.Add(EventDuration),
		TimeZone:    loc.String(),
	}, nil
}

// DayDue is the calendar date of roadmap day n (1-based) — the user's local
// date of roadmaps.created_at plus n-1 days — expressed as UTC midnight,
// which is how the Tasks API stores `due` (it keeps only the date part).
// Same calendar-day arithmetic as quests.DayNumber, inverted.
func DayDue(createdAt time.Time, n int, loc *time.Location) time.Time {
	l := createdAt.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day()+n-1, 0, 0, 0, 0, time.UTC)
}

// TaskTitle is "Day N: title · title · title", skipping empty titles.
func TaskTitle(day int, titles []string) string {
	var kept []string
	for _, t := range titles {
		if t = strings.TrimSpace(t); t != "" {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return fmt.Sprintf("Day %d", day)
	}
	return fmt.Sprintf("Day %d: %s", day, strings.Join(kept, " · "))
}
