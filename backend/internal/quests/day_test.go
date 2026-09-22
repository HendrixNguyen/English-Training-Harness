package quests

import (
	"testing"
	"time"
)

func TestLocationFallsBackToUTC(t *testing.T) {
	for _, name := range []string{"", "Not/AZone", "Mars/Olympus"} {
		if got := Location(name); got != time.UTC {
			t.Errorf("Location(%q) = %v, want UTC", name, got)
		}
	}
	if got := Location("Asia/Ho_Chi_Minh"); got.String() != "Asia/Ho_Chi_Minh" {
		t.Errorf("Location(Asia/Ho_Chi_Minh) = %v", got)
	}
}

func TestLocalDateUsesTheUsersTimezone(t *testing.T) {
	// 2026-09-22T18:30Z is already 2026-09-23 in Ho Chi Minh City (UTC+7).
	now := time.Date(2026, time.September, 22, 18, 30, 0, 0, time.UTC)

	if got := LocalDate(now, Location("UTC")); got != "2026-09-22" {
		t.Errorf("LocalDate(UTC) = %q, want 2026-09-22", got)
	}
	if got := LocalDate(now, Location("Asia/Ho_Chi_Minh")); got != "2026-09-23" {
		t.Errorf("LocalDate(Asia/Ho_Chi_Minh) = %q, want 2026-09-23", got)
	}
}

func TestDayNumberCountsCalendarDaysFromCreation(t *testing.T) {
	loc := Location("UTC")
	created := time.Date(2026, time.September, 1, 22, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want int
	}{
		{"same day", time.Date(2026, time.September, 1, 23, 59, 0, 0, time.UTC), 1},
		{"next calendar day, 2h later", time.Date(2026, time.September, 2, 0, 30, 0, 0, time.UTC), 2},
		{"a week in", time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC), 8},
		{"last day", time.Date(2026, time.September, 28, 9, 0, 0, 0, time.UTC), 28},
		{"clamped past the end", time.Date(2026, time.November, 1, 9, 0, 0, 0, time.UTC), 28},
		{"clamped before creation (clock skew)", time.Date(2026, time.August, 30, 9, 0, 0, 0, time.UTC), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DayNumber(created, tt.now, loc); got != tt.want {
				t.Errorf("DayNumber = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDayNumberCrossesTheBoundaryInTheUsersTimezone(t *testing.T) {
	loc := Location("Asia/Ho_Chi_Minh")
	// Created 2026-09-01T20:00 local (= 13:00Z).
	created := time.Date(2026, time.September, 1, 13, 0, 0, 0, time.UTC)

	// 2026-09-01T23:30 local — still day 1.
	if got := DayNumber(created, time.Date(2026, time.September, 1, 16, 30, 0, 0, time.UTC), loc); got != 1 {
		t.Errorf("before local midnight: DayNumber = %d, want 1", got)
	}
	// 2026-09-02T00:30 local — day 2, even though it is still 2026-09-01 in UTC.
	if got := DayNumber(created, time.Date(2026, time.September, 1, 17, 30, 0, 0, time.UTC), loc); got != 2 {
		t.Errorf("after local midnight: DayNumber = %d, want 2", got)
	}
}

func TestRoadmapLength(t *testing.T) {
	if RoadmapDays != 28 {
		t.Errorf("RoadmapDays = %d, want 28 (spec §6.1: 4 modules x 7 days)", RoadmapDays)
	}
}
