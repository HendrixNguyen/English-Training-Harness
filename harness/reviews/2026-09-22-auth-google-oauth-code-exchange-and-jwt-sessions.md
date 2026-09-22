---
plan: harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md, harness/ideas/_inbox/migrate-s-advisory-lock-can-hang-boot-forever-with-no-bound-.md, harness/ideas/_inbox/auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md, harness/ideas/_inbox/jwt-secret-is-accepted-at-any-length-including-one-character.md, harness/ideas/_inbox/signing-in-on-a-second-device-silently-logs-the-first-one-ou.md, harness/ideas/_inbox/integration-tests-share-one-database-and-p-1-is-a-workaround.md, harness/ideas/_inbox/service-and-require-failure-paths-are-untested-the-fakes-err.md, harness/ideas/_inbox/jwt-verify-does-not-require-exp-or-bind-iss-aud-and-bearer-i.md, harness/ideas/_inbox/stale-main-go-header-comment-and-a-double-prefixed-config-er.md]
---
# Review — Auth: Google OAuth code exchange and JWT sessions

**Plan:** `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
**Branch/worktree:** `harness/2026-09-22-high-auth-google-oauth-code-exchange-and-jwt-sessions` / `.worktrees/auth-google-oauth-code-exchange-and-jwt-sessions`
**Diff:** `git diff main...harness/2026-09-22-high-auth-google-oauth-code-exchange-and-jwt-sessions --stat`

## Plan vs idea

Delivered, line by line against the idea's *Expected output*.

| Idea asks for | State |
| --- | --- |
| `POST /api/v1/auth/google` body `{code, redirect_uri}` | `handler.go`, mounted at `cmd/api/main.go:57` |
| Exchange with `GOOGLE_CLIENT_ID`/`SECRET`, fetch `sub`/`email`/`name` | `google.go` — form POST to the token endpoint, bearer GET to userinfo |
| Upsert `users` on `google_id`, keep `cefr_current`/`target_goal` | `repo.go:33-40`, proved by the live integration test |
| JWT 24h + `sess:{user_id}:token` at the §4 TTL | `token.go:14` — `TokenTTL = store.SessionTTL`, one constant, asserted by `TestTokenExpiryMatchesTheRedisSessionTTL` |
| Response `{token, user:{id,email,full_name,cefr_current}}` | `handler.go:31-39`, asserted by `TestHandlerReturnsTokenAndUser` |
| Five scopes incl. `calendar.events` + `tasks`, `access_type=offline` | `scopes.go:18-41` |
| `auth.Require()` verifying JWT + Redis, `user_id` in context | `middleware.go` |
| `target_goal` `''` on insert, noted in CODEMAP | `repo.go:21`, CODEMAP paragraph |
| Tests: httptest Google, new vs returning upsert, four middleware rejections | all present |

The scope decision — the *sharp* argument in the idea's *Why*, that `calendar.events` and `tasks`
must be requested at first consent so no early user has to re-consent when the google slice lands —
is honoured and pinned by `TestScopesCoverOpenIDCalendarAndTasks`, which asserts the exact list and
its length. `AuthCodeURL` is exported so the frontend-shell slice cannot drift from it. That is the
part of this slice most likely to be quietly lost, and it was not.

Nothing in the *Expected output* is missing. The three open questions the evaluator recorded
(metadata blob in the session value, `JWT_SECRET` absent from spec §8, no logout endpoint) remain
open and are correctly left open — the plan explicitly ruled all three out of scope and did not
invent an API.

## Code vs plan

Tasks 1-10 all followed. Three deviations, addressed below in the order of their seriousness.

### Deviation 3 (the big one): `internal/store` changes, and `-p 1` in CI and the Makefile

**Verdict: a legitimate fix within the plan's intent, not a scope expansion. It should not have been
a `failed` plan.**

The reasoning that matters is *who created the problem*. Both CI failures are caused by this plan and
by nothing else: `internal/store` was the only package with a `TestIntegration*` until this slice
added the second one, and `backend-integration` runs every package against one Postgres service
container. The moment `internal/auth/integration_test.go` landed, `go test ./...` began running two
test binaries in parallel against one database, and the existing code — `CREATE TABLE/TYPE IF NOT
EXISTS`, which is check-then-act and not atomic — raced. The executor did not go looking for
something to improve in `store`; the plan's own Task 9 broke `store`, and the fix had to live where
the breakage was. A plan that leaves CI red is not done, and CI is this repo's stated outer
verification loop.

The counter-argument deserves a hearing: `internal/store` is outside the plan's file list, the change
is ~170 lines, it adds a public interface to a package another role owns, and there is an *open inbox
idea* asking for exactly this work — which is normally the signal to stop and hand it to the
evaluator. What makes this different is that the open idea describes a **production** hazard (two
Railway replicas booting onto a fresh database) that was not blocking anything, while what the
executor hit was the **same root cause** blocking the branch in front of it. Filing a blocker and
stopping would have parked a green-except-for-this branch behind a full evaluate/approve/execute
cycle to fix a bug the branch itself introduced. That is the wrong trade.

Two things make it land on the right side of the line rather than the wrong one:

- **It is additive, not a rewrite.** `Locker` is a new optional interface taken by type assertion
  (`migrations.go:39`); the existing `Migrator` interface is untouched, every existing fake still
  compiles, and a `Migrator` that does not implement `Locker` behaves exactly as before.
- **It is honestly tested.** I disabled the type assertion behind an env flag and ran
  `TestIntegrationConcurrentMigrateDoesNotRace` five times against a live Postgres: **5/5 FAIL**.
  Restored, it passes. That is a real regression test, not a test written to match the code. The
  three unit tests (`fakeLockingMigrator`) cover lock-once/unlock-once, unlock-after-Apply-failure,
  and lock-failure-short-circuits-EnsureVersionTable.

What it should have done differently is small and procedural, not substantive: the execution summary
documents all of this well, but the deviation is large enough that a `medium` inbox note saying "this
branch also resolved the open advisory-lock idea" would have saved the evaluator the discovery. See
*Superseded inbox idea* below.

**Is the advisory lock itself correct?** Mostly. Taking each dimension the review asked for:

- **Lock key** — fine. `pg_advisory_lock(bigint)`, one pinned constant (`migrationLockKey =
  727100001`), documented as arbitrary. Advisory locks are per-database and this is the only one the
  codebase takes, so there is no collision to worry about yet.
- **Span** — correct, and this is the part that is easy to get wrong. The lock is acquired *before*
  `EnsureVersionTable` and released by `defer unlock()` covering `EnsureVersionTable` →
  `AppliedVersions` → the `Apply` loop. A narrower span around only `Apply` would not have fixed the
  bug, because the read of `schema_migrations` is half the race. The executor got this right.
- **Transaction semantics with pgx** — correct in the respect that matters. `pg_advisory_lock` is
  session-scoped, and the session here is a connection pinned out of the pool with `m.pool.Acquire`
  and held for the whole run, so the lock genuinely outlives each individual statement. Migration
  statements run on *other* pool connections, which is fine — advisory locks serialise the callers,
  not the statements.
- **Released on every path** — **no, and this is a real bug.** See the first two bugs filed. `unlock`
  runs `pg_advisory_unlock` **on the caller's context** and discards the error
  (`postgres.go:69-72`), and pgxpool's `Release` does not reset session state (pgx v5.11.0
  `pgxpool/conn.go:32` destroys only closed / busy / in-transaction connections). So if the context
  is already cancelled at unlock time, the statement never runs, the error vanishes, and a connection
  still holding the lock goes back into the pool for up to `MaxConnLifetime` (default 1h).
  Reproduced against a live database with a temporary probe: after `cancel(); unlock()`, `pg_locks`
  still showed the lock held and a second `Lock` with a 3s deadline returned
  `timeout: context deadline exceeded`.
- **Bonus hazard found while checking the above:** `Lock` pins one connection while the migration
  needs a second, so `DATABASE_URL` with `pool_max_conns=1` deadlocks. Confirmed live: `Migrate`
  returned `store: ensuring version table: context deadline exceeded` under a 5s budget, and
  `main.go`'s `context.Background()` would make that a permanent, silent boot hang.

Neither is reachable from any caller on this branch — `cmd/api/main.go` and both integration tests
pass `context.Background()`, and nothing sets `pool_max_conns`. They are latent, so they are filed
rather than blocking. They become live the moment the open inbox idea's *secondary* ask (a bounded
context for the boot migration) is implemented, which is why the two bugs cross-reference each other.

**Is `-p 1` a fix or a bandage?** It is the right fix for this branch and a bandage for the design
underneath, and both halves of that are true at once. Right for this branch: it is one flag, it is
carried in *both* places that run the suite, and — unusually for a CI flag — it carries a nine-line
comment in `ci.yml` and a four-line one in the `Makefile` explaining precisely why it is load-bearing,
so nobody deletes it as noise. The CODEMAP entry says "`-p 1` is load-bearing, not cosmetic". That is
better documentation than most such flags ever get.

But the design flaw is untouched: `internal/store`'s `reset()` still `DROP`s the whole schema of a
database every other package's tests are using. `-p 1` is scheduling, not isolation, so the collision
is prevented by the command line rather than by the code. It comes back via a `t.Parallel()` inside
any package (which `-p 1` does not constrain), a developer running two package test commands in two
shells, a bare `go test ./... -run Integration` typed from memory, or a second CI job sharing the
container — and it serialises a suite that seven more slices will each add to. Filed as a medium:
the durable fix is a per-package schema (`search_path`) or database, after which `-p 1` becomes a
performance preference rather than a correctness requirement.

### Deviation 2: `DATABASE_URL` → `TEST_DATABASE_URL` in `integration_test.go`

Correct, and the executor handled it exactly as it should be handled. Judging the correction and not
the plan, as instructed: the plan was written before the `TEST_*` convention existed, and gating on
the production `DATABASE_URL` would have been wrong twice over — `internal/store`'s tests already
establish `TEST_DATABASE_URL` as the only allowed gate *because these tests drop data*, and CI's
`backend-integration` job exports only the `TEST_*` pair, so the test would have silently `SKIP`ped
and tripped the workflow's own no-skip guard. The executor changed the gate, updated the CODEMAP
wording to match, and committed it separately (`54ee414`) so both the plan's mistake and the fix stay
visible in history. That is the right instinct: the correction is not hidden inside a task commit.

### Deviation 1: `gotForm` typed `url.Values` instead of `map[string][]string`

Trivial and necessary — the plan's test called `.Get()`, which only exists on `url.Values`. Same
assertions, compiles.

### Plan documentation inaccuracy (executor-reported, confirmed)

Task 3's verification command `go test ./internal/auth/... -run Token -v` claims six `--- PASS`
lines; `-run` is a substring regex over test *names*, and only two of the six contain "Token". The
executor spotted this, verified with the full package run, and reported it rather than quietly
adjusting the expectation. Correct handling.

## Superseded inbox idea

**`harness/ideas/_inbox/migrations-run-on-every-boot-with-no-advisory-lock.md` (medium) is resolved
in its primary ask but NOT fully, and should not be rejected outright as superseded.**

Stating it plainly for the evaluator, because the reviewing instructions asked for a clear call:

- **Resolved.** Its main ask — "take `pg_advisory_lock(<constant>)` on a dedicated connection at the
  top of `Migrate`, release it at the end" — is implemented exactly as described, on a dedicated
  pinned connection, with the span covering all three steps. Its secondary ask for honesty — "a test
  with a fake `Migrator` that records lock/unlock ordering keeps it honest" — is implemented as
  `fakeLockingMigrator` plus three unit tests, and goes further with a live-database concurrency
  test that I verified fails 5/5 without the fix. The production crash-loop the idea describes (two
  Railway replicas booting onto a fresh database) can no longer happen.
- **Not resolved.** Its closing paragraph: *"`main.go` should not use a deadline-free
  `context.Background()` for the migration, so an unreachable database fails the boot in bounded time
  rather than hanging."* `cmd/api/main.go:19` is still `ctx := context.Background()`. And this branch
  makes that gap **worse, not neutral**: `pg_advisory_lock` is the blocking variant, so where an
  unreachable database used to hang, now a *contended lock* also hangs, indefinitely and with no log
  line, and `pool_max_conns=1` hangs outright.

So the right disposition is to narrow the idea to its remaining secondary item rather than close it —
and to sequence it after
`harness/ideas/_inbox/migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md`, because
adding a deadline before fixing the unlock path converts a latent lock leak into a live one. I have
not edited that idea file, as instructed.

## Quality

### Security — the part this slice most needed to get right

The core of it is solid, and worth saying so before the criticisms. **Algorithm confusion is pinned
twice**: the keyfunc rejects any non-`*jwt.SigningMethodHMAC` method *and* `jwt.WithValidMethods`
restricts to HS256, and `TestVerifyRejectsTheNoneAlgorithm` uses a real hand-built `alg=none` token
rather than asserting on a constant. `TestVerifyRejectsAnotherSecret` covers the wrong-key case.
`exp` is enforced through an injectable clock, so `TestVerifyRejectsAnExpiredToken` tests the real
expiry path rather than sleeping. Google is reached over HTTPS with a 10s client timeout, response
bodies are read through `io.LimitReader(..., 1<<20)`, and the profile is rejected when `sub` or
`email` is missing. The upsert keys on `sub`, never on email, so an unverified-email takeover is not
possible. Claims are minimal — `sub`, `iat`, `exp` — with no PII in the token.

Google's error body does **not** leak to the client: `doJSON` embeds it in the error string, but
`Handler` discards the error entirely and returns a fixed `{"error":"google_auth_failed"}`. Verified
live against the running binary. The flip side is that it leaks nowhere at all — see below.

Three security-shaped gaps filed:

- **`JWT_SECRET` has no minimum strength** (medium). `config.Load` checks non-empty only; I booted
  the branch's binary with `JWT_SECRET=x` and it served both routes. HS256's entire security is the
  secret, and this is the one variable that is **absent from spec §8**, so whoever provisions Railway
  is inventing it from nothing with no stated requirement. The config loader is the only thing that
  could catch `secret`, and it does not. RFC 7518 §3.2 wants ≥32 bytes.
- **No `iss`/`aud` binding and `exp` not *required*** (low). `WithExpirationRequired()` is not passed,
  so a same-secret token with no `exp` verifies forever; and nothing binds tokens to this service, so
  a `JWT_SECRET` reused in staging or a worker would cross-authenticate.
- **Session revocation is strictly single-device** (medium). This one is a design question, not a
  defect: `Put` overwrites `sess:{user_id}:token` and `Require` compares byte-for-byte, so signing in
  on a phone silently logs the laptop out, and `prompt=consent` means the laptop then gets the full
  Google dialog again. It is deliberate, documented and tested — but it was chosen to make revocation
  work, not because the product asked for it, and §4 does not require it. For an installable PWA whose
  §5.1 loop starts with a notification tap, phone-plus-laptop is the normal case. Better settled now,
  before six slices harden the semantics behind `Require`.

On the byte-for-byte comparison specifically: `stored != raw` is not constant-time, but it is not
exploitable — `Verify` runs first, so an attacker must already hold a validly-signed token for that
user before the comparison is reached. Not filed.

### Error handling and failure modes — the weakest dimension

`auth` collapses every failure into "your credentials are bad" and logs nothing anywhere. `Handler`
maps *any* `SignIn` error to `401 google_auth_failed` — including Postgres being unreachable, the
`users` upsert violating a constraint, and Redis refusing the session write. `Require` maps any
`sessions.Get` error, including a Redis transport failure, to `401 unauthorized`. So a Redis blip
logs the entire user base out rather than returning 503, and a database outage presents to users as
"Google sign-in failed" and to the operator as silence. I confirmed live that a failed sign-in
produces **zero** log output.

There is a concrete data-shaped instance: §3.2 makes `users.email UNIQUE NOT NULL`, but the upsert's
conflict target is `(google_id)` alone. A second Google account presenting an email that already
belongs to a different `google_id` raises a `users_email_key` violation that `ON CONFLICT
(google_id)` does not catch — that user is permanently unable to sign in, sees `google_auth_failed`,
and nothing is logged. Nobody will diagnose that from outside. Filed as medium; this is the first
package in the codebase handling a request that can fail for infrastructure reasons, so whatever
convention it sets is what six more slices will copy.

### Test honesty — good, with one specific hole

The suite is honest, which is the question worth asking. The injectable-endpoint design does make the
tests real rather than merely green: `TokenURL`/`UserInfoURL` are fields on the *production*
`GoogleClient`, so `google_test.go` exercises the actual code path — form encoding, `Content-Type`,
the `Bearer` header, status handling, JSON decoding, the empty-`access_token` guard — against
`httptest`, rather than a parallel test-only implementation. `TestExchangeSendsTheAuthorizationCodeGrant`
asserts the *outgoing* form values, which is the thing that would silently break against real Google.
`Service` then uses a separate `fakeExchanger`, correctly layered. The four middleware rejection cases
the idea asked for (missing, malformed, expired, revoked) are all present, plus superseded. The
integration test is a genuine round trip: it progresses the learner to `B2 / IELTS 7.0`, re-logs in
with an empty refresh token, and reads back through SQL that `target_goal` and the stored refresh
token survived — that is the actual contract, not a restatement of the code. `repo_test.go`
additionally slices the SQL at `DO UPDATE` and asserts `cefr_current`/`target_goal` never appear on
the update side, which pins the regression that matters even without a database.

The hole: **`fakeRepo.err` and `fakeSessions.err` are declared and honoured inside the fakes, and no
test ever sets either one.** Only `fakeExchanger.err` is used. So `SignIn` with a failing repo, `SignIn`
with a failing `Put` — including the ordering guarantee asserted in the comment at `service.go:48-49`
— and `Require` with a session-store transport error are entirely unexercised. Those are exactly the
paths the 401-for-everything bug is about, and unused error fields on a fake read as coverage that
does not exist. Filed as low. `fakeSessions.Delete` also ignores `f.err` where `Put`/`Get` honour it.

### Design, boundaries and conventions

Clean. Four collaborators behind interfaces (`Exchanger`, `UserRepo`, `SessionStore`) with a concrete
`*TokenIssuer`, wired in `NewService` — the handler and middleware are genuinely thin. `var _ Iface =
(*Impl)(nil)` assertions on all three implementations, matching `store`'s style. No cross-package
table access: `auth` owns `users` and reaches Redis only through `store.SessionKey`/`store.SessionTTL`,
never a literal key string, and `TestSessionKeyComesFromStore` pins that boundary explicitly. Comments
cite spec sections (§3.2, §4, §5.1, §7) the way `store` does. Naming and file layout match the
existing package. Idiomatic Go: wrapped errors throughout, `context.Context` first, injectable clock,
`defer func() { _ = resp.Body.Close() }()` for the errcheck-clean close the CODEMAP's linter note
cares about.

Resource use is appropriate to how this will be called: one bounded HTTP client with a timeout, a
single-statement upsert with no N+1, `RETURNING` instead of a follow-up SELECT, no per-request
allocation of clients, no goroutines. Nothing to raise.

One nit worth recording about the `Locker` type assertion: a future `Migrator` implementation that
forgets to implement `Locker` silently gets no locking and no warning. Acceptable — it is the price
of keeping the change additive, and it is documented on the interface — but it is implicit.

### Documentation

`cmd/api/main.go`'s package comment still says "This slice serves only `GET /healthz`" fifteen lines
above where it mounts `POST /api/v1/auth/google`, and `log.Fatalf("config: %v", err)` doubles a prefix
the error already carries (observed: `config: config: DATABASE_URL is required`). Filed as low.

### CODEMAP

**Accurate — no correction needed.** I checked both rewritten paragraphs against the code. The `auth`
paragraph correctly describes the upsert semantics, the `target_goal = ''` contract and why, the
scope list, the three-part `Require` acceptance rule, `TEST_DATABASE_URL` (not the plan's
`DATABASE_URL`), and flags `JWT_SECRET`'s absence from spec §8. The `store` paragraph correctly adds
the `Locker`/advisory-lock behaviour with its root cause, and the CI paragraph correctly updates the
`backend-integration` command and explains why `-p 1` is load-bearing. The executor also updated the
CI section, which is easy to forget. Nothing to fix, so no commit on the branch beyond `harness/`.

## Verification output

Re-run by me in `.worktrees/auth-google-oauth-code-exchange-and-jwt-sessions`. Everything in the
plan's *Verification* section and the executor's *Runtime proof* reproduces. **No executor gate
failure.**

```
$ go build ./... && go vet ./...
(no output)

