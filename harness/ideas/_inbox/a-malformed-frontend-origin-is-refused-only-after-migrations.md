---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# A malformed FRONTEND_ORIGIN is refused only after migrations have run against the database

## Why
`FRONTEND_ORIGIN` is a pure config value, but it is validated in `cmd/api/main.go:107` (`middleware.ParseOrigins(cfg.FrontendOrigin)`), after `main` has connected to Postgres and applied migrations. A typo in the env var therefore still touches the production database (takes the migration lock, may apply pending migrations) before the process refuses to boot, and with Postgres unreachable the operator sees a database dial error instead of the origin error — the one message this plan worked to make clear. The log line is also prefixed `config:` although it is not produced by `config.Load`. Pre-existing ordering, surfaced while reviewing this plan: pure-config validation should fail before any side effect.

## Expected output
`FRONTEND_ORIGIN` is parsed before any store connection — either inside `config.Load` (storing the parsed list on `Config`) or as the first thing in `main` after `config.Load`. Booting with a bad origin and no database prints the origin error and exits 1 without a dial attempt.

## Evidence
- Plan: `harness/plans/2026-09-25-parseorigins-accepts-frontend-origin-entries-no-browser-send.md` (the boot-refusal path it strengthens).
- `backend/cmd/api/main.go:107` (branch @ `4884a04`, same on `main`).
- Reviewer run 2026-09-27: built `./cmd/api`, `FRONTEND_ORIGIN='https://*.up.railway.app'` with `DATABASE_URL` pointing at a closed port → `migrate: store: locking migrator: … connection refused`, no origin error; against a live scratch Postgres the same value → `config: middleware: FRONTEND_ORIGIN entries must be … (wildcard hosts are not supported …)`, `exit=1`, after migrations ran.
