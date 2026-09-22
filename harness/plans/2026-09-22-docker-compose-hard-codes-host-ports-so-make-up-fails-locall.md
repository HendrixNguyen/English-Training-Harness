---
idea: harness/ideas/_inbox/docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md
status: done
priority: high
merged: false
amends: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
branch: harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001
worktree: .worktrees/store-go-module-postgres-and-redis-clients-migration-0001
---
# docker-compose hard-codes host ports so make up fails locally — Plan

**Idea:** `harness/ideas/_inbox/docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md`
**Amends:** `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
**Goal:** Make the dev stack's host ports overridable through `${POSTGRES_PORT:-5432}` / `${REDIS_PORT:-6379}` with a committed `backend/.env.example`, and give Postgres a named volume so `make down && make up` no longer discards the dev database.

**Branch:** work on the existing `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001` in `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001`. Do **not** create a new branch or worktree — this plan amends the branch under review. Every file below already exists on it except the one new `backend/.env.example`, which the idea's *Expected output* requires.

**Run every command from `backend/`** inside that worktree unless the step says otherwise. `rg` is not installed — use `grep -n`.

**Approach:** use Compose's own mechanism rather than documenting a workaround. Compose reads a `.env` beside the compose file automatically and interpolates `${VAR:-default}` in `ports`, so a developer sets `REDIS_PORT=6380` once in an untracked `.env` and `make up` works, while a clean machine keeps the standard ports with no configuration at all.

**Do not start containers while verifying.** Host port 6379 is held by the owner's unrelated `scio3-redis-1` container, which must keep running. `docker compose config` renders and validates the file without touching Docker's daemon state, and that is what this plan verifies with.

## File structure

| Path | Change |
| --- | --- |
| `backend/docker-compose.yml` | parameterised host ports; named `postgres_data` volume |
| `backend/.env.example` | **new** — documents `POSTGRES_PORT` / `REDIS_PORT` and the matching `TEST_*` URLs |
| `backend/.gitignore` | ignore the real `.env` |
| `backend/Makefile` | `up` comment explaining the override |

---

## Tasks

### Task 1: Parameterise the host ports and persist Postgres data

**Files:**
- Modify: `backend/docker-compose.yml`

- [ ] **Step 1: Make both host ports overridable**

Line 11: `      - "5432:5432"` → `      - "${POSTGRES_PORT:-5432}:5432"`
Line 21: `      - "6379:6379"` → `      - "${REDIS_PORT:-6379}:6379"`

Only the **host** side is parameterised. The container side stays fixed, so the URLs inside the
compose network and the images' own defaults are unaffected.

- [ ] **Step 2: Add the named volume**

Add to the `postgres` service, after its `environment:` block:

```yaml
    volumes:
      - postgres_data:/var/lib/postgresql/data
```

and at the end of the file, at top level (column 0, a sibling of `services:`):

```yaml

# Named so `make down` keeps the dev database. `docker compose down -v` discards it.
volumes:
  postgres_data:
```

Redis gets no volume: it holds sessions and rate-limit counters (spec §4), all of which are
TTL-bounded and disposable by design.

- [ ] **Step 3: Extend the header comment**

Append to the existing two-line header comment at the top of the file:

```yaml
# Host ports are overridable: copy .env.example to .env and set POSTGRES_PORT /
# REDIS_PORT if something else on this machine already holds 5432 or 6379.
```

- [ ] **Step 4: Verify it parses, both ways, without starting anything**

```sh
docker compose config | grep -n 'published\|postgres_data'
REDIS_PORT=6380 POSTGRES_PORT=5433 docker compose config | grep -n 'published'
```
Expected: the first shows `published: "5432"` and `published: "6379"` (defaults applied) plus the
`postgres_data` volume in both the service mount and the top-level `volumes:` map; the second shows
`published: "5433"` and `published: "6380"`. No container is created by either command.

**Commit:** `fix(backend): make dev stack host ports overridable and persist postgres data`

---

### Task 2: Commit `.env.example`, ignore the real `.env`, and note the override in the Makefile

**Files:**
- Create: `backend/.env.example`
- Modify: `backend/.gitignore`
- Modify: `backend/Makefile`

- [ ] **Step 1: Write `backend/.env.example`**

```sh
# Copy to .env (untracked). Docker Compose reads .env from this directory
# automatically, so `make up` picks these up with no extra flags.

# Host ports for the dev stack. Change either one if something else on this
# machine already holds it (e.g. another project's Redis on 6379).
POSTGRES_PORT=5432
REDIS_PORT=6379

