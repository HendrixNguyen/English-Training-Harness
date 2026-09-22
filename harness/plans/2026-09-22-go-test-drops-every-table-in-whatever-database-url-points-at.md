---
idea: harness/ideas/_inbox/go-test-drops-every-table-in-whatever-database-url-points-at.md
status: done
priority: high
merged: false
amends: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
branch: harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001
worktree: .worktrees/store-go-module-postgres-and-redis-clients-migration-0001
---
# go test drops every table in whatever DATABASE_URL points at — Plan

**Idea:** `harness/ideas/_inbox/go-test-drops-every-table-in-whatever-database-url-points-at.md`
**Amends:** `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
**Goal:** Gate the store integration tests on test-only `TEST_DATABASE_URL` / `TEST_REDIS_URL` so that `go test ./...` can never drop tables in the production database `DATABASE_URL` points at, and make the CODEMAP's "no live services needed" claim true unconditionally.

**Branch:** work on the existing `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001` in `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001`. Do **not** create a new branch or worktree — this plan amends the branch under review. Every file below already exists on it.

**Run every command from `backend/`** inside that worktree unless the step says otherwise. `rg` is not installed — use `grep -n`.

**Approach:** one predicate change, in one place, plus a hard refusal inside `reset()` so no future caller can reach the drops by another path. `DATABASE_URL` / `REDIS_URL` keep their §8 production meaning and stop being test triggers entirely; `internal/config` is untouched.

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/store/integration_test.go` | gate on `TEST_DATABASE_URL` / `TEST_REDIS_URL`; `reset()` refuses without `TEST_DATABASE_URL` |
| `backend/internal/store/integration_gate_test.go` | **new** — unit test proving the gate skips, no database required |
| `backend/Makefile` | `test-integration` comment block names the test-only variables |
| `harness/CODEMAP.md` | `store` bullet: test commands and the now-unconditional `make test` claim |

---

## Tasks

### Task 1: Prove the gate before changing it (failing test first)

**Files:**
- Create: `backend/internal/store/integration_gate_test.go`

- [ ] **Step 1: Write the gating test**

It asserts the behaviour we want and needs no live services: with a production `DATABASE_URL`
exported and `TEST_DATABASE_URL` unset, `requirePostgres` must *skip*. `t.Skip` calls
`runtime.Goexit`, so the assertion is made from a deferred function inside a subtest, where
`t.Skipped()` already reflects the skip.

```go
package store

import "testing"

// TestRequirePostgresSkipsWithoutTestDatabaseURL pins the safety property that
// `go test ./...` must never run the destructive integration tests just because
// the production DATABASE_URL (spec §8) happens to be exported.
func TestRequirePostgresSkipsWithoutTestDatabaseURL(t *testing.T) {
	// A syntactically valid URL pointing nowhere: if the gate ever lets the test
	// through, NewPostgres is reached and this subtest fails instead of skipping.
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2")
	t.Setenv("TEST_DATABASE_URL", "")

	var skipped bool
	t.Run("gated", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		requirePostgres(t)
		t.Error("requirePostgres did not skip; it would have connected and dropped tables")
	})
	if !skipped {
		t.Fatal("requirePostgres must skip when TEST_DATABASE_URL is unset")
	}
}

// TestRequireRedisURLSkipsWithoutTestRedisURL is the same property for Redis.
func TestRequireRedisURLSkipsWithoutTestRedisURL(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://127.0.0.1:59999/0")
	t.Setenv("TEST_REDIS_URL", "")

	var skipped bool
	t.Run("gated", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		_ = requireRedisURL(t)
		t.Error("requireRedisURL did not skip")
	})
	if !skipped {
		t.Fatal("requireRedisURL must skip when TEST_REDIS_URL is unset")
	}
}
```

- [ ] **Step 2: Watch it fail for the right reason**

