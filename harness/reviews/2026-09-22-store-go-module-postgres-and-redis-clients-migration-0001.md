---
plan: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/go-test-drops-every-table-in-whatever-database-url-points-at.md, harness/ideas/_inbox/docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md, harness/ideas/_inbox/healthz-leaks-postgres-and-redis-driver-error-strings-public.md, harness/ideas/_inbox/migrations-run-on-every-boot-with-no-advisory-lock.md, harness/ideas/_inbox/no-ci-runs-the-integration-suite-so-pgmigrator-is-never-veri.md, harness/ideas/_inbox/migrate-is-only-tested-against-the-single-embedded-migration.md, harness/ideas/_inbox/healthz-shares-one-2s-deadline-across-two-sequential-pings.md, harness/ideas/_inbox/spec-3-2-leaves-user-id-nullable-on-three-child-tables.md]
---
# Review — Store: Go module, Postgres and Redis clients, migration 0001

**Plan:** `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
**Branch/worktree:** `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001` / `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001`
**Diff:** `git diff main...harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001 --stat`

## Plan vs idea
The idea's *Expected output* is nine bullets. Checked one by one against the worktree:

- **`backend/go.mod` + `cmd/api/main.go` booting Gin with one route `GET /healthz` that pings both services and returns 200/503, no other endpoints** — delivered. Module `github.com/HendrixNguyen/English-Training-Harness/backend`; `cmd/api/main.go:44-45` mounts exactly one route; `internal/health/health.go:23-45` returns 200 or 503. `grep -n 'r\.\(GET\|POST\|PUT\|DELETE\|Group\)' cmd/api/main.go` shows one line.
- **`store.Postgres` (pgx pool from `DATABASE_URL`) and `store.Redis` (go-redis from `REDIS_URL`) constructors, plus a config loader for the §8 vars** — delivered: `postgres.go:24`, `redis.go:18`, `config/config.go:18-34`. `Load()` requires both URLs and defaults `PORT` to 8080.
- **`store.Migrate(ctx)` and `0001_init.up.sql` / `.down.sql` with the §3.2 DDL** — delivered. Signature is `Migrate(ctx, m Migrator, fsys fs.FS)` rather than the idea's shorthand `Migrate(ctx)`; that is the interface seam the evaluator's test strategy asked for, so it is the right deviation.
- **The §3.2 DDL itself: 3 types, 6 tables, `pet_states.user_id UNIQUE NOT NULL`, `UNIQUE(user_id, date)`** — delivered *verbatim*. I re-ran the executor's own spec diff rather than trusting it: exit 0, all six tables and three types match the spec line for line (output below). `pet_stage` carries all six values including `wilted`.
- **Redis key builders with the §4 TTLs as constants** — delivered: all five keys and the four TTLs in `keys.go`, matching spec lines 262-266 (`sess:{user_id}:token` 24h, `quiz:placement:{user_id}` 2h, `daily:accumulated:{user_id}:{YYYY-MM-DD}` 48h, `queue:webpush:delay` no TTL, `ratelimit:ai:{user_id}` 1m).
- **`Makefile` with `make test` = `go test ./...`; tests cover migration-applies-and-is-idempotent, `pet_states` rejects a second row, key builders produce the exact §4 strings** — delivered, with a caveat: the first two are covered *only* by the skippable integration file, so the default run proves neither. See *Quality*.
- **Integration tests skip (not fail) without `DATABASE_URL`/`REDIS_URL`; `docker-compose.yml` provides local Postgres + Redis** — delivered as written, and both halves carry bugs (`…drops-every-table…`, `…hard-codes-host-ports…`).
- **CODEMAP `store` paragraph with the module path and how to run migrations** — delivered and accurate (see *Quality*).
- **"Tables: all six. Redis keys: all five. Screens: none."** — matches.

The plan delivered the idea. Where the plan went beyond the idea it was to the good (the `Migrator`/`Pinger` seams). The defects filed below are in the plan's own text, not in the executor's reading of it.

## Code vs plan
10 commits on `harness/2026-09-22-high-…`, one per task, every one carrying the `Co-Authored-By: Claude Opus 5 (1M context)` trailer (checked with `git log --format='%(trailers:key=Co-Authored-By)'`). 22 files, +1167/-2. Working tree clean.

| Task | Verdict |
| --- | --- |
| 1 Module skeleton + config loader | followed (except the `go` directive, below) |
| 2 Redis key builders + TTLs | followed, byte-identical to the plan |
| 3 Migration 0001 SQL | followed, DDL verbatim from §3.2 |
| 4 Migration runner over `Migrator` | followed |
| 5 Postgres client + `PgMigrator` | followed |
| 6 Redis client | followed |
| 7 `/healthz` handler | followed |
| 8 `cmd/api/main.go` wiring | followed, byte-identical to the plan's code block |
| 9 docker-compose + integration tests | followed, byte-identical to the plan's code blocks |
| 10 CODEMAP | followed |

I diffed the plan's embedded code blocks against the committed files for Tasks 7, 8 and 9 and they match character for character. The executor's claim "all SQL, Go source and test files are byte-for-byte the plan's" holds where I checked it.

**Deviation 1 — `go.mod` says `go 1.25.0`, not the plan's `go 1.22`.** Justified, and the right call. Task 1 Step 1 says `go mod edit -go=1.22`; Task 5 Step 1 says `go get github.com/jackc/pgx/v5@latest`. Those two instructions are in conflict — pgx v5.11.0 declares `go 1.25`, so the toolchain raised the floor. Pinning an older pgx instead would have substituted an approach the plan did not authorise, and silently downgrading a dependency to satisfy a stale version literal is worse than raising the floor. The executor flagged it rather than hiding it. **Does it matter for Railway (§8)?** No. §8 deploys "compiled Go binary as lightweight Docker container", so the toolchain comes from the build image, and any current `golang:` image satisfies ≥1.25. (If the build ever falls back to Nixpacks, Nixpacks reads the same `go` directive and provisions 1.25 — still fine.) The real residue is local: every contributor now needs Go ≥1.25, and nothing in the repo pins a toolchain, because there is no Dockerfile and no CI yet. That is slice-2 scope, not a defect here. **Is the CODEMAP now accurate?** Yes — the bullet says "Go 1.25", which matches `go.mod:3`. Writing "Go 1.22" there to match the plan would have made the map false; the executor chose correctly.

**Deviation 2 — the five-key grep returns 6 lines, not 5.** Correct and harmless: the sixth is the comment on `keys.go:10` that names `queue:webpush:delay`. All five formats appear exactly once in code. The plan's expected count was simply wrong.

**Deviation 3 — the live-stack run used host port 6380 for Redis.** Forced: `make up` cannot bind 6379 on this machine. The workaround stayed outside the repo and the compose file was committed as the plan specifies. That is the correct executor behaviour, and it is also the evidence for bug `…hard-codes-host-ports…`.

**Deviation 4 — no PR.** `gh` is authenticated as the work account `hendrixnguyen-optisigns`, which has no write access to `HendrixNguyen/English-Training-Harness`; `gh pr create` returned `must be a collaborator (createPullRequest)`. The plan frontmatter has no `pr`, so this review skips step 8 (PR comment, `gh pr ready`) entirely. The branch is pushed.

### Verification re-run by the reviewer

All of it re-run in the worktree, unedited:

```
$ cd backend && go build ./... && go vet ./...
BUILD_OK
VET_OK
(no other output)

