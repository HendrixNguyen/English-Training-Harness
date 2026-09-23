package google

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestLocationFallsBackToUTC(t *testing.T) {
	for _, name := range []string{"", "Not/AZone"} {
		if got := Location(name); got != time.UTC {
			t.Errorf("Location(%q) = %v, want UTC", name, got)
		}
	}
	if got := Location("Asia/Ho_Chi_Minh"); got.String() != "Asia/Ho_Chi_Minh" {
		t.Errorf("Location(Asia/Ho_Chi_Minh) = %v", got)
	}
}

func TestNextOccurrenceIsTodayWhenStillAheadElseTomorrow(t *testing.T) {
	hcm := Location("Asia/Ho_Chi_Minh") // UTC+7, no DST
	// 2026-09-22T10:00Z = 17:00 in Ho Chi Minh City.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	got, err := NextOccurrence(now, "20:00:00", hcm)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 22, 20, 0, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("20:00 still ahead: got %v, want %v", got, want)
	}

	got, err = NextOccurrence(now, "09:30", hcm) // HH:MM is accepted too
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 23, 9, 30, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("09:30 already past: got %v, want %v", got, want)
	}

	if _, err := NextOccurrence(now, "25:00:00", hcm); err == nil {
		t.Error("expected an error for hour 25")
	}
	if _, err := NextOccurrence(now, "eight", hcm); err == nil {
		t.Error("expected an error for a non-clock string")
	}
}

func TestNextOccurrenceKeepsWallClockAcrossDST(t *testing.T) {
	ny := Location("America/New_York")
	// 2026-11-01 is the fall-back day in New York (25 hours). 20:00 local on
	// 2026-10-31 has passed; the next 20:00 must be 20:00 on 11-01, EST.
	now := time.Date(2026, time.October, 31, 21, 0, 0, 0, ny)
	got, err := NextOccurrence(now, "20:00:00", ny)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hour() != 20 || got.Day() != 1 || got.Month() != time.November {
		t.Errorf("got %v, want 2026-11-01 20:00 America/New_York", got)
	}
	if _, off := got.Zone(); off != -5*3600 {
		t.Errorf("offset = %d, want -18000 (EST after fall-back)", off)
	}
}

func TestPracticeEventIsThirtyMinutesDailyFor28Days(t *testing.T) {
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)
	ev, err := PracticeEvent(now, "20:00:00", "Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatal(err)
	}
	if ev.End.Sub(ev.Start) != 30*time.Minute {
		t.Errorf("duration = %v, want 30m", ev.End.Sub(ev.Start))
	}
	if ev.Summary != EventSummary || ev.TimeZone != "Asia/Ho_Chi_Minh" {
		t.Errorf("summary/timezone = %q/%q", ev.Summary, ev.TimeZone)
	}

	body, _ := json.Marshal(ev.payload())
	s := string(body)
	for _, want := range []string{
		`"recurrence":["RRULE:FREQ=DAILY;COUNT=28"]`,
		`"dateTime":"2026-09-22T20:00:00+07:00"`,
		`"dateTime":"2026-09-22T20:30:00+07:00"`,
		`"timeZone":"Asia/Ho_Chi_Minh"`,
		`"summary":"English practice"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("payload %s is missing %s", s, want)
		}
	}
}

func TestDayDueIsTheRoadmapDayAsADate(t *testing.T) {
	hcm := Location("Asia/Ho_Chi_Minh")
	// Created 2026-09-01T18:00Z = 2026-09-02 01:00 in HCM: day 1 is the 2nd.
	created := time.Date(2026, time.September, 1, 18, 0, 0, 0, time.UTC)
	if got := DayDue(created, 1, hcm); got.Format("2006-01-02") != "2026-09-02" {
		t.Errorf("day 1 due %v, want 2026-09-02", got)
	}
	if got := DayDue(created, 28, hcm); got.Format("2006-01-02") != "2026-09-29" {
		t.Errorf("day 28 due %v, want 2026-09-29", got)
	}
	if got := DayDue(created, 1, hcm); got.Location() != time.UTC || got.Hour() != 0 {
		t.Errorf("due must be UTC midnight (Tasks API keeps only the date), got %v", got)
	}
}

func TestPracticeEventIDIsDeterministicBase32Hex(t *testing.T) {
	const uuid = "A0EEBC99-9C0B-4EF8-BB6D-6BB9BD380A11"
	id := PracticeEventID(uuid)
	if id != PracticeEventID(uuid) {
		t.Fatal("not deterministic")
	}
	// Calendar v3 events.insert: id is 5–1024 chars of base32hex, i.e. [a-v0-9].
	// (Length checked separately: Go's RE2 rejects a {5,1024} repeat count.)
	if !regexp.MustCompile(`^[a-v0-9]+$`).MatchString(id) {
		t.Fatalf("id %q is not base32hex", id)
	}
	if len(id) < 5 || len(id) > 1024 {
		t.Fatalf("id length = %d, want 5-1024", len(id))
	}
	if id != "aelpa0eebc999c0b4ef8bb6d6bb9bd380a11" {
		t.Fatalf("id = %q", id)
	}
	if PracticeEventID("u1") == PracticeEventID("u2") {
		t.Fatal("two users share an id")
	}
}

func TestEventPayloadCarriesIDAndConfirmedStatus(t *testing.T) {
	ev := sampleEvent()
	if _, has := ev.payload()["id"]; has {
		t.Fatal("payload sends an id when none is set")
	}
	ev.ID = "aelpu1"
	p := ev.payload()
	if p["id"] != "aelpu1" || p["status"] != "confirmed" {
		t.Fatalf("payload = %v", p)
	}
}

func TestTaskTitleJoinsTheDaysExercises(t *testing.T) {
	if got := TaskTitle(3, []string{"Greetings", "Short story", "Order a coffee"}); got != "Day 3: Greetings · Short story · Order a coffee" {
		t.Errorf("got %q", got)
	}
	if got := TaskTitle(7, []string{"", "Only one"}); got != "Day 7: Only one" {
		t.Errorf("empty titles are skipped: got %q", got)
	}
	if got := TaskTitle(9, nil); got != "Day 9" {
		t.Errorf("no titles: got %q", got)
	}
}
