package quests

import (
	"fmt"
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

func TestDayNumberCountsCalendarDaysAcrossDSTTransitions(t *testing.T) {
	// DayNumber must count calendar days in the user's zone. A spring-forward
	// day is 23h long and a fall-back day 25h; dividing elapsed hours by 24
	// loses a day at every spring-forward (reviewer 2026-09-22). All 2026
	// transitions: Europe/London 03-29 / 10-25, America/New_York 03-08 / 11-01,
	// Australia/Sydney 10-04 (forward) / 04-05 (back).
	at := func(loc *time.Location, y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 9, 0, 0, 0, loc)
	}
	tests := []struct {
		zone    string
		created [3]int // y, m, d — 09:00 local
		now     [3]int // y, m, d — 09:00 local
		want    int
	}{
		// Europe/London, spring-forward 2026-03-29
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 3, 28}, 4},
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 3, 29}, 5},
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 3, 30}, 6},  // was 5
		{"Europe/London", [3]int{2026, 3, 25}, [3]int{2026, 4, 21}, 28}, // was 27: day 28 reached a day late
		// Europe/London, fall-back 2026-10-25 (25h day — unchanged, regression guard)
		{"Europe/London", [3]int{2026, 10, 20}, [3]int{2026, 10, 25}, 6},
		{"Europe/London", [3]int{2026, 10, 20}, [3]int{2026, 10, 26}, 7},
		// America/New_York, spring-forward 2026-03-08
		{"America/New_York", [3]int{2026, 3, 5}, [3]int{2026, 3, 7}, 3},
		{"America/New_York", [3]int{2026, 3, 5}, [3]int{2026, 3, 8}, 4},
		{"America/New_York", [3]int{2026, 3, 5}, [3]int{2026, 3, 9}, 5}, // was 4
		// America/New_York, fall-back 2026-11-01
		{"America/New_York", [3]int{2026, 10, 28}, [3]int{2026, 11, 1}, 5},
		{"America/New_York", [3]int{2026, 10, 28}, [3]int{2026, 11, 2}, 6},
		// Australia/Sydney, spring-forward 2026-10-04 (southern hemisphere)
		{"Australia/Sydney", [3]int{2026, 9, 29}, [3]int{2026, 10, 3}, 5},
		{"Australia/Sydney", [3]int{2026, 9, 29}, [3]int{2026, 10, 4}, 6},
		{"Australia/Sydney", [3]int{2026, 9, 29}, [3]int{2026, 10, 5}, 7}, // was 6
		// Australia/Sydney, fall-back 2026-04-05
		{"Australia/Sydney", [3]int{2026, 4, 1}, [3]int{2026, 4, 5}, 5},
		{"Australia/Sydney", [3]int{2026, 4, 1}, [3]int{2026, 4, 6}, 6},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s %04d-%02d-%02d -> %04d-%02d-%02d", tt.zone,
			tt.created[0], tt.created[1], tt.created[2], tt.now[0], tt.now[1], tt.now[2])
		t.Run(name, func(t *testing.T) {
			loc := Location(tt.zone)
			if loc == time.UTC {
				t.Fatalf("Location(%q) fell back to UTC — tzdata missing on this machine", tt.zone)
			}
			created := at(loc, tt.created[0], time.Month(tt.created[1]), tt.created[2])
			now := at(loc, tt.now[0], time.Month(tt.now[1]), tt.now[2])
			if got := DayNumber(created, now, loc); got != tt.want {
				t.Errorf("DayNumber = %d, want %d", got, tt.want)
			}
		})
	}

	// Inside the skipped hour itself: 01:30Z on 2026-03-29 is 02:30 BST, still day 5.
	loc := Location("Europe/London")
	created := at(loc, 2026, time.March, 25)
	if got := DayNumber(created, time.Date(2026, time.March, 29, 1, 30, 0, 0, time.UTC), loc); got != 5 {
		t.Errorf("DayNumber inside the transition hour = %d, want 5", got)
	}
}

func TestRoadmapLength(t *testing.T) {
	if RoadmapDays != 28 {
		t.Errorf("RoadmapDays = %d, want 28 (spec §6.1: 4 modules x 7 days)", RoadmapDays)
	}
}

func TestDayDateIsTheInverseOfDayNumberAcrossDST(t *testing.T) {
	at := func(loc *time.Location, y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 9, 0, 0, 0, loc)
	}
	tests := []struct {
		zone    string
		created [3]int // y, m, d — 09:00 local
	}{
		{"Europe/London", [3]int{2026, 3, 25}},
		{"Europe/London", [3]int{2026, 10, 20}},
		{"America/New_York", [3]int{2026, 3, 5}},
		{"America/New_York", [3]int{2026, 10, 28}},
		{"Australia/Sydney", [3]int{2026, 9, 29}},
		{"Australia/Sydney", [3]int{2026, 4, 1}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %04d-%02d-%02d", tt.zone, tt.created[0], tt.created[1], tt.created[2]), func(t *testing.T) {
			loc := Location(tt.zone)
			if loc == time.UTC {
				t.Fatalf("Location(%q) fell back to UTC — tzdata missing on this machine", tt.zone)
			}
			created := at(loc, tt.created[0], time.Month(tt.created[1]), tt.created[2])
			for n := 1; n <= RoadmapDays; n++ {
				d := DayDate(created, n, loc)
				parsed, err := time.ParseInLocation("2006-01-02", d, loc)
				if err != nil {
					t.Fatalf("DayDate(%d) = %q: %v", n, d, err)
				}
				parsed = parsed.Add(9 * time.Hour)
				if got := DayNumber(created, parsed, loc); got != n {
					t.Errorf("DayNumber(created, DayDate(created, %d)) = %d, want %d (date %q)", n, got, n, d)
				}
			}
		})
	}

	loc := Location("Europe/London")
	created := at(loc, 2026, time.March, 25)
	if got := DayDate(created, 5, loc); got != "2026-03-29" {
		t.Errorf("DayDate(london 2026-03-25, 5) = %q, want 2026-03-29 (the spring-forward day itself)", got)
	}
	if got := DayDate(created, 28, loc); got != "2026-04-21" {
		t.Errorf("DayDate(london 2026-03-25, 28) = %q, want 2026-04-21", got)
	}
}

func TestDayDateStartsOnTheLocalCreationDate(t *testing.T) {
	// 18:30 UTC on 2026-09-01 is already 01:30 on 2026-09-02 in Ho Chi Minh City.
	created := time.Date(2026, time.September, 1, 18, 30, 0, 0, time.UTC)

	if got := DayDate(created, 1, Location("UTC")); got != "2026-09-01" {
		t.Errorf("DayDate(UTC, 1) = %q, want 2026-09-01", got)
	}
	if got := DayDate(created, 1, Location("Asia/Ho_Chi_Minh")); got != "2026-09-02" {
		t.Errorf("DayDate(Asia/Ho_Chi_Minh, 1) = %q, want 2026-09-02 — a roadmap created late in the day still counts that day as day 1, in the learner's zone", got)
	}
	if got := DayDate(created, 2, Location("Asia/Ho_Chi_Minh")); got != "2026-09-03" {
		t.Errorf("DayDate(Asia/Ho_Chi_Minh, 2) = %q, want 2026-09-03", got)
	}
}
