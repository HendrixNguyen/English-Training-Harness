---
plan: harness/plans/2026-09-25-auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/auth-session-store-flattens-the-redis-error-with-v-so-a-clie.md, harness/ideas/_inbox/login-shows-generic-retry-copy-for-409-email-in-use-so-a-loc.md]
---
# Review — auth: a Google rejection is 401, an outage is 5xx, and every failure is logged

**Plan:** `harness/plans/2026-09-25-auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md`
**Branch/worktree:** `harness/2026-09-25-medium-auth-reports-postgres-and-redis-failures-as-401-and-logs-not` / `.worktrees/auth-reports-postgres-and-redis-failures-as-401-and-logs-not`
**Diff:** `git diff main...harness/2026-09-25-medium-auth-reports-postgres-and-redis-failures-as-401-and-logs-not --stat`

## Plan vs idea
Delivered. Every item in the idea's *Expected output* exists:
- `ErrGoogleRejected` → 401. Repo/JWT/transport → 500. Session-store write → 503.
- `Require` separates `ErrNoSession` (401) from a transport error (503).
- One log line per failure names the google id or user id and no token. `TestHandlerLogsTheFailureWithoutTheTokens` pins this.
- Response bodies stay opaque codes.
- A repo failure and a session failure are tested as 5xx.
- The `users_email_key` collision is decided as `409 email_in_use`.

The idea's "log at error level" is `log.Printf`. The stdlib `log` has no levels, and the plan said so, so this is acceptable. The frontend copy for 409 was left as a follow-up in the plan's Notes, and I filed it (low).

## Code vs plan
Reviewed at origin head `58a41e0` in a detached reviewer worktree. Diff base is `eefe92f`, with 14 files and +381/−14. `backend/internal/auth` is unchanged on `origin/main` since the base. The only conflict with `origin/main` (`27f57e1`) is `harness/CODEMAP.md`, from `git merge-tree`, and the orchestrator resolves it when building the daily branch.

- Task 1, sentinels and the Google leg: followed.
- Task 2, the session store: followed. Deviation 1 is justified: the plan's gated `TestRedisSessionStore…` never existed, so the unreachable-port test is ungated.
- Task 3, repo 23505 → `ErrEmailTaken` and the service wrap: followed. The integration test exists and passes.
- Task 4, handler and `Require` mapping plus logging: followed. Deviation 2 is justified: the old `errGoogleRejected` fixture now wraps the sentinel, so the existing 401 test still exercises the real mapping.
- Task 5, CODEMAP: followed. The two sentences are present in the `auth` paragraph.

Verification, re-run by the reviewer from `backend/`:
```
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/auth/ -count=1 -race -v | grep -c '^--- PASS'
41            (0 FAIL; only the 2 gated TestIntegration* SKIP)
grep -rn '"error": *"' internal/auth/*.go | grep -v _test | grep -o '"[a-z_]*"' | sort -u
"email_in_use" "error" "google_auth_failed" "internal_error" "invalid_request" "unauthorized" "unavailable"
env -u … make check        -> fmt-check silent, vet silent, every package ok under -race
python3 tools/harness/cli.py validate -> exit 0
# isolated stack (COMPOSE_PROJECT_NAME=rv-auth, pg 5442, redis 6392)
go test ./internal/auth/ -run Integration -count=1 -v -p 1
--- PASS: TestIntegrationUpsertCreatesThenPreservesTheLearnerState (0.28s)
--- PASS: TestIntegrationUpsertRejectsAnEmailOwnedByAnotherGoogleAccount (0.04s)
go test ./... -count=1 -p 1 (all integration)  -> every package ok; only SKIPs are the gate self-tests' "gated" subtests
gh run list --branch <branch> --limit 1 -> completed success, run 36093540852
```
Runtime proof, re-run by the reviewer: I built `./cmd/api` and booted it on the isolated stack with a scratch `JWT_SECRET`. A throwaway `cmd/rvproof` minted and stored a session for a fake user id, and I deleted it right away (`git status` clean).
- With Redis stopped, `GET /api/v1/pet/status` answered `503 {"error":"unavailable"}` and logged one line per call: `auth: session store unavailable for user 00000000-…abcd: … dial tcp [::1]:6392: connect: connection refused`.
- After Redis restarted, the same token answered `500 internal_error`, not 401. The fake user has no row, which is the same result the executor saw. The session survived.
- My first call raced boot and got no response (`000`). That is not a finding.

## Quality
- Boundaries: everything stays inside `auth`. Callers use the `SessionStore`/`UserRepo` interfaces, and there is no cross-package table access.
- Tests are honest. The handler table drives each sentinel through the real `Service`. The log test checks for the absence of the Google access and refresh tokens and the JWT secret. The unreachable-Redis test uses a real go-redis client. The middleware tests cover 503 and pin that `ErrNoSession` still answers 401.
- Error handling: the session-store wraps use `%v` for the cause, which drops it from the chain. A client-aborted request (`context.Canceled`) is then logged and answered as a Redis outage, and the log line repeats its prefix. Filed low.
- A Google 429 maps to 401 by explicit design and is documented in `doJSON`. That is not a finding.
- Security: no token reaches a log or a response body. Google's raw error body appears only in the server log, as the plan intended.
- CODEMAP: accurate for the branch. It conflicts textually with main's newer CODEMAP, so resolve it at integration.

## Bugs filed
- `harness/ideas/_inbox/auth-session-store-flattens-the-redis-error-with-v-so-a-clie.md` (low): `%v` drops the cause, so a client cancel is logged as an outage, and the log prefix is doubled.
- `harness/ideas/_inbox/login-shows-generic-retry-copy-for-409-email-in-use-so-a-loc.md` (low): `/login` has no copy for `email_in_use`. This is the plan's own named follow-up.

## Verdict
**pass-with-bugs.** Plan and idea are delivered, all verification reproduces and CI is green. There are no blockers. The `harness/CODEMAP.md` conflict with `origin/main` has to be resolved when this branch is merged into the daily branch.
