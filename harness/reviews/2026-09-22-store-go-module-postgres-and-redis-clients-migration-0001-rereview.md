---
plan: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md, harness/ideas/_inbox/gin-default-ships-debug-mode-and-all-proxies-trusted-to-prod.md, harness/ideas/_inbox/root-gitignore-env-silently-swallows-every-module-s-env-exam.md, harness/ideas/_inbox/cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md, harness/ideas/_inbox/backend-env-example-omits-the-app-s-own-database-url-redis-u.md]
---
# Re-review — Store: Go module, Postgres and Redis clients, migration 0001

**Plan:** `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
**Branch/worktree:** `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001` / `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001`
**First review:** `harness/reviews/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md` (`pass-with-bugs`, 8 bugs, 2 blockers) — kept intact; this file is the re-review after the two amending plans landed.
**Amending plans re-verified:**
`harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md`,
`harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md`

## Both blockers are genuinely fixed

Five commits landed on the branch since the first review, all carrying the required
`Co-Authored-By` trailer, all confined to seven files — `.env.example`, `.gitignore`, `Makefile`,
`docker-compose.yml`, `integration_test.go`, the new `integration_gate_test.go`, and one CODEMAP
clause. Nothing else on the branch moved (`git diff --name-only 4690cd2..HEAD` touches no `.sql`, no
`health.go`, no `main.go`, no `migrations.go`).

### Blocker 1 — `go test` no longer drops tables

The original reproduction, re-run verbatim. Previously these three tests *ran* and failed inside
`reset()`; they must now skip, and no `down migration:` line may appear:

```
$ env -u TEST_DATABASE_URL -u TEST_REDIS_URL \
    DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
    env -u REDIS_URL go test ./internal/store/ -count=1 -run Integration -v
--- SKIP: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.00s)
    integration_test.go:63: TEST_DATABASE_URL is unset; these tests DROP every table, …
--- SKIP: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.00s)
--- SKIP: TestIntegrationRedisRoundTrip (0.00s)
PASS
ok  	…/internal/store	0.540s
exit=0

