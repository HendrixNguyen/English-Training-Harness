---
plan: harness/plans/2026-09-27-no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/concurrent-subscription-saves-for-one-user-overshoot-the-10-.md, harness/ideas/_inbox/backend-spec-is-silent-on-push-fail-key-subscription-key-val.md]
---
# Review — Push subscriptions: validate `p256dh`/`auth`, cap 10 per user (evict oldest), prune after 3 consecutive send failures

**Plan:** `harness/plans/2026-09-27-no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md`
**Branch/worktree:** `harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint` / `.worktrees/no-per-user-subscription-cap-and-no-length-bound-on-endpoint`
**Diff:** `git diff main...harness/2026-09-27-medium-no-per-user-subscription-cap-and-no-length-bound-on-endpoint --stat`

## Plan vs idea
Delivered. The idea's open items on `main` were a per-user subscription cap and bounds on what a subscription row may hold. The endpoint length bound already existed (`MaxEndpointLength`, now pinned by a handler test); this branch adds key validation (400, nothing written), a 10-row cap with oldest-first eviction, and pruning after 3 consecutive failed sends, so neither unbounded nor undeliverable rows accumulate. Eviction instead of a 400 is justified in the plan (no unsubscribe endpoint).

## Code vs plan
Reviewed at origin head `273be93` in a detached scratch worktree (12 files, +572/-24). Merges cleanly into current `main`; no migration; off-limits paths untouched.
- Task 1 (keys): followed — `subscription_keys.go`, padded/unpadded base64url accepted, `+`/`/` refused, 65-byte `0x04` point checked on-curve by `ecdh.P256().NewPublicKey`, 16-byte auth; validated after `ValidateEndpoint` and before any write; errors never echo the value.
- Task 2 (cap): followed — insert + trim in one `pgx.BeginFunc`; fake mirrors it; service and integration tests assert 10 rows with the oldest gone. Not safe under concurrent saves (bug 1).
- Task 3 (prune): followed — `RecordFailure` (`TxPipeline` INCR+EXPIRE) / `ClearFailures` on the `Queue` interface, so `NewService` and `main.go` are unchanged; a counter error never prunes; gone/forbidden paths unchanged.
- Task 4: CODEMAP `notify`/`store` paragraphs accurate.
- The spec follow-up the plan deferred is filed (bug 2).

Re-run evidence:
```
$ gh run list --branch <branch> --limit 1
completed success docs: CODEMAP — push key validation, 10-subscription cap, prune after…  CI ... 36292338964
$ env -u … make check          -> go vet silent; 13 "ok" lines under -race
$ go test ./internal/notify -run 'Keys|SettingsHandler|UpdateSettings|AtMostTen|Tick' -count=1 -v
27 PASS, 0 FAIL, incl. TestValidateSubscriptionKeysRefusesBadMaterial, TestSettingsHandlerRejectsMalformedKeysAndOverlongEndpointsWith400,
TestUpdateSettingsKeepsAtMostTenSubscriptionsPerUser, TestTickPrunesASubscriptionAfterThreeConsecutiveFailures,
TestTickResetsTheFailureCountOnSuccess, TestTickKeepsTheRowWhenTheCounterIsUnavailable
$ git diff --stat <base> -- cmd/api internal/store/migrations internal/middleware ../project-base   -> (empty)
Integration against scratch pg 5443 / redis 6393, -p 1:
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent / ConcurrentMigrateDoesNotRace / PetStatesRejects... / RedisRoundTrip
--- PASS: TestIntegrationScheduleAndSubscriptionRoundTrip (0.15s)   (11th-endpoint eviction on real Postgres; push:fail TTL on real Redis)
Live: built ./cmd/api, scratch user + minted session, POST /api/v1/settings/notifications:
  placeholder keys ("BNc5T...")           -> 400 {"error":"invalid_request"}
  standard-base64 p256dh of a real point  -> 400 {"error":"invalid_request"}
  auth of 17 bytes                        -> 400 {"error":"invalid_request"}
  2100-char endpoint                      -> 400 {"error":"invalid_request"}
  11 distinct endpoints, real openssl P-256 key -> 200 x11; SELECT count(*) = 10, rv0 absent, rv10 present
```

## Quality
- Validation order and "nothing written on 400" hold (handler tests assert an empty call log).
- Failure semantics: a counter read/write error never deletes a row — correct under a Redis outage. `EXPIRE` is refreshed on each failure, so "consecutive" means consecutive attempts with no success between, within 7 days of each other — matches the stated design.
- Concurrency (bug 1): the trim is not serialised per user. Probe with 20 parallel saves after 10 sequential ones left 13, 15 and 11 rows on three runs; the next sequential save heals to 10.
- Noted, not filed: re-saving an existing endpoint keeps its original `created_at` (plan Notes, theoretical at 10); an orphaned `push:fail` key after eviction just expires in 7 days.
- Security: key material is validated but never logged; errors name the field only.

## Bugs filed
- `harness/ideas/_inbox/concurrent-subscription-saves-for-one-user-overshoot-the-10-.md` — low.
- `harness/ideas/_inbox/backend-spec-is-silent-on-push-fail-key-subscription-key-val.md` — low (the plan's deferred spec follow-up).

## Verdict
pass-with-bugs — keys, cap and prune all verified by unit, integration and live HTTP; two low follow-ups.