$ env -u DATABASE_URL -u REDIS_URL go test ./... -count=1
?   	github.com/HendrixNguyen/English-Training-Harness/backend/cmd/api	[no test files]
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/config	0.376s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/health	0.719s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store	1.172s
test exit=0

$ env -u DATABASE_URL -u REDIS_URL make test
go test ./...
?   	github.com/HendrixNguyen/English-Training-Harness/backend/cmd/api	[no test files]
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/config	(cached)
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/health	(cached)
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store	(cached)

$ grep -c 'CREATE TABLE' internal/store/migrations/0001_init.up.sql
6
$ grep -c 'CREATE TYPE' internal/store/migrations/0001_init.up.sql
3
$ grep -n 'user_id UUID UNIQUE NOT NULL' internal/store/migrations/0001_init.up.sql
35:    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,

$ grep -n 'sess:%s:token\|quiz:placement:%s\|daily:accumulated:%s:%s\|ratelimit:ai:%s\|queue:webpush:delay' internal/store/keys.go
10:// TTLs from spec §4. queue:webpush:delay is persistent and has no TTL.
19:const WebPushDelayQueueKey = "queue:webpush:delay"
22:func SessionKey(userID string) string { return fmt.Sprintf("sess:%s:token", userID) }
25:func PlacementQuizKey(userID string) string { return fmt.Sprintf("quiz:placement:%s", userID) }
31:	return fmt.Sprintf("daily:accumulated:%s:%s", userID, day.Format("2006-01-02"))
35:func AIRateLimitKey(userID string) string { return fmt.Sprintf("ratelimit:ai:%s", userID) }