$ grep -n 'os.Getenv' internal/store/integration_test.go
16:	url := os.Getenv("TEST_DATABASE_URL")
32:	url := os.Getenv("TEST_REDIS_URL")
46:	if os.Getenv("TEST_DATABASE_URL") == ""
```

Every `os.Getenv` in the destructive file now reads a `TEST_`-prefixed name; `DATABASE_URL` survives
only inside an explanatory comment. The second line of defence — `reset()` refusing outright without
`TEST_DATABASE_URL` (`integration_test.go:46-48`) — is present as the plan specifies.

**This is a real gate, not an always-skip.** That distinction is the whole point and it is *not*
provable from the suite (see the bug filed below), so I proved it by hand against a live database:

```
$ export TEST_DATABASE_URL=postgres://english:english@localhost:5433/english?sslmode=disable
$ export TEST_REDIS_URL=redis://localhost:6380/0
$ make test-integration
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.08s)
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.04s)
--- PASS: TestIntegrationRedisRoundTrip (0.00s)
ok  	…/internal/store	0.943s
```

`internal/config` is untouched, so `DATABASE_URL`/`REDIS_URL` keep their spec §8 production meaning
and have simply stopped being test triggers. That is the right shape: the fix removes the hazard
rather than relocating it, and it did not need a build tag.

### Blocker 2 — `make up` works on this machine

```
$ env -u REDIS_PORT -u POSTGRES_PORT docker compose config | grep 'published\|postgres_data'
published: "5432"   published: "6379"   source: postgres_data   postgres_data: (name backend_postgres_data)
$ POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose config | grep 'published'
published: "5433"   published: "6380"
$ docker compose config --quiet   → exit 0
```

Then the thing the first review could not do at all. I copied `.env.example` to `.env`, set
`POSTGRES_PORT=5433` / `REDIS_PORT=6380`, and ran the documented commands with no flags and no files
outside the repo:

```
$ make up
backend-postgres-1	running	0.0.0.0:5433->5432/tcp
backend-redis-1	running	0.0.0.0:6380->6379/tcp
$ git status --short          → clean (the .env is ignored, as required)
```

The binary boots against that stack and answers:

```
2026/09/22 17:00:57 migrations applied: [0001_init]
2026/09/22 17:00:57 listening on :8099
$ curl -s -w ' HTTP %{http_code}' http://127.0.0.1:8099/healthz
{"postgres":"ok","redis":"ok","status":"ok"} HTTP 200
```

`make down` then left `backend_postgres_data` in place, confirming the named volume does what its
comment claims. Afterwards: scratch `.env` removed, worktree clean, no `backend-*` container
running, and the owner's unrelated container untouched —
`docker inspect scio3-redis-1` → `StartedAt=2026-09-22T01:56:03Z Status=running RestartCount=0`,
i.e. the same start time as before this review, never stopped or restarted.

### Did the fixes introduce anything?

One thing, and it is about the test rather than the code: **the gate tests only assert the skip
direction**, so they would still pass if either helper were changed to skip unconditionally — the
integration suite would go silently dead and `make test-integration` would look identical. Filed
`medium`. Everything else checks out: the two amending plans touched disjoint files apart from
different comment blocks in the `Makefile` (no conflict, both present and coherent), the compose
change parameterises only the host side so in-network URLs are unaffected, Redis correctly gets no
volume (its contents are TTL-bounded by spec §4), and `go build ./... && go vet ./...` and
`make tidy` are clean with no working-tree churn.

Harness bookkeeping is honest rather than bypassed: `blockers` exits 0 because
`scan.blockers_for` resolves a blocker once *its own fix plan* is `status: done`
(`tools/harness/scan.py:33-42`), and both fix plans are `done`. The two blocker ideas still carry
their `blocks:` pointer, which is the intended record.

## The six remaining bugs from the first review

None were expected to be fixed here and none were. Each subject file is byte-identical to what the
first review read (`git diff --name-only 4690cd2..HEAD` lists no Go source outside the two
integration test files, and no SQL):

| Bug | Status |
| --- | --- |
| `healthz-leaks-postgres-and-redis-driver-error-strings-public` (medium) | unchanged — `health.go` untouched |
| `migrations-run-on-every-boot-with-no-advisory-lock` (medium) | unchanged — `main.go`, `migrations.go` untouched |
| `no-ci-runs-the-integration-suite-so-pgmigrator-is-never-veri` (medium) | unchanged, and **more acute**: now that the destructive tests require deliberate `TEST_*` nomination, an accidental local run can no longer stand in for CI, so CI is the only remaining path by which `PgMigrator` is ever executed |
| `migrate-is-only-tested-against-the-single-embedded-migration` (medium) | unchanged |
| `healthz-shares-one-2s-deadline-across-two-sequential-pings` (low) | unchanged |
| `spec-3-2-leaves-user-id-nullable-on-three-child-tables` (low) | unchanged — `0001_init.up.sql` untouched |

Not re-filed.

## The two executor deviations

**Deviation 1 — `!.env.example` negation in `backend/.gitignore`. Correct, and the better of the two
available options.** The plan's Task 2 Step 2 asserted "`.env.example` is unaffected by that pattern
and stays tracked — Step 4 proves it". Step 4 proved the opposite, which is exactly what a
verification step is for. The executor did not paper over it: `git add -f` was rejected with the
right reason — it would have left the file permanently invisible to `git status`, so the *next*
edit to the template would also need `-f` and would quietly be forgotten. The negation is sound
git: `.env*` in the root excludes a *file*, not a parent directory, so re-inclusion is permitted,
and a deeper `.gitignore` takes precedence. Verified from the worktree:

```
$ git check-ignore -v .env           → backend/.gitignore:2:.env	.env        (exit 0, ignored)
$ git check-ignore -v .env.example   → (no output, exit 1 — NOT ignored)
$ git ls-files .env.example          → .env.example
```

The three-line comment above the negation explaining *why* it exists is the part that makes this
maintainable — without it the line reads as noise and gets deleted in a future tidy-up.

**Deviation 2 — CODEMAP updated though the plan's file table omitted it. Correct.** The plan's own
goal is to make `make up` work; the CODEMAP is the file that tells a human to run `make up`, so
leaving it silent about `.env` would have re-created the documentation gap the plan exists to close,
and the executor role independently requires a CODEMAP update for any package changed. The change is
one clause and I verified each claim in it: `make down` does keep the `postgres_data` volume, the
`.env.example` path is right, and the test-command sentence is now true unconditionally. Both
deviations were disclosed in the execution summary rather than discovered by me.

I made **no CODEMAP correction** — it is accurate as it stands, so the skill's step 7 does not
apply. One readability note for whoever writes slice 2: the `store` bullet is now a single
~1100-character paragraph carrying module path, stack, migration call, schema facts, key-builder
rule and four test commands. Every role reads this file first. It should be split when the second
package lands; it is not wrong today, so I have not filed it.

## Quality review of the whole branch

This is the first application code in the repo, so I judged it as the template every later slice
copies rather than as one feature.

**Design and boundaries — the strongest part of the branch, unchanged and still holding.**
`internal/health` never imports `internal/store`; it declares its own one-method `Pinger` and
`main.go` injects the concretes. That is consumer-side interface definition, correct Go idiom, and
exactly what CODEMAP's "packages talk through interfaces" means. `store.Migrator` gives the runner
the same seam with a compile-time `var _ Migrator = (*PgMigrator)(nil)` assertion, and it is what
made the fs-level testing possible. `cmd/api/main.go` is 51 lines of wiring with no logic. The
forward-looking weakness I raised last time stands: `Postgres.Pool` and `Redis.Client` are exported,
so nothing in code prevents a future handler doing `pg.Pool.Query` directly. Deliberate — later
slices are raw SQL — but it means the "no cross-package table access" rule has no enforcement point,
and slice 2's reviewer should check that `auth` owns its queries behind a repository type.

**Error handling.** The wrapping is consistent and uses `%w` throughout, and `PgMigrator.Apply`
wraps DDL plus the version row in one transaction with a `defer tx.Rollback`, which is right. Two
idiom problems in the foundation. First, prefixes are applied twice: `config.go:25` returns
`config: DATABASE_URL is required` and `main.go:21` logs `config: %v`, producing
`config: config: DATABASE_URL is required` — I reproduced it; the same double-prefix shape exists at
`main.go:26`, `:32` and `:38` over `store:`-prefixed errors. Go convention is that the error carries
its own context and the caller does not restate the package. Second, every failure in `main` is
`log.Fatalf`, which means `os.Exit` and therefore no deferred cleanup ever runs — covered in the
graceful-shutdown bug below.

**Correctness on untried inputs.** `Migrate` is careful: glob, `sort.Strings`, skip applied, and it
returns the partial `applied` slice alongside the error so a caller can see how far it got. Behaviour
on an empty FS (returns `nil, nil`) is reasonable but silent, and `main.go:40` only logs when
something was applied — so "migrated nothing because the directory was empty" and "migrated nothing
because everything was current" are indistinguishable in the log. Worth a line when 0002 lands.
`DailyAccumulatedKey` correctly documents that it formats in the caller's location, which pushes the
timezone decision to the caller — the right call for a spec that keys daily progress per user.

**Performance and resource use.** Nothing to fault at this size. The pgx pool is process-wide and
lazy (`NewPostgres` parses without dialling), `AppliedVersions` is one query with `rows.Close`
deferred and `rows.Err()` checked, and there are no queries in loops. The unbounded thing is the
pool's own defaults — pgx's `max_conns` is left at 4×NumCPU with no `DATABASE_URL` parameters
documented — which is fine for a slice with one read-only endpoint but is worth a config knob before
the daily loop lands.

**Security.** Two findings. The `/healthz` driver-error leak is already filed and unchanged. New:
`main.go:44` uses a bare `gin.Default()`, so the deployed binary runs in Gin's **debug** mode and
leaves `TrustedProxies` at its default of everything — I saw both warnings in the boot log above.
Nothing calls `ClientIP()` today, but the auth and AI-rate-limit slices are its natural first
callers and this is the engine they will mount on. Filed `medium`.

**Test honesty.** I ran a dedicated test-gap pass over the whole slice. Coverage on the default run:
`config` 100%, `health` 100%, `store` 45.7%, `cmd/api` 0%, with every wire-speaking function at 0%.
The skips are genuine skips with printed reasons, `t.Setenv` is used correctly, nothing is parallel
or order-dependent, and I found no flakiness. The substantive gaps are the ones already filed —
the `strings.Contains`-on-SQL-text tests standing in for executing the DDL, `Migrate`'s untested
error branches, and no multi-migration/ordering coverage because `MigrationsFS` embeds exactly one
file (`Migrate` already takes `fs.FS`, so `fstest.MapFS` closes this without a database; doing it
now matters because this is the template later migration slices copy). Two smaller observations I am
*not* filing: `TestNewRedisAcceptsAValidURLWithoutDialling` (`redis_test.go:14`) proves only that
`NewRedis` returns no error, not the "without dialling" its name claims; and `keys_test.go:30` uses
`Asia/Ho_Chi_Minh`, which has had no DST since 1975, so it demonstrates non-UTC divergence but not a
real transition boundary. The one genuinely new gap — the vacuous gate test — is filed.

**Documentation accuracy.** CODEMAP is accurate throughout, and the sentence the first review
flagged as conditionally false ("no live services needed") is now true unconditionally. The one
inaccuracy left is `backend/.env.example`: it is the file CODEMAP and the Makefile tell you to copy,
it is named for the backend's environment, and it contains none of the three variables the binary
requires — `make run` after following it exactly still fails. Filed `low`.

**One rough edge I tried to make fail and could not.** `docker-compose.yml` declares healthchecks for
both services but nothing consumes them — no `depends_on: condition: service_healthy`, and `make up`
is `docker compose up -d` without `--wait` — so on paper the documented `make up && make test-integration`
loop races Postgres's first-run `initdb`. I probed it with a throwaway compose project on a fresh
volume and it passed: Go's compile step absorbs the startup time. Latent tidiness (`--wait` is a
one-word fix), not a defect I can evidence, so not filed. Related and also not filed: `.env.example`
asks the developer to keep `TEST_DATABASE_URL`'s port in step with `POSTGRES_PORT` by hand — the fix
plan considered deriving one from the other and rejected it with reasons, and the credentials are
project-specific enough that a stale port is very unlikely to land on someone else's database.

## Bugs filed

All medium or low. **None blocks the merge** — no `blocks:` is set on any of them.

- `harness/ideas/_inbox/integration-gate-tests-only-prove-the-skip-and-would-pass-if.md` — **medium**.
  Both gate tests assert only `t.Skipped()`, so an unconditionally-skipping helper would keep the
  suite green while the integration tests went silently dead. Needs the positive direction pinned.
- `harness/ideas/_inbox/gin-default-ships-debug-mode-and-all-proxies-trusted-to-prod.md` — **medium**.
  `gin.Default()` with no `SetMode`/`SetTrustedProxies` anywhere; debug logging and spoofable
  `ClientIP()` in the engine every later slice mounts on.
- `harness/ideas/_inbox/root-gitignore-env-silently-swallows-every-module-s-env-exam.md` — **medium**.
  The trap the executor flagged, confirmed: `git check-ignore -v frontend/.env.example` →
  `.gitignore:6:.env*`. Fix once at the root instead of a per-module negation.
- `harness/ideas/_inbox/cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md` — **low**.
  `defer pg.Close()` / `rdb.Close()` are unreachable; no signal handling, so Railway's `SIGTERM`
  cuts in-flight requests. Cheap now, expensive once slice 3 writes `daily_progress`.
- `harness/ideas/_inbox/backend-env-example-omits-the-app-s-own-database-url-redis-u.md` — **low**.
  The documented copy-and-run path does not work; also notes the doubled `config: config:` prefix.

## Verdict

**pass-with-bugs. The branch is mergeable.**

Both blockers are genuinely fixed and I proved each one in both directions rather than only the
direction that is easy to show: the destructive tests skip with the production variables exported,
and they really do run and pass against a nominated test database; the dev stack really does start
on overridden host ports and the binary boots and answers `200` against it. Nothing in the fixes was
papered over, both executor deviations were the right calls and were disclosed rather than hidden,
and the six pre-existing bugs are untouched, as expected. The five new findings are ordinary
quality bugs for the ideation queue — the foundation ones (Gin mode, graceful shutdown) are worth
taking early precisely because this slice is the template, but none of them is a reason to hold the
branch.

Not `pass`, only because bugs were filed. Not `fail`: the idea is delivered, no test fails, no
boundary is violated.

PR steps skipped — the plan has no `pr`. `gh` on this machine is authenticated as a work account
with no write access to this personal repo, so `gh pr comment` / `gh pr ready` do not apply. The
branch is pushed; a human with write access can open the PR by hand.

Human merge: `/harness merge harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
