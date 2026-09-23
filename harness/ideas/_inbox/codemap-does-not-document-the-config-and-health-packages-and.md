---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# CODEMAP does not document the config and health packages and still says three CI jobs

## Why
`harness/CODEMAP.md` is the map every role reads before touching code, and two of the nine backend
packages have no paragraph: `internal/config` (`config.Load`, the required env list, the `PORT`
default, the `JWT_SECRET`-not-in-spec note) and `internal/health` (`GET /healthz`, the shared 2 s
`pingTimeout`, the 503 body). Both are the packages the main.go / config-hardening fixes in the
inbox will change, and an executor reading CODEMAP today would not know they exist. The CI section
also still opens "Three parallel GitHub Actions jobs" above a list of four (the frontend job).

Folds in `codemap-ci-section-still-says-three-parallel-jobs-after-the-.md` (rejected as a duplicate
of this file's third item). Filed by the evaluator at the orchestrator's request while triaging the
inbox, 2026-09-23.

## Expected output
- A `**config**` bullet: `Load()` reads `DATABASE_URL`, `REDIS_URL`, `PORT` (default 8080),
  `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `JWT_SECRET` (not in spec §8); presence-only validation;
  tests in `config_test.go`.
- A `**health**` bullet: `Handler(db, cache Pinger)`; one `pingTimeout` (2 s) shared by two sequential
  pings; 200 `{status, postgres, redis}` or 503 with the failing dependency named (today the raw driver
  error — see `healthz-leaks-postgres-and-redis-driver-error-strings-public.md`).
- CI section opens "Four parallel GitHub Actions jobs".
- Executors of the cmd/api and health plans update these bullets rather than creating them.

## Evidence
- `grep -n '^\- \*\*' harness/CODEMAP.md` → store, auth, onboarding, quests, pet, airouter, google,
  notify, shell; no config, no health.
- `harness/CODEMAP.md` CI section: "Three parallel GitHub Actions jobs" followed by four bullets.
- `backend/internal/config/config.go`, `backend/internal/health/health.go` (evaluator, 2026-09-23).

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage._

**Select — low.** Documentation debt in the file every role reads first. The cmd/api hardening plan adds a `cmd/api` paragraph and names `GIN_MODE`; this idea adds `config` and `health` and fixes the job count. Ten minutes of executor time; batch with any CODEMAP-touching plan.