# Test-only database URLs. The store integration tests DROP every table, so they
# are gated on these and never on the production DATABASE_URL / REDIS_URL of
# spec §8. Keep the port here in step with the two above.
TEST_DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
TEST_REDIS_URL=redis://localhost:6379/0
```

- [ ] **Step 2: Keep the real `.env` out of git**

Append to `backend/.gitignore` (currently the single line `/api`):

```
.env
```

`.env.example` is unaffected by that pattern and stays tracked — Step 4 proves it.

- [ ] **Step 3: Note the override in the Makefile**

Replace the `up:` target (lines 12-13) with:

```make
# Ports come from .env (copy .env.example). Set REDIS_PORT / POSTGRES_PORT there
# if 6379 or 5432 is already taken on this machine; `down` keeps the postgres
# volume, `docker compose down -v` discards it.
up:
	docker compose up -d
```

Leave `down:` as it is — plain `docker compose down` now preserves `postgres_data`, which is the
point of Task 1 Step 2.

- [ ] **Step 4: Verify the tracking and the wiring**

```sh
git check-ignore -v .env || echo ".env NOT ignored"
git check-ignore -v .env.example || echo ".env.example tracked-eligible (expected)"
git add .env.example .gitignore Makefile && git status --short
```
Expected: `.env` matches the new `.gitignore` rule; `.env.example` prints the "tracked-eligible"
fallback line (it is *not* ignored) and appears as `A` in `git status --short`.

**Commit:** `docs(backend): add .env.example for dev stack ports and test database URLs`

---

## Verification

Run from the worktree. This re-runs the review's evidence: the review could not start the stack at
all because 6379 is held by `scio3-redis-1`, and judged the compose file by reading only. The file
must now render with an overridden port.

```sh
cd .worktrees/store-go-module-postgres-and-redis-clients-migration-0001/backend

# 1. The override the executor needed, now expressible in the repo. No containers started.
REDIS_PORT=6380 docker compose config | grep -n 'published'; echo "exit=$?"
```
Expected: `published: "6380"` for redis and `published: "5432"` for postgres. This is the exact
configuration the executor had to keep in a throwaway file outside the repo, and the exact reason the
reviewer skipped the live stack.

```sh
# 2. Defaults still apply on a clean machine.
env -u REDIS_PORT -u POSTGRES_PORT docker compose config | grep -n 'published'

# 3. Both ports override independently.
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose config | grep -n 'published'

# 4. The file is valid YAML/Compose and the volume is declared.
docker compose config --quiet; echo "config exit=$?"
docker compose config --volumes
docker compose config | grep -n 'postgres_data'

# 5. Nothing was started, and the owner's unrelated container is untouched.
docker ps --format '{{.Names}}\t{{.Ports}}' | grep -n 'scio3-redis-1\|backend-'
```
Expected: (2) `5432` and `6379`. (3) `5433` and `6380`. (4) `config exit=0`; `--volumes` prints
`postgres_data`; the mount and the top-level key both appear. (5) `scio3-redis-1` still running on
6379, and **no** `backend-postgres-1` / `backend-redis-1` — this plan never starts a container.

```sh
# 6. Docs and ignores are right.
grep -n 'POSTGRES_PORT\|REDIS_PORT' docker-compose.yml Makefile .env.example
git check-ignore -v .env
git ls-files .env.example

# 7. Go side untouched and still green.
env -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1; echo "exit=$?"