$ env -u DATABASE_URL -u REDIS_URL -u GOOGLE_CLIENT_ID -u GOOGLE_CLIENT_SECRET -u JWT_SECRET go test ./... -count=1
?   	.../backend/cmd/api	[no test files]
ok  	.../backend/internal/auth	0.748s
ok  	.../backend/internal/config	1.724s
ok  	.../backend/internal/health	1.134s
ok  	.../backend/internal/store	1.499s
```

Runtime proof, re-run against a scratch stack. The owner's `scio3-redis-1` holds 6379 as stated, and
`mls-demo-redis` has since taken **6380**, so I used `POSTGRES_PORT=5433 REDIS_PORT=6381` via a
scratch `backend/.env`, torn down and deleted afterwards. `docker ps` before and after shows the four
pre-existing containers untouched.

```
$ make test-integration      # TEST_DATABASE_URL/TEST_REDIS_URL on 5433/6381
--- PASS: TestIntegrationUpsertCreatesThenPreservesTheLearnerState (0.08s)
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.10s)
--- PASS: TestIntegrationConcurrentMigrateDoesNotRace (0.11s)
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.07s)
--- PASS: TestIntegrationRedisRoundTrip (0.00s)
ok  	.../internal/auth 1.292s     ok  	.../internal/store 0.907s