# §3.2 DDL comparison, re-run rather than trusted:
$ diff <(sed -n '144,256p' 1st-thinking-architecture-doc.md | tr -d '\\' | grep -v '^$' | grep -v '^## ') \
       <(grep -v '^--' backend/internal/store/migrations/0001_init.up.sql | grep -v '^$')
diff exit=0

$ git log --oneline main..HEAD | wc -l
      10
$ git status --short
(clean)

$ python3 tools/harness/cli.py validate; echo exit=$?      # from ROOT
exit=0
```

Docker containers were deliberately **not** started: host port 6379 is held by the owner's unrelated `scio3-redis-1`, and `docker-compose.yml` insists on that port. The compose file was judged by reading. That constraint is itself the subject of a filed bug.

## Quality

**Migration 0001 vs §3.2 (review focus 3).** Faithful. The diff above is byte-exact once the spec's rich-text backslash escaping is stripped: three enums with the right members (`pet_stage` includes `wilted`), six tables in spec order, every default, every `CHECK`, `UNIQUE(user_id, date)`, and the merged `pet_states.user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE`. The header comment correctly notes that `gen_random_uuid()` is core from PG 13 and that compose pins PG 16.

**Does `0001_init.down.sql` actually reverse the up?** Yes. Six `DROP TABLE IF EXISTS` in reverse dependency order (`exercises` before `roadmaps`; all children before `users`), then the three `DROP TYPE IF EXISTS` after the tables that use those types are gone — which is the ordering that matters, since dropping `cefr_level` while `users.cefr_current` exists would fail. It does not drop `schema_migrations`, which is right: the bookkeeping table is the runner's, not 0001's. Two caveats worth knowing: nothing in production code ever applies a `.down.sql` (there is no down runner in this slice — the file's only consumer is the integration test's `reset()`), and the down test asserts substring presence rather than order for the `DROP TYPE` block, so a reordering regression would only be caught by the skippable integration test. Filed under `…only-tested-against-the-single-embedded-migration…`.

**Boundaries (review focus 4).** They hold where the CODEMAP rule bites today. `internal/health` never imports `internal/store`: it declares its own one-method `Pinger` and `main.go` passes `*store.Postgres` and `*store.Redis` into it — consumer-side interface, defined where it is used, exactly the Go idiom and exactly what "packages talk via interfaces" means. `store.Migrator` gives the migration runner the same seam, with a compile-time `var _ Migrator = (*PgMigrator)(nil)` assertion. `cmd/api/main.go` is wiring only, 51 lines, no logic.

One forward-looking weakness, not a violation today and not filed as a bug: `store.Postgres.Pool` and `store.Redis.Client` are exported fields, so any future package can reach straight through `store` to raw SQL. That is deliberate — later slices are written in raw SQL and need query access — but it means CODEMAP's "no cross-package table access" rule has no enforcement point in code. When slice 2 lands, the reviewer should check that `auth` owns its own queries inside `store` or an `auth` repository, rather than doing `pg.Pool.Query` from a handler.

**Test strategy (review focus 4): transparent, but not yet honest on its own.** The skips are genuine skips with a reason printed, not silent passes, and the plan's claim that `go test ./...` needs no live services is true when the variables are unset. But measured coverage on the default run shows what the strategy costs: `internal/store` 45.7%, `cmd/api` 0%, and every function that actually speaks a wire protocol at 0% — `PgMigrator.EnsureVersionTable/AppliedVersions/Apply`, `Postgres.Ping/Close/Migrator`, `Redis.Ping`. `NewPostgres` is 42.9% (only the parse-failure branch). Nothing in the repo ever runs the other half: there is no CI configuration anywhere, and `make up && export … && make test-integration` is a manual step no document lists as required. The unit tests that stand in for the migration are `strings.Contains` checks against the embedded SQL text (`migrations_test.go:21-86`, `postgres_test.go:15-21`) — they never execute the DDL, so invalid Postgres syntax or an inverted `CHECK` would pass them. Filed as `…no-ci-runs-the-integration-suite…` and `…only-tested-against-the-single-embedded-migration…`.

Other real gaps, all filed: four of `Migrate`'s five error branches are dead (`migrations.go:35-36, 39-40, 44-45, 56-57`); every `Migrate` test uses the real one-file `MigrationsFS`, so ordering and "apply the pending one, skip the applied one" are untested; `health`'s 2s deadline is unexercised because the fake ignores its context, and the both-dependencies-down case has no test. Not flaky: no `t.Parallel`, `t.Setenv` used correctly and completely in `config_test.go`, the one time-based test uses a fixed `time.Date`.

**The test file is also a live hazard.** `integration_test.go` is gated only on `DATABASE_URL` — the production variable name from §8 — and its first action is `reset()`, which runs the down migration and drops `schema_migrations`. So `go test ./...` in a shell where `DATABASE_URL` is exported destroys that database's schema, and CODEMAP tells the reader to export exactly that variable one sentence before describing `make test` as needing "no live services". I reproduced the gating without a database: with `DATABASE_URL` set to an unreachable host the tests run rather than skip, failing inside `reset()` at `integration_test.go:44` — against a reachable database that line succeeds. Filed `priority: high`.

**Migrations on every boot (review focus 5): idempotent, not concurrency-safe.** For one instance it is correct — `Migrate` reads `schema_migrations`, skips applied versions, and `PgMigrator.Apply` wraps DDL plus the version row in one transaction, so a failure leaves neither a half-applied schema nor a version row (Postgres DDL is transactional; pgx v5 sends the multi-statement body over the simple protocol because `Exec` gets no arguments, which the executor's live run confirmed). Re-running applies nothing. What is missing is a lock: `EnsureVersionTable` → `AppliedVersions` → `Apply` is unserialised, so two instances booting together both try to apply `0001_init`, the loser hits a duplicate-object error, and `main.go:38` turns that into `log.Fatalf` and a crash loop. `pg_advisory_lock` is the dependency-free fix. Filed medium. Related: `main.go` migrates on a deadline-free `context.Background()`.

**Security.** `/healthz` is unauthenticated, as it must be, and copies raw driver errors into the response body (`health.go:32`, `:36`). pgx errors embed `user=…`, `database=…`, host and port — that shape is visible in the reproduction output above. Filed medium.

**CODEMAP.** Correct and genuinely useful: module path, stack ("Go 1.25" matching `go.mod:3`), the exact `store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)` call, `schema_migrations`, "run on every boot", the §3.2/`pet_states` facts, "use the key builders, never literal key strings", and both test commands. The heading change from "Planned backend packages — none exist yet" to "Backend packages" is right now that they exist. **No reviewer correction was needed, so step 7 was skipped** — with one caveat I did not edit into it, because the sentence becomes true again once the high bug is fixed: `make test` is described as "no live services needed", which is false while `DATABASE_URL` is exported. The fix belongs with the bug, not with this review.

## Bugs filed
Two of these gate the merge.

- `harness/ideas/_inbox/go-test-drops-every-table-in-whatever-database-url-points-at.md` — **high**. `go test ./...` runs the integration tests whenever `DATABASE_URL` is set, and their first action drops every table plus `schema_migrations`. CODEMAP instructs exporting that variable. Fix: gate on `TEST_DATABASE_URL` or a `//go:build integration` tag.
- `harness/ideas/_inbox/docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md` — **high**. `"5432:5432"` / `"6379:6379"` are unoverridable, so `make up` fails on the owner's machine (6379 held by `scio3-redis-1`). The executor hit this, worked around it outside the repo, and shipped the file unchanged. Fix: `"${REDIS_PORT:-6379}:6379"` etc. Secondary: no named volume, so the dev database is discarded on every `down`/`up`.
- `harness/ideas/_inbox/healthz-leaks-postgres-and-redis-driver-error-strings-public.md` — **medium**. Unauthenticated `/healthz` returns `err.Error()`, which embeds the database user, database name, host and port.
- `harness/ideas/_inbox/migrations-run-on-every-boot-with-no-advisory-lock.md` — **medium**. Concurrent boots race on first migration; the loser `log.Fatalf`s into a crash loop.
- `harness/ideas/_inbox/no-ci-runs-the-integration-suite-so-pgmigrator-is-never-veri.md` — **medium**. No CI anywhere, so the skipped half never runs and `PgMigrator` and `Redis.Ping` ship at 0% automated coverage.
- `harness/ideas/_inbox/migrate-is-only-tested-against-the-single-embedded-migration.md` — **medium**. Ordering, apply-one-skip-one, four of five error branches and the partial-`applied` return are all untested; `Migrate` already takes `fs.FS`, so `fstest.MapFS` covers them without a database.
- `harness/ideas/_inbox/healthz-shares-one-2s-deadline-across-two-sequential-pings.md` — **low**. A slow Postgres can consume the whole budget and make a healthy Redis report `context deadline exceeded`; the deadline and the both-down case are untested.
- `harness/ideas/_inbox/spec-3-2-leaves-user-id-nullable-on-three-child-tables.md` — **low**. Spec defect surfaced by the faithful transcription: `push_subscriptions`/`daily_progress`/`roadmaps` `user_id` are nullable, which also defeats `UNIQUE(user_id, date)`. Belongs in migration 0002, never an edit to 0001.

## Verdict
**pass-with-bugs.** The idea is delivered in full, the plan was followed task for task with four deviations that are all justified and all disclosed, the §3.2 DDL is verbatim, build/vet/tests are green and I reproduced every verification command myself.

I did not call this `fail`: none of the role's three fail conditions hold — the idea is delivered, no test fails, and no CODEMAP boundary is violated. But two of the eight bugs are `priority: high`, and both originate in the plan's own text rather than in the execution, so **do not merge until they are fixed**: `go test ./...` will destroy any database `DATABASE_URL` points at, and `make up` — the local dev stack the owner explicitly wants working — cannot start on the owner's machine. Both are small, self-contained fixes.

PR steps skipped: the plan has no `pr`. `gh` on this machine is authenticated as a work account with no write access to this personal repo, so step 8 (`gh pr comment`, `gh pr ready`) does not apply. The branch is pushed; a PR can be opened by hand.

Human merge: `/harness merge harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
