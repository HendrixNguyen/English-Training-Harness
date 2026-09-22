package airouter

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func TestRateLimitConstantsMatchSpec4(t *testing.T) {
	if AILimitPerMinute != 5 {
		t.Errorf("AILimitPerMinute = %d, want 5 (spec §4)", AILimitPerMinute)
	}
	if store.AIRateLimitTTL != time.Minute {
		t.Errorf("store.AIRateLimitTTL = %v, want 1m", store.AIRateLimitTTL)
	}
	if store.AIRateLimitKey("u") != "ratelimit:ai:u" {
		t.Errorf("store.AIRateLimitKey = %q", store.AIRateLimitKey("u"))
	}
}

// Gated on TEST_REDIS_URL, never the production REDIS_URL (spec §9). CI exports
// TEST_* and fails on --- SKIP; run with -p 1 (make test-integration).
func TestIntegrationRateLimiterAllowsFiveThenBlocks(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is unset; run `make up` and export it to run integration tests")
	}
	ctx := context.Background()
	rdb, err := store.NewRedis(ctx, url)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	const user = "airouter-integration-user"
	key := store.AIRateLimitKey(user)
	rdb.Client.Del(ctx, key)
	t.Cleanup(func() { rdb.Client.Del(ctx, key) })

	rl := NewRedisRateLimiter(rdb)
	for i := 1; i <= AILimitPerMinute; i++ {
		if err := rl.Allow(ctx, user); err != nil {
			t.Fatalf("call %d: %v, want allowed", i, err)
		}
	}
	if err := rl.Allow(ctx, user); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("call %d: err = %v, want ErrRateLimited", AILimitPerMinute+1, err)
	}

	ttl, err := rdb.Client.TTL(ctx, key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	if ttl <= 0 || ttl > store.AIRateLimitTTL {
		t.Errorf("TTL = %v, want within (0, %v] — the key must expire even after six INCRs", ttl, store.AIRateLimitTTL)
	}

	// Another user is unaffected.
	if err := rl.Allow(ctx, user+"-2"); err != nil {
		t.Errorf("second user: %v, want allowed", err)
	}
	rdb.Client.Del(ctx, store.AIRateLimitKey(user+"-2"))
}