$ ./api   (DATABASE_URL/REDIS_URL on 5433/6381, fake Google creds, JWT_SECRET=x, PORT=8099)
[GIN-debug] GET    /healthz                  --> ...health.Handler.func1
[GIN-debug] POST   /api/v1/auth/google       --> ...auth.Handler.func1
listening on :8099

$ curl /healthz                                   → 200 {"postgres":"ok","redis":"ok","status":"ok"}
$ curl -X POST /api/v1/auth/google -d '{}'        → 400 {"error":"invalid_request"}
$ curl -X POST /api/v1/auth/google -d '{"code":"bogus",...}'
                                                  → 401 {"error":"google_auth_failed"}
                                                    (server log: nothing — see the 401/logging bug)
$ env -u ... ./api                                → config: config: DATABASE_URL is required, exit 1

$ git status --short        (worktree)            → clean
```

Two additional probes I ran, each with a temporary test file removed immediately afterwards
(worktree confirmed clean after both):

```
# Is TestIntegrationConcurrentMigrateDoesNotRace honest?
# Type assertion in Migrate disabled behind an env flag, 5 runs:
FAIL  FAIL  FAIL  FAIL  FAIL      (5/5) — restored: PASS. The regression test is real.

# Does unlock release the lock when the caller's context is cancelled?
advisory lock still held after unlock-on-cancelled-ctx: true
SECOND CALLER BLOCKED/FAILED: store: acquiring the migration lock: timeout: context deadline exceeded

