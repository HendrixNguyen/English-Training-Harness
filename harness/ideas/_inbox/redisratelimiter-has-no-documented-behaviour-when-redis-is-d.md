---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Overtaken: the only caller (onboarding/service.go:64) fails closed on any limiter error — non-ErrRateLimited errors become 500 and no AI call is made — so the fail-open hazard is not live; documenting it and guarding a zero Limit literal are tidiness."
---
# RedisRateLimiter has no documented behaviour when Redis is down and a zero Limit blocks everything

## Why
`RedisRateLimiter.Allow` returns three distinguishable outcomes but documents two. `nil` means
proceed, `ErrRateLimited` means the user is over 5/min, and anything else — Redis down, connection
pool exhausted, `TxPipeline` failing mid-transaction — comes back as
`fmt.Errorf("airouter: rate limit: %w", err)` (`backend/internal/airouter/ratelimit.go:39-41`). The
`RateLimiter` interface comment (`:15-16`) mentions only the first two, and no code yet consumes the
third.

That third outcome is the interesting one, because the limiter exists to cap an AI bill. A caller
who writes `if errors.Is(err, ErrRateLimited) { 429 } // else proceed` fails **open**: during a Redis
outage every user's cap disappears while the most expensive endpoint in the product stays up. A
caller who writes `if err != nil { 429 }` fails **closed**: AI features are down for the duration,
which is the correct trade for a cost-bearing call and matches the `EXPIRE NX` fail-safe the plan
already chose for the crash case. Today the package leaves that decision to whoever writes onboarding,
with nothing in the godoc pointing at it — and the two idioms are equally natural to write.

A second, smaller hazard sits in the same type. `RedisRateLimiter` has exported fields and no
validation, and `NewRedisRateLimiter` is the only thing that sets `Limit`
(`ratelimit.go:24-32`). A hand-constructed `&RedisRateLimiter{Client: c}` has `Limit == 0`, so the
first `INCR` returns 1 > 0 and **every** call is rate-limited:

```
PROBE RedisRateLimiter{} zero Limit = 0 (any INCR result > 0 is rate-limited)
```

Fail-closed, so not dangerous — but it is a silent total outage of AI features from a struct literal
that compiles and reads fine.

## Expected output
The `RateLimiter` interface and `Allow` document the three outcomes explicitly, and state which way
callers must fail: an error that is not `ErrRateLimited` means the limiter could not be consulted,
and the caller **must refuse the AI call** (503) rather than proceed, because the request is
cost-bearing. A sentinel (`ErrLimiterUnavailable`) wrapping the Redis error makes that checkable
rather than a comment.

`NewRedisRateLimiter` stays the only supported constructor: either the fields become unexported, or
`Allow` treats `Limit <= 0` as `AILimitPerMinute` and says so. A unit test with a client pointed at a
closed port asserts `Allow` returns the unavailable sentinel and *not* `ErrRateLimited`, and another
asserts the zero-`Limit` behaviour, so the contract is pinned without needing a live Redis.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Task 5).
- `backend/internal/airouter/ratelimit.go:15-19` — the interface contract, silent on infrastructure errors; `:34-46` — `Allow`; `:24-27` — exported `Client`/`Limit` with no guard.
- `backend/internal/airouter/ratelimit_test.go` — one constants test plus `TestIntegrationRateLimiterAllowsFiveThenBlocks`; no case for a dead Redis and none for a zero `Limit`.
- Reviewer verification: the integration test passes against a live Redis 7 (`redis:7-alpine`, `TEST_REDIS_URL` on a scratch stack) — five allowed, sixth `ErrRateLimited`, TTL within (0, 60s]. The window semantics (fixed 60 s from the first hit, so up to 10 calls can straddle a boundary) are spec-conformant and are **not** part of this bug.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — overtaken.** Onboarding wrote the safe idiom. A `RateLimiter` doc comment can be added whenever `airouter` is next touched.
