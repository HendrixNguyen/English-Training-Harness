package quests

import (
	"strings"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func TestCounterUsesTheStoreKeyAndTTL(t *testing.T) {
	// The §4 key topology lives in store; quests must never build key strings.
	key := store.DailyAccumulatedKey("u1", mustDate(t, "2026-09-22"))
	if key != "daily:accumulated:u1:2026-09-22" {
		t.Fatalf("store.DailyAccumulatedKey = %q", key)
	}
	if store.DailyAccumulatedTTL.Hours() != 48 {
		t.Errorf("DailyAccumulatedTTL = %v, want 48h", store.DailyAccumulatedTTL)
	}
}

func TestRedisCounterKeyIsBuiltFromTheLocalDate(t *testing.T) {
	// counterKey takes the already-localised YYYY-MM-DD string so the counter
	// cannot silently use server time.
	got := counterKey("u1", "2026-09-22")
	if !strings.HasSuffix(got, ":2026-09-22") || !strings.HasPrefix(got, "daily:accumulated:u1:") {
		t.Errorf("counterKey = %q", got)
	}
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parsing %s: %v", s, err)
	}
	return d
}