# 8. Harness artifacts still valid (from the repo root).
cd /Users/hendrixnguyen/Workspaces/self/Learning-English-Project
python3 tools/harness/cli.py validate; echo "exit=$?"
git -C .worktrees/store-go-module-postgres-and-redis-clients-migration-0001 status --short
```
Expected: (6) all three variables present in the compose file, the Makefile comment and the example;
`.env` ignored; `.env.example` listed by `git ls-files`. (7) all packages `ok`, `exit=0`. (8)
`exit=0`; clean worktree after the two commits.

## Notes

- **`backend/.env.example` is the one new file** in this plan; everything else edits a file that
  already exists on the branch. The idea's *Expected output* names it explicitly and there is nowhere
  else to put the port documentation that Compose will also read.
- The `TEST_DATABASE_URL` / `TEST_REDIS_URL` lines in `.env.example` are the variables introduced by
  the sibling blocker
  `harness/plans/2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md`. The two
  plans touch disjoint files (both edit `backend/Makefile`, but different comment blocks — no
  conflict) and can land in either order.
- If a developer changes `REDIS_PORT`, they must change the port in their `TEST_REDIS_URL` to match;
  the comment in `.env.example` says so. Deriving one from the other is not worth a shell layer for a
  two-line file.
- Scope guard: no changes to the images, credentials, healthchecks or the `down` recipe; no Go code.

---

## Execution summary

**Built on the existing branch** `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001`
in `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001` (amending plan — no new
branch, no new worktree).

Commits:

```
ff03447 fix(backend): make dev stack host ports overridable and persist postgres data
1f34336 docs(backend): add .env.example for dev stack ports and test database URLs
5117223 codemap: dev stack port overrides and the persistent postgres volume
```

### Deviations

1. **`backend/.gitignore` also needs `!.env.example`.** Task 2 Step 2 says ".env.example is
   unaffected by that pattern and stays tracked — Step 4 proves it." Step 4 proved the opposite: the
   **repo-root** `.gitignore` carries `.env*` (line 6), which swallows `backend/.env.example`:

   ```
   $ git add .env.example .gitignore Makefile
   The following paths are ignored by one of your .gitignore files:
   backend/.env.example
   $ git check-ignore -v .env.example
   .gitignore:6:.env*	.env.example
   ```

   The plan's own Step 4 and Verification #6 require `.env.example` to be tracked, so the fix is
   inside the plan's intent. `git add -f` was rejected: it would leave the next developer's copy
   untracked and the file permanently invisible to `git status`. Instead `backend/.gitignore` (a file
   this task already modifies) re-includes it — a deeper `.gitignore` takes precedence over the root:

   ```
   /api
   .env
   # The repo-root .gitignore has `.env*`, which would swallow the committed
   # example too. This deeper file wins, so re-include it explicitly.
   !.env.example
   ```

   ```
   $ git check-ignore -q .env;         echo $?   → 0  (ignored, as intended)
   $ git check-ignore -q .env.example; echo $?   → 1  (NOT ignored)
   $ git ls-files .env.example                   → .env.example
   ```

2. **`harness/CODEMAP.md` updated** (commit `5117223`), which the plan's file table does not list.
   The CODEMAP tells a human to run `make up`; leaving it silent about `.env` would have re-created
   the documentation gap this plan exists to close. Required by the executor role's "update CODEMAP
   for any package you change". One clause, no other change.

Everything else was implemented exactly as written.

### Verification

**1. The override the executor previously kept outside the repo, now expressible in it. No containers started.**

```
$ REDIS_PORT=6380 docker compose config | grep -n 'published'
21:        published: "5432"
43:        published: "6380"
exit=0
```

**2. Defaults still apply on a clean machine.**

```
$ env -u REDIS_PORT -u POSTGRES_PORT docker compose config | grep -n 'published'
21:        published: "5432"
43:        published: "6379"
```

**3. Both ports override independently.**

```
$ POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose config | grep -n 'published'
21:        published: "5433"
43:        published: "6380"
```

**4. Valid Compose, volume declared.**

```
$ docker compose config --quiet; echo "config exit=$?"
config exit=0
$ docker compose config --volumes
postgres_data
$ docker compose config | grep -n 'postgres_data'
25:        source: postgres_data
49:  postgres_data:
50:    name: backend_postgres_data
```

**5. Nothing started by the verification, owner's container untouched.**

```
$ docker ps --format '{{.Names}}\t{{.Ports}}' | grep -n 'scio3-redis-1\|backend-'
1:scio3-redis-1	0.0.0.0:6379->6379/tcp, [::]:6379->6379/tcp
```

No `backend-postgres-1` / `backend-redis-1` — the verification section itself never starts a
container. (The separate *Runtime proof* below does, on 5433/6380, and tears them down.)

**6. Docs and ignores.**

```
$ grep -n 'POSTGRES_PORT\|REDIS_PORT' docker-compose.yml Makefile .env.example
docker-compose.yml:3:# Host ports are overridable: copy .env.example to .env and set POSTGRES_PORT /
docker-compose.yml:4:# REDIS_PORT if something else on this machine already holds 5432 or 6379.
docker-compose.yml:15:      - "${POSTGRES_PORT:-5432}:5432"
docker-compose.yml:25:      - "${REDIS_PORT:-6379}:6379"
.env.example:6:POSTGRES_PORT=5432
.env.example:7:REDIS_PORT=6379
Makefile:12:# Ports come from .env (copy .env.example). Set REDIS_PORT / POSTGRES_PORT there
$ git check-ignore -v .env
backend/.gitignore:2:.env	.env
$ git ls-files .env.example
.env.example
```

**7. Go side untouched and green.**

```
$ env -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1
?   	.../cmd/api	[no test files]
ok  	.../internal/config	0.404s
ok  	.../internal/health	0.245s
ok  	.../internal/store	0.462s
exit=0
```

**8. Harness artifacts.**

```
$ python3 tools/harness/cli.py validate; echo "exit=$?"
exit=0
$ git -C .worktrees/store-go-module-postgres-and-redis-clients-migration-0001 status --short
(clean)
```

### Runtime proof

This plan makes the dev stack startable on this machine, so the branch's full end-to-end proof was
run here (it covers the sibling blocker
`2026-09-22-go-test-drops-every-table-in-whatever-database-url-points-at.md` as well).

**a. It builds.**

```
$ go build ./... && go vet ./...     → clean
$ go build -o <scratch>/api ./cmd/api
BUILD_OK
```

**b. The whole suite, clean shell.**

```
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL -u PORT go test ./... -count=1
?   	.../cmd/api	[no test files]
ok  	.../internal/config	0.410s
ok  	.../internal/health	0.242s
ok  	.../internal/store	0.466s
exit=0
```

**c/d. The documented workflow, exactly as the Makefile and `.env.example` describe it.**
Host 6379 is held by the owner's unrelated `scio3-redis-1`, which is precisely the case this plan
makes expressible. Copying `.env.example` to `.env` and editing the two ports is the documented path:

```
$ sed -e 's/^POSTGRES_PORT=5432/POSTGRES_PORT=5433/' -e 's/^REDIS_PORT=6379/REDIS_PORT=6380/' \
      -e 's#localhost:5432#localhost:5433#' -e 's#localhost:6379#localhost:6380#' .env.example > .env
