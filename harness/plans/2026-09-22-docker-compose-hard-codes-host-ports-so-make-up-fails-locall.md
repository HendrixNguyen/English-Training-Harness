---
idea: harness/ideas/_inbox/docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md
status: draft
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
