// Package store owns the Postgres and Redis clients, the migration runner and
// the Redis key topology from spec §4.
package store

import (
	"fmt"
	"time"
)

// TTLs from spec §4. queue:webpush:delay is persistent and has no TTL.
const (
	SessionTTL          = 24 * time.Hour
	PlacementQuizTTL    = 2 * time.Hour
	DailyAccumulatedTTL = 48 * time.Hour
	AIRateLimitTTL      = time.Minute

	// PetReviveTTL bounds the pet:revive hash. Not in spec §4 — chosen by the
	// pet slice: the challenge is bound to the local day it started, and 24h is
	// shorter than DailyAccumulatedTTL, whose counter the pass check reads.
	PetReviveTTL = 24 * time.Hour
)

// WebPushDelayQueueKey is the single ZSET of scheduled reminders (spec §4).
const WebPushDelayQueueKey = "queue:webpush:delay"

// SessionKey is sess:{user_id}:token — the active JWT session (TTL SessionTTL).
func SessionKey(userID string) string { return fmt.Sprintf("sess:%s:token", userID) }

// PlacementQuizKey is quiz:placement:{user_id} (TTL PlacementQuizTTL).
func PlacementQuizKey(userID string) string { return fmt.Sprintf("quiz:placement:%s", userID) }

// DailyAccumulatedKey is daily:accumulated:{user_id}:{YYYY-MM-DD}
// (TTL DailyAccumulatedTTL). The date is formatted in day's own location, so
// callers must pass a time already converted to the user's timezone.
func DailyAccumulatedKey(userID string, day time.Time) string {
	return fmt.Sprintf("daily:accumulated:%s:%s", userID, day.Format("2006-01-02"))
}

// AIRateLimitKey is ratelimit:ai:{user_id} (TTL AIRateLimitTTL, max 5 req/min).
func AIRateLimitKey(userID string) string { return fmt.Sprintf("ratelimit:ai:%s", userID) }

// PetReviveKey is pet:revive:{user_id} — the active 15-minute revival
// challenge (Hash: started_at, local_date, start_seconds; TTL PetReviveTTL).
// Not in spec §4; added by the pet slice and documented in CODEMAP.
func PetReviveKey(userID string) string { return fmt.Sprintf("pet:revive:%s", userID) }
