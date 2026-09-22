---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
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
