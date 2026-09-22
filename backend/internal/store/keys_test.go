package store

import (
	"testing"
	"time"
)

func TestKeyBuilders(t *testing.T) {
	const uid = "3f0d1a7e-0000-4000-8000-000000000001"
	day := time.Date(2026, time.September, 22, 23, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"session", SessionKey(uid), "sess:3f0d1a7e-0000-4000-8000-000000000001:token"},
		{"placement", PlacementQuizKey(uid), "quiz:placement:3f0d1a7e-0000-4000-8000-000000000001"},
		{"daily", DailyAccumulatedKey(uid, day), "daily:accumulated:3f0d1a7e-0000-4000-8000-000000000001:2026-09-22"},
		{"ratelimit", AIRateLimitKey(uid), "ratelimit:ai:3f0d1a7e-0000-4000-8000-000000000001"},
		{"webpush", WebPushDelayQueueKey, "queue:webpush:delay"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestDailyAccumulatedKeyUsesTheGivenLocation(t *testing.T) {
	// 2026-09-23T00:30 in Asia/Ho_Chi_Minh is still 2026-09-22 in UTC.
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	day := time.Date(2026, time.September, 23, 0, 30, 0, 0, loc)

	if got, want := DailyAccumulatedKey("u", day), "daily:accumulated:u:2026-09-23"; got != want {
		t.Errorf("DailyAccumulatedKey = %q, want %q", got, want)
	}
}

func TestTTLs(t *testing.T) {
	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"SessionTTL", SessionTTL, 24 * time.Hour},
		{"PlacementQuizTTL", PlacementQuizTTL, 2 * time.Hour},
		{"DailyAccumulatedTTL", DailyAccumulatedTTL, 48 * time.Hour},
		{"AIRateLimitTTL", AIRateLimitTTL, time.Minute},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