```sh
go test ./internal/store/ -count=1 -run 'SkipsWithout' 2>&1 | tail -n 20
```
Expected: a **compile** failure — `undefined: requireRedisURL` — because that helper does not exist
yet. That is the correct first failure; Task 2 introduces it. Do not commit yet.

**Commit:** none (this task's file is committed with Task 2, so the branch never carries a
non-compiling tree).

---

### Task 2: Gate on the test-only variables and make `reset()` unreachable without one

**Files:**
- Modify: `backend/internal/store/integration_test.go`

- [ ] **Step 1: Re-gate `requirePostgres`**

Replace lines 9-23 (the current `requirePostgres`, which reads `DATABASE_URL`) with:

```go
// requirePostgres skips the test unless the developer has explicitly nominated a
// disposable database in TEST_DATABASE_URL. It deliberately does NOT read
// DATABASE_URL: that is the production variable from spec §8, and these tests
// drop every table (see reset). A plain `go test ./...` must never be
// destructive, whatever the shell has exported.
func requirePostgres(t *testing.T) *Postgres {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is unset; these tests DROP every table, so they only run against a database you nominate (see backend/.env.example)")
	}
	pg, err := NewPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	return pg
}

// requireRedisURL skips the test unless TEST_REDIS_URL nominates a disposable
// Redis. Same reasoning as requirePostgres.
func requireRedisURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL is unset; run `make up` and export it to run integration tests")
	}
	return url
}
```

- [ ] **Step 2: Make `reset()` refuse without the nomination**

Insert at the top of `reset` (currently line 28, right after `t.Helper()`):

```go
	// Belt and braces: reset is the destructive step. Even if a future test
	// reaches it by another path, it must not run against a database the
	// developer did not nominate as disposable.
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Fatal("reset called without TEST_DATABASE_URL; refusing to drop tables")
	}
```

- [ ] **Step 3: Point the Redis test at the helper**

In `TestIntegrationRedisRoundTrip`, replace the inline four-line `os.Getenv("REDIS_URL")` /
`t.Skip` block (currently lines 105-108) with:

```go
	url := requireRedisURL(t)
```

- [ ] **Step 4: Verify no production variable name is left in the file**

```sh
grep -n 'DATABASE_URL\|REDIS_URL' internal/store/integration_test.go
```
Expected: only `TEST_DATABASE_URL` and `TEST_REDIS_URL` lines — no bare `os.Getenv("DATABASE_URL")`
and no bare `os.Getenv("REDIS_URL")`. (The comment mentioning `DATABASE_URL` by name is expected and
correct.)

```sh
go vet ./... && go test ./internal/store/ -count=1 -run 'SkipsWithout' -v
```
Expected: `VET_OK`-equivalent silence from vet, and both gate tests `--- PASS`.

**Commit:** `fix(store): gate integration tests on TEST_DATABASE_URL, not the production DATABASE_URL`

---

### Task 3: Update the Makefile and the CODEMAP sentence

**Files:**
- Modify: `backend/Makefile`
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1: Rewrite the `test-integration` comment block**

Replace `backend/Makefile` lines 18-20 (the `# Matches docker-compose.yml:` comment naming
`DATABASE_URL` / `REDIS_URL`) with:

```make
# Live-database tests. They DROP every table, so they are gated on test-only
# variables and never on the production DATABASE_URL / REDIS_URL of spec §8.
# Copy .env.example, then:
#   export TEST_DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
#   export TEST_REDIS_URL=redis://localhost:6379/0
# Without them every Integration test skips.
```

Leave the `test-integration` recipe itself (`go test ./... -count=1 -v -run Integration`) unchanged —
the gate now lives in the test file, which is the only place that can enforce it.

- [ ] **Step 2: Fix the stale CODEMAP sentence**

In `harness/CODEMAP.md` (same worktree), in the `store` bullet, replace:

> Tests: `cd backend && make test` (no live services needed); `make up` + exported
> `DATABASE_URL`/`REDIS_URL` then `make test-integration` for the live-database assertions, which
> skip otherwise.

with:

> Tests: `cd backend && make test` (never touches a live service — the integration tests are gated on
> `TEST_DATABASE_URL`/`TEST_REDIS_URL`, never on the production `DATABASE_URL`/`REDIS_URL`, because
> they DROP every table); `make up` + exported `TEST_DATABASE_URL`/`TEST_REDIS_URL` then
> `make test-integration` for the live-database assertions, which skip otherwise.

- [ ] **Step 3: Verify the claim is now unconditional**

```sh
grep -n 'TEST_DATABASE_URL' Makefile ../harness/CODEMAP.md
env -u TEST_DATABASE_URL -u TEST_REDIS_URL \
  DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
  REDIS_URL='redis://127.0.0.1:59999/0' \
  go test ./... -count=1; echo "exit=$?"
```
Expected: both files mention `TEST_DATABASE_URL`; `exit=0` with all packages `ok` — the production
variables no longer change what runs.

**Commit:** `docs(store): document the test-only database variables in Makefile and CODEMAP`

---

## Verification

Run from the worktree. The first block is the review's own reproduction, re-run: previously those
three tests *ran* and failed inside `reset()` at `integration_test.go:44`; they must now **skip**.

```sh
cd .worktrees/store-go-module-postgres-and-redis-clients-migration-0001/backend

# 1. The review's evidence, unchanged except that it must now skip.
env -u TEST_DATABASE_URL -u TEST_REDIS_URL \
  DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
  env -u REDIS_URL go test ./internal/store/ -count=1 -run Integration -v; echo "exit=$?"
```
Expected: `--- SKIP` for all three of
`TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent`,
`TestIntegrationPetStatesRejectsASecondRowForTheSameUser` and `TestIntegrationRedisRoundTrip`, each
printing the `TEST_DATABASE_URL is unset` / `TEST_REDIS_URL is unset` reason, `PASS`, `exit=0`.
**No `down migration:` line may appear** — that line is `reset()` executing, and its absence is the
proof. A `FAIL` at `integration_test.go` line ~44 means the fix did not take.

```sh
# 2. The gating unit tests pass on their own, with no database anywhere.
go test ./internal/store/ -count=1 -run 'SkipsWithout' -v; echo "exit=$?"

# 3. The whole default suite is green and non-destructive with production vars set.
env -u TEST_DATABASE_URL -u TEST_REDIS_URL \
  DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
  REDIS_URL='redis://127.0.0.1:59999/0' \
  make test; echo "exit=$?"

# 4. Build and vet still clean.
go build ./... && go vet ./... && echo BUILD_VET_OK

# 5. No production variable can reach the drops.
grep -n 'os.Getenv' internal/store/integration_test.go
grep -n 'TEST_DATABASE_URL' internal/store/integration_test.go Makefile ../harness/CODEMAP.md
```
Expected: (2) both gate tests `PASS`, `exit=0`. (3) all packages `ok`, `exit=0`. (4) `BUILD_VET_OK`.
(5) every `os.Getenv` in `integration_test.go` reads a `TEST_`-prefixed name — three of them:
`requirePostgres`, `reset`'s refusal, `requireRedisURL`; and all three files mention
`TEST_DATABASE_URL`.

```sh
# 6. Harness artifacts still valid (from the repo root).
cd /Users/hendrixnguyen/Workspaces/self/Learning-English-Project
python3 tools/harness/cli.py validate; echo "exit=$?"
git -C .worktrees/store-go-module-postgres-and-redis-clients-migration-0001 status --short
```
Expected: `exit=0`; clean worktree after the two commits.

## Notes