# Does Migrate survive pool_max_conns=1?
Migrate with pool_max_conns=1: err=store: ensuring version table: context deadline exceeded
```

CI is green at branch HEAD `75b56d3`: run
[35742567690](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35742567690),
all three jobs. I read both red predecessors and they match the executor's account exactly — 35740944612
`duplicate key value violates unique constraint "pg_type_typname_nsp_index"`, 35742036352
`first run applied [], want [0001_init]`.

## Bugs filed

None blocks the merge. Nothing here is data loss, a security hole that this branch opens, a broken
developer workflow, or a dishonest test — the two `store` bugs are latent with no reachable caller on
this branch, and the rest are hardening and observability work that belongs in the queue.

| Priority | Bug |
| --- | --- |
| high | `migrate-s-advisory-lock-leaks-into-the-pool-when-unlock-runs.md` — `unlock` runs on the caller's context and swallows the error; pgxpool does not reset session state, so a cancelled context leaves the lock held by a pooled connection for up to an hour. Reproduced live. Latent today (every caller passes `context.Background()`); live the moment the boot migration gets a deadline. |
| medium | `migrate-s-advisory-lock-can-hang-boot-forever-with-no-bound-.md` — blocking `pg_advisory_lock` + `context.Background()` turns a visible crash-loop into a silent hang; `pool_max_conns=1` deadlocks outright. Reproduced live. |
| medium | `auth-reports-postgres-and-redis-failures-as-401-and-logs-not.md` — infrastructure failures surface as 401 and are logged nowhere; includes the permanent, undiagnosable lockout on a `users_email_key` collision. |
| medium | `jwt-secret-is-accepted-at-any-length-including-one-character.md` — presence-only validation on the one secret that is missing from spec §8. `JWT_SECRET=x` boots. |
| medium | `signing-in-on-a-second-device-silently-logs-the-first-one-ou.md` — one session per user, chosen for revocation rather than for the product; settle it before six slices mount behind `Require`. |
| medium | `integration-tests-share-one-database-and-p-1-is-a-workaround.md` — the destructive `reset()` still targets a shared database; `-p 1` is scheduling, not isolation. |
| low | `service-and-require-failure-paths-are-untested-the-fakes-err.md` — `fakeRepo.err` and `fakeSessions.err` are declared and never set. |
| low | `jwt-verify-does-not-require-exp-or-bind-iss-aud-and-bearer-i.md` — no `WithExpirationRequired`, no `iss`/`aud`, case-sensitive `Bearer`. |
| low | `stale-main-go-header-comment-and-a-double-prefixed-config-er.md` — "serves only GET /healthz" above the auth route; `config: config: ...`. |

## Verdict

**`pass-with-bugs` — mergeable.**

The plan delivers the idea, including the part most likely to have been lost (the full scope list at
first consent, so no early user has to re-consent when the google slice lands). The security
foundation is sound where it counts: the signing algorithm is pinned twice and proved with a real
`alg=none` token, `exp` is enforced through an injectable clock, Google's error body cannot reach a
client, the upsert keys on `sub` rather than email, and the JWT `exp` and the Redis TTL come from one
constant so they cannot drift. The tests are honest — the injectable endpoints exercise the
production `GoogleClient` rather than a stand-in, and the new concurrency regression test fails 5/5
when I disabled the fix.

The third deviation was the right call. The executor broke `internal/store` by adding the second
package with integration tests and fixed it where the breakage was, additively and with a genuine
regression test, rather than parking a green branch behind a full evaluator cycle for a bug the
branch itself introduced. Two real defects in that new code — the unlock-on-cancelled-context leak
and the `pool_max_conns=1` deadlock — are filed with live reproductions; neither is reachable from
any caller on this branch.

The weakest dimension is error handling: every infrastructure failure in `auth` becomes a 401 and is
logged nowhere. That is worth fixing early, because this is the first package in the codebase to
handle a request that can fail for infrastructure reasons and the six remaining slices will copy
whatever it does.

No PR exists (`gh pr create` failed on permissions during execution), so the PR steps are skipped.