$ git status --short
(clean — .env is ignored, as Task 2 Step 2 requires)

$ make up
docker compose up -d
 Volume "backend_postgres_data"  Created
 Container backend-postgres-1  Started
 Container backend-redis-1  Started
up exit=0

$ docker compose ps --format '{{.Name}}\t{{.State}}\t{{.Ports}}'
backend-postgres-1	running	0.0.0.0:5433->5432/tcp, [::]:5433->5432/tcp
backend-redis-1	running	0.0.0.0:6380->6379/tcp, [::]:6380->6379/tcp
```

`make up` succeeds with no flags and no files outside the repo — the exact thing the reviewer could
not do.

**`make test-integration` with the nominated test databases — the tests now actually run:**

```
$ export TEST_DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable'
$ export TEST_REDIS_URL='redis://localhost:6380/0'
$ make test-integration
=== RUN   TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.07s)
=== RUN   TestIntegrationPetStatesRejectsASecondRowForTheSameUser
--- PASS: TestIntegrationPetStatesRejectsASecondRowForTheSameUser (0.04s)
=== RUN   TestIntegrationRedisRoundTrip
--- PASS: TestIntegrationRedisRoundTrip (0.00s)
ok  	.../internal/store	0.390s
```

That closes the loop on the sibling blocker: the gate skips on the production variables and runs on
the `TEST_` ones, both proved against a live database.

**It boots and answers:**

```
$ DATABASE_URL='postgres://english:english@localhost:5433/english?sslmode=disable' \
  REDIS_URL='redis://localhost:6380/0' PORT=8099 ./api &
2026/09/22 16:55:21 migrations applied: [0001_init]
2026/09/22 16:55:21 listening on :8099
[GIN-debug] GET    /healthz  --> ...internal/health.Handler.func1 (3 handlers)

$ curl -fsS http://127.0.0.1:8099/healthz
{"postgres":"ok","redis":"ok","status":"ok"}
$ curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8099/healthz
200
[GIN] 2026/09/22 - 16:55:26 | 200 |   6.41ms |       127.0.0.1 | GET      "/healthz"
```

**The named volume does what the comment claims — `make down` keeps the dev database:**

```
$ docker compose exec -T postgres psql -U english -d english -Atc "select version from schema_migrations"
0001_init
$ make down
docker compose down     → containers and network removed, down exit=0
$ docker volume ls --format '{{.Name}}' | grep backend_postgres_data
backend_postgres_data
$ make up && docker compose exec -T postgres psql -U english -d english -Atc "select version from schema_migrations"
0001_init          # survived down + up
$ make down
 Network backend_default  Removed
```

**e. Safety and cleanliness.**

```
$ docker ps --format '{{.Names}}\t{{.Ports}}' | grep -n 'scio3-redis-1\|backend-'
1:scio3-redis-1	0.0.0.0:6379->6379/tcp, [::]:6379->6379/tcp

$ docker inspect -f 'StartedAt={{.State.StartedAt}} Status={{.State.Status}} RestartCount={{.RestartCount}}' scio3-redis-1
StartedAt=2026-09-22T01:56:03.734248879Z Status=running RestartCount=0
```

The owner's unrelated container was never stopped or restarted (start time ~8h before this run,
`RestartCount=0`), and nothing this run started remains. The scratch `.env` was removed afterwards;
the worktree is clean and the full suite is still green from a clean shell.

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
