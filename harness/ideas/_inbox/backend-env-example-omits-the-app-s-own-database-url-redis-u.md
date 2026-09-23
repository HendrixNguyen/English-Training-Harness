---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Folded into harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md, whose Task on .env.example/Makefile adds the application section (DATABASE_URL, REDIS_URL, PORT, GIN_MODE, JWT_SECRET, GOOGLE_*) and the note that the Go process does not read .env."
---
# backend/.env.example omits the app's own DATABASE_URL REDIS_URL PORT

## Why
`backend/.env.example` is the file `harness/CODEMAP.md` and `backend/Makefile:12` both tell a
developer to copy, and its name promises the backend's environment. It contains only four
variables — `POSTGRES_PORT`, `REDIS_PORT`, `TEST_DATABASE_URL`, `TEST_REDIS_URL` — and none of the
three the application itself requires:

```
$ grep -n 'DATABASE_URL\|REDIS_URL\|PORT' backend/.env.example
6:POSTGRES_PORT=5432
7:REDIS_PORT=6379
12:TEST_DATABASE_URL=postgres://english:english@localhost:5432/english?sslmode=disable
13:TEST_REDIS_URL=redis://localhost:6379/0
```

`internal/config.Load` requires `DATABASE_URL` and `REDIS_URL` and defaults `PORT`
(`internal/config/config.go:24-32`). Nothing in the Go process reads a `.env` file — there is no
`godotenv` dependency — so a developer who follows the documented path exactly (copy
`.env.example` → `.env`, `make up`, `make run`) gets:

```
$ env -u DATABASE_URL -u REDIS_URL make run
2026/09/22 17:03:37 config: config: DATABASE_URL is required
exit status 1
```

The refusal itself is correct and safe. The defect is that the repo's one environment template does
not describe the environment the binary needs, and nothing says the template is Compose-only. There
is also nowhere else a newcomer can discover the §8 variable names: `make run` has no comment, and
the CODEMAP `store` bullet mentions `DATABASE_URL` / `REDIS_URL` only to warn that they are *not*
the test variables.

(The doubled prefix in that error — `config: config:` — comes from `main.go:21` re-prefixing an
error that `config.go:25` already prefixed. Same pattern at `main.go:26`, `:32`, `:38`. Worth
tidying alongside, since every later slice copies this wiring file.)

## Expected output
Copying `backend/.env.example` to `backend/.env` and following the documented commands works with no
undocumented step:

- `.env.example` gains a clearly-labelled section for the spec §8 application variables —
  `DATABASE_URL`, `REDIS_URL`, `PORT` — pointing at the dev stack's ports and consistent with the
  `POSTGRES_PORT` / `REDIS_PORT` values above them.
- The file states in one line which consumer reads which half: Docker Compose reads this file
  automatically for the `*_PORT` values; the Go process does **not**, so the application variables
  must be exported (or the `run` target must source them).
- Either `make run` exports them from `.env`, or its Makefile comment says how to (`set -a; . ./.env; set +a`),
  so `make up && make run` is a complete, documented local loop.
- The `config:` prefix is emitted once, not twice, on a config failure.

## Evidence
- Plan under review: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
- File introduced by: `harness/plans/2026-09-22-docker-compose-hard-codes-host-ports-so-make-up-fails-locall.md`
  (Task 2 Step 1 — the plan's verbatim content, so this originates in the plan text).
- `backend/.env.example:1-13`; `backend/internal/config/config.go:24-32`; `backend/Makefile:6-7`
  (`run` target, no comment) and `:12-14` (the comment that points at `.env.example`).
- `harness/CODEMAP.md:7` — "copy `backend/.env.example` to `backend/.env`".
- Reviewer reproduction: `env -u DATABASE_URL -u REDIS_URL make run` → `config: config: DATABASE_URL is required`, exit 1.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — folded.** Still true on `main` (`backend/.env.example` has ports, TEST_* URLs and AI keys only; `config.Load` requires five more). The cmd/api hardening plan adds `GIN_MODE` to config and must document it in `.env.example` anyway, so it carries this whole fix (app section + "Go does not read .env" note + `make run` comment + the doubled `config: config:` prefix).
