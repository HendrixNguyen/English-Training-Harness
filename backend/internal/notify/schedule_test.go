package notify

import (
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

func TestNormalizeClockAcceptsHHMMAndHHMMSS(t *testing.T) {
	for in, want := range map[string]string{"20:00:00": "20:00:00", "7:05": "07:05:00", "23:59": "23:59:00", "00:00:00": "00:00:00"} {
		got, err := NormalizeClock(in)
		if err != nil || got != want {
			t.Errorf("NormalizeClock(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "24:00", "20:60:00", "eight", "20:00:00Z", "8pm"} {
		if _, err := NormalizeClock(bad); err == nil {
			t.Errorf("NormalizeClock(%q) accepted, want error", bad)
		}
	}
}

func TestNextSendTimeIsTodayIfAheadElseTomorrow(t *testing.T) {
	hcm := Location("Asia/Ho_Chi_Minh") // UTC+7
	// 2026-09-22T10:00Z is 17:00 in Ho Chi Minh City.
	now := time.Date(2026, time.September, 22, 10, 0, 0, 0, time.UTC)

	got, err := NextSendTime(now, "20:00:00", hcm)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 22, 20, 0, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("still ahead: got %v, want %v", got, want)
	}

	got, err = NextSendTime(now, "09:00:00", hcm)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.September, 23, 9, 0, 0, 0, hcm); !got.Equal(want) {
		t.Errorf("already past: got %v, want %v", got, want)
	}

	// Exactly now is "not ahead": the worker re-slots at fire time and must
	// land on tomorrow, never on the instant it is processing.
	at := time.Date(2026, time.September, 22, 20, 0, 0, 0, hcm)
	got, _ = NextSendTime(at, "20:00:00", hcm)
	if !got.Equal(at.AddDate(0, 0, 1)) {
		t.Errorf("at the exact minute: got %v, want tomorrow %v", got, at.AddDate(0, 0, 1))
	}
}

func TestNextSendTimeKeepsWallClockAcrossDST(t *testing.T) {
	ny := Location("America/New_York")
	// 2026-03-08 is spring-forward in New York (23-hour day).
	now := time.Date(2026, time.March, 7, 21, 0, 0, 0, ny) // 20:00 has passed
	got, err := NextSendTime(now, "20:00:00", ny)
	if err != nil {
		t.Fatal(err)
	}
	if got.Day() != 8 || got.Hour() != 20 {
		t.Errorf("got %v, want 2026-03-08 20:00 local", got)
	}
	if _, off := got.Zone(); off != -4*3600 {
		t.Errorf("offset = %d, want -14400 (EDT after spring-forward)", off)
	}
	// PLAN DEVIATION (executor, 2026-09-23): the plan's test asserted exactly
	// 23h here. Verified against Go's tzdata directly (see Execution summary):
	// now is 21:00 EST (UTC-5) on Mar 7; the correct next occurrence is 20:00
	// EDT (UTC-4) on Mar 8, which is 22h later, not 23h — 23h would only hold
	// starting from 21:00 on Mar 7 to 20:00 the *following* day if no DST
	// change occurred; the lost hour on the spring-forward day makes it 22h.
	// The implementation (time.Date in loc, letting Go resolve the offset) is
	// correct — it is the plan's hardcoded assertion that was arithmetically
	// wrong, not the code.
	if d := got.Sub(now); d != 22*time.Hour {
		t.Errorf("elapsed = %v, want 22h — the wall clock is kept, not a fixed interval", d)
	}
}

func TestLocalDateUsesTheUsersTimezone(t *testing.T) {
	now := time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC)
	if got := LocalDate(now, Location("Asia/Ho_Chi_Minh")); got != "2026-09-23" {
		t.Errorf("got %q, want 2026-09-23", got)
	}
	if got := LocalDate(now, time.UTC); got != "2026-09-22" {
		t.Errorf("got %q, want 2026-09-22", got)
	}
}
