---
plan: harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md
verdict: pass
bugs: []
---
# Review — docker-compose hard-codes host ports so make up fails locally

**Plan:** `harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md`
**Branch/worktree:** `harness/2026-09-22-high-store-go-module-postgres-and-redis-clients-migration-0001` / `.worktrees/store-go-module-postgres-and-redis-clients-migration-0001` — both gone; the plan is already merged.
**Reviewed on:** a detached worktree of `origin/main` at `3f4242d` (post-hoc review, 2026-09-25 daily run). The plan commits are on main: `ff03447`, `1f34336`, `5117223`.

## Plan vs idea
The idea's headline is "`make up` succeeds on a machine that already has something on 5432 or 6379". It is **delivered, and I proved it live.** Port 6379 on this machine is held by the unrelated `scio3-redis-1`. With a scratch `backend/.env` of `COMPOSE_PROJECT_NAME=rev0925olds`, `POSTGRES_PORT=5447`, `REDIS_PORT=6397`, a plain `make up` started both services on 5447 and 6397 and never touched `scio3-redis-1`. `make down` then removed them.
- Parameterised ports with defaults: done.
- Named `postgres_data` volume: done.
- The idea's second bullet asked for the Makefile's URL comment to *derive* from the port variables. The plan chose not to, and said so in its Notes ("not worth a shell layer"). Instead the `.env.example` comment says to keep the ports in step. That is a justified, documented deviation. The Makefile's `export TEST_DATABASE_URL=…:5432…` example can still drift from a changed `POSTGRES_PORT`, but only in a comment.

## Code vs plan
- **Task 1 (compose): followed**, and unchanged since.
- **Task 2 (`.env.example`, `.gitignore`, Makefile comment): followed.** `backend/.gitignore:2` ignores `.env`, and `git ls-files` tracks `.env.example`. The `up:` comment is present. `.env.example` has grown a lot since then (app, AI, VAPID and secrets sections added by later plans).
- **Task 3 (CODEMAP): followed.** The CODEMAP CI section documents the `POSTGRES_PORT/REDIS_PORT` repro.

Verification re-run (with the scratch `.env` moved aside):
```
REDIS_PORT=6380 docker compose config | grep published         # "5432", "6380"
env -u REDIS_PORT -u POSTGRES_PORT docker compose config | grep published   # "5432", "6379"
POSTGRES_PORT=5433 REDIS_PORT=6380 docker compose config | grep published   # "5433", "6380"
docker compose config --quiet                                   # exit 0
docker compose config --volumes                                 # postgres_data
git check-ignore -v .env                                        # backend/.gitignore:2:.env
make up (scratch .env)  → rev0925olds-postgres-1 :5447, rev0925olds-redis-1 :6397; scio3-redis-1 untouched
```

## Quality
Observations, none filed:
- `.env.example` has `JWT_SECRET=` twice: once uncommented and empty in the app block (line 19), and once commented out in the secrets block (line 60). This is cosmetic and came from later plans. The open plan `2026-09-25-the-documented-set-a-env-export-also-exports-test-database-u.md` rewrites this file and is the natural place to dedupe it.
- `.env.example` does not mention `COMPOSE_PROJECT_NAME`, although AGENTS.md requires a unique one per worktree. The named volume this plan added makes that rule matter more: worktrees on the default project share `backend_postgres_data` as well as the containers. AGENTS.md already carries the rule, so this is a documentation nicety.
- The danger of the `.env.example` `TEST_*` lines pointing at the dev database (sourcing `.env` makes the destructive tests run against dev data) is already tracked as a planned medium bug: `the-documented-set-a-env-export-also-exports-test-database-u.md`.

## Bugs filed
None.

## Verdict
**pass.** Delivered and still true on current main. The one deviation from the idea is documented and reasonable.
