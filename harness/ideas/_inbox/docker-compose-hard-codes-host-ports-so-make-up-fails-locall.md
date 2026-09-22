---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md
plan: harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md
---
# docker-compose hard-codes host ports so make up fails locally

## Why
`backend/docker-compose.yml` publishes `"5432:5432"` and `"6379:6379"` with no way to override the
host side. The repo owner runs other projects on this machine — `scio3-redis-1` already holds host
port 6379 — so `make up`, the one command the dev stack exists for, fails outright:

```
Error response from daemon: … Bind for 0.0.0.0:6379 failed: port is already allocated
make: *** [up] Error 1
```

The executor hit exactly this during execution, worked around it with a throwaway override file kept
outside the repo, and committed the compose file unchanged. So the repo ships a local-dev stack that
is known not to start on the only machine that uses it, and the workaround is not written down
anywhere a future reader will find it. Every later slice that needs a local Postgres or Redis
inherits the failure.

A secondary defect in the same file: the `postgres` service declares no named volume, so its data
lives in an anonymous volume that a `down` / `up` cycle discards. Each `make up` gives a fresh empty
database. That may be tolerable, but it is undocumented and surprising.

## Expected output
`make up` succeeds on a machine that already has something on 5432 or 6379.

- Host ports parameterised with defaults, e.g. `"${POSTGRES_PORT:-5432}:5432"` and
  `"${REDIS_PORT:-6379}:6379"`, so a developer sets `REDIS_PORT=6380` once (an `.env` next to the
  compose file is read automatically by Compose) instead of maintaining an override file.
- The Makefile comment block that spells out `DATABASE_URL` / `REDIS_URL` updated to derive from the
  same variables, so the documented URLs cannot drift from the published ports.
- A named volume for `postgres` (or an explicit comment saying data is intentionally ephemeral).

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  Task 9 Step 1 — the fixed ports come from the plan text, so this is a plan defect the executor
  reproduced faithfully.
- `backend/docker-compose.yml:10-11` (postgres ports), `:20-21` (redis ports); no `volumes:` key.
- `backend/Makefile:12-13` (`up: docker compose up -d`), `:18-20` (hard-coded URLs in the comment).
- The plan's own *Local stack verification* section records the `make up` failure and the 6380
  override used instead.

## Evaluation

**Verdict: select, `priority: high`.** Merge blocker on
`harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`, confirmed
real by the review and by the executor's own run. Not re-litigated here.

**Is the *Why* real?** Yes. `backend/docker-compose.yml:10-11` and `:20-21` publish `"5432:5432"` and
`"6379:6379"` with no indirection. Host port 6379 on the only machine that uses this repo is held by
an unrelated `scio3-redis-1` container that must keep running, so `make up` — the single command the
dev stack exists to provide — fails with `Bind for 0.0.0.0:6379 failed: port is already allocated`.
The executor hit exactly this, used a throwaway override file kept outside the repo, and committed
the compose file unchanged as the plan specified; the reviewer then declined to start containers for
the same reason and judged the file by reading. So the repo currently ships a local stack that is
known not to start, and the only working recipe exists nowhere in version control. Every later slice
that needs a local Postgres or Redis inherits it.

The secondary defect is real too: `postgres` declares no volume, so its data lands in an anonymous
volume that `docker compose down` discards. `make down && make up` silently hands back an empty
database. Tolerable for a dev stack, surprising and undocumented as shipped.

**Achievable in one plan?** Yes — two port lines, a volume, one new example env file, two Makefile
comments. Trivially under a day.

**Dependencies:** none.

**Root cause (systematic-debugging, read-only).** Again a plan defect, not an execution one: Task 9
Step 1 dictates the compose file verbatim and the executor reproduced it faithfully. The plan assumed
a clean machine. Compose already solves this — it reads a `.env` beside the compose file and
interpolates `${VAR:-default}` in the `ports` mapping — so the fix is to use the mechanism rather
than to document a workaround.

**Chosen fix (owner's call).** `"${POSTGRES_PORT:-5432}:5432"` and `"${REDIS_PORT:-6379}:6379"`, a
committed `backend/.env.example` documenting both ports together with the matching
`TEST_DATABASE_URL` / `TEST_REDIS_URL` values (the sibling blocker's new test-only variables, so the
two fixes agree on one story), a named volume for Postgres data, and a Makefile note on overriding.
A committed `.env` was rejected — Compose reads it automatically, so committing one would silently
pin every developer to this machine's ports; the example file plus a `.gitignore` entry keeps the
real `.env` local. Deriving the documented URLs from the same variables is what stops the Makefile's
comment block from drifting away from the published ports, which is the drift the idea calls out.

**Priority rationale:** `high`. It blocks an unmerged branch and it breaks the repo's primary
developer entry point on the owner's machine — everything downstream that needs a database is
stalled behind it.