- **No live database is needed to verify this plan.** Do not start containers: host port 6379 is held
  by the owner's unrelated `scio3-redis-1`, which must not be disturbed. The sibling blocker
  `harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md` makes
  that port overridable and adds `backend/.env.example` carrying the matching `TEST_DATABASE_URL` /
  `TEST_REDIS_URL` lines. The two plans touch disjoint files (`Makefile` excepted — different comment
  blocks, no conflict) and can land in either order; the `.env.example` reference in the skip message
  is a documentation pointer, not a dependency.
- `internal/config` is **not** touched. `DATABASE_URL` / `REDIS_URL` keep their §8 production meaning
  for `cmd/api`; they simply stop being test triggers.
- Scope guard: no new endpoints, no changes to migration SQL, no `//go:build` tag (rejected in the
  idea's `## Evaluation` — it moves the hazard rather than removing it).

---

## Execution summary

**Built on the existing branch** `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001`
in `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001` (amending plan — no new
branch, no new worktree, per the skill's step 5).

Commits:

```
dd868b3 fix(store): gate integration tests on TEST_DATABASE_URL, not the production DATABASE_URL
c400e5f docs(store): document the test-only database variables in Makefile and CODEMAP
```

**Deviations:** none. All three tasks were implemented exactly as written. Task 1's file was
committed together with Task 2 as the plan instructs, so the branch never carried a non-compiling
tree. Task 1 Step 2 produced the expected first failure:

```
# .../internal/store [.../internal/store.test]
internal/store/integration_gate_test.go:33:7: undefined: requireRedisURL
FAIL	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store [build failed]
```

### Verification

**1. The review's own reproduction — previously these three *ran* and failed inside `reset()`; they now skip.**

```
$ env -u TEST_DATABASE_URL -u TEST_REDIS_URL \
    DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
    env -u REDIS_URL go test ./internal/store/ -count=1 -run Integration -v
=== RUN   TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
    integration_test.go:63: TEST_DATABASE_URL is unset; these tests DROP every table, so they only run against a database you nominate (see backend/.env.example)
--- SKIP: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.00s)
=== RUN   TestIntegrationPetStatesRejectsASecondRowForTheSameUser
    integration_test.go:99: TEST_DATABASE_URL is unset; these tests DROP every table, so they only run against a database you nominate (see backend/.env.example)
--- SKIP: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.00s)
=== RUN   TestIntegrationRedisRoundTrip
    integration_test.go:125: TEST_REDIS_URL is unset; run `make up` and export it to run integration tests
--- SKIP: TestIntegrationRedisRoundTrip (0.00s)
PASS
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store	0.319s
exit=0
```

No `down migration:` line appears — `reset()` never executed. That absence is the proof.

**2. Gate unit tests, no database anywhere.**

```
$ go test ./internal/store/ -count=1 -run 'SkipsWithout' -v
--- PASS: TestRequirePostgresSkipsWithoutTestDatabaseURL (0.00s)
    --- SKIP: TestRequirePostgresSkipsWithoutTestDatabaseURL/gated (0.00s)
--- PASS: TestRequireRedisURLSkipsWithoutTestRedisURL (0.00s)
    --- SKIP: TestRequireRedisURLSkipsWithoutTestRedisURL/gated (0.00s)
PASS
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store	0.338s
exit=0
```

**3. Whole default suite, green and non-destructive with the production variables set.**

```
$ env -u TEST_DATABASE_URL -u TEST_REDIS_URL \
    DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
    REDIS_URL='redis://127.0.0.1:59999/0' make test
go test ./...
?   	.../cmd/api	[no test files]
ok  	.../internal/config	0.240s
ok  	.../internal/health	(cached)
ok  	.../internal/store	0.351s
exit=0
```

**4. Build and vet.**

```
$ go build ./... && go vet ./... && echo BUILD_VET_OK
BUILD_VET_OK
```

**5. No production variable can reach the drops.**

```
$ grep -n 'os.Getenv' internal/store/integration_test.go
16:	url := os.Getenv("TEST_DATABASE_URL")
32:	url := os.Getenv("TEST_REDIS_URL")
46:	if os.Getenv("TEST_DATABASE_URL") == "" {
```

All three reads are `TEST_`-prefixed. `TEST_DATABASE_URL` is present in
`internal/store/integration_test.go` (lines 10, 16, 18, 46, 47), `Makefile` (line 21) and
`../harness/CODEMAP.md` (line 7).

**6. Harness artifacts.**

```
$ python3 tools/harness/cli.py validate; echo "exit=$?"
exit=0
$ git -C .worktrees/store-go-module-postgres-and-redis-clients-migration-0001 status --short
(clean)
```

### Runtime proof

**a. It builds**, from a shell with every service variable unset:

```
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL -u PORT go build ./...
BUILD_OK
```

**b. The whole suite, clean shell** — not only this plan's tests:

```
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL -u PORT go test ./... -count=1
?   	.../cmd/api	[no test files]
ok  	.../internal/config	0.437s
ok  	.../internal/health	0.564s
ok  	.../internal/store	0.792s
exit=0

$ python3 -m unittest discover -s tools/harness/tests   # the CODEMAP's other documented suite
Ran 27 tests in 0.073s
OK
exit=0
```

**c. It boots and answers.** `cmd/api` connects to Postgres and Redis before serving, so the
end-to-end boot needs the dev stack, which could not be started until the sibling blocker
`2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md` landed (host 6379 is
held by an unrelated container). The boot + `curl /healthz` proof for this branch was therefore run
immediately after that plan landed on the same branch and is recorded in its *Runtime proof*; the
store gate introduced here was re-verified in that same run (integration tests execute only when
`TEST_DATABASE_URL`/`TEST_REDIS_URL` are exported, and skip otherwise). The safe failure mode is
proved below.

**d. Every documented command, exactly as documented, in a clean environment.**

```
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make test
go test ./...   → all ok, exit=0

$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make test-integration
--- SKIP: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.01s)
--- SKIP: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.00s)
--- SKIP: TestIntegrationRedisRoundTrip (0.00s)
ok  	.../internal/store	0.508s

# the destructive-sounding one, with the PRODUCTION variables exported — must still refuse:
$ env -u TEST_DATABASE_URL -u TEST_REDIS_URL \
    DATABASE_URL='postgres://u:p@127.0.0.1:59999/db?sslmode=disable&connect_timeout=2' \
    REDIS_URL='redis://127.0.0.1:59999/0' make test-integration
--- SKIP: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.00s)
--- SKIP: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.00s)
--- SKIP: TestIntegrationRedisRoundTrip (0.00s)
ok  	.../internal/store	0.524s
exit=0

$ make tidy
go mod tidy          # worktree still clean afterwards

$ env -u DATABASE_URL -u REDIS_URL make run
2026/09/22 16:52:43 config: config: DATABASE_URL is required
exit status 1        # safe refusal, no partial start
```

**e. The commands are safe to run.** This is the point of the plan: with `DATABASE_URL` and
`REDIS_URL` exported to a real database, `make test`, `make test-integration` and `go test ./...` all
skip the destructive tests and `reset()` is never entered. `make up` / `make down` are covered by the
sibling blocker's plan on this same branch.

### Push / PR

Branch pushed:

```
$ git push -u origin harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001
   4690cd2..5117223  harness/...-migration-0001 -> harness/...-migration-0001
```

No PR was opened. This is an **amending** plan, so the skill's step 5 already says to skip
`gh pr create` — the branch's own plan owns the PR. The single attempt made for the record failed as
expected, because `gh` is authenticated as a work account with no write access to this repository:

```
$ gh pr create --draft --base main --head harness/...-migration-0001 ...
pull request create failed: GraphQL: must be a collaborator (createPullRequest)
```

`gh auth status` → logged in as `hendrixnguyen-optisigns`. A human with write access must open or
update the PR for this branch.
