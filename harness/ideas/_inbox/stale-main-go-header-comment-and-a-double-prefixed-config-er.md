---
type: bug
status: rejected
source: reviewer
run: _inbox
priority: low
rejected_reason: "Folded into harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md, which rewrites main.go's header comment and drops the redundant config: prefix as part of the same edit."
---
# stale main.go header comment and a double-prefixed config error

## Why
Two cosmetic inaccuracies in `cmd/api`, both of which a reader hits immediately.

- The package comment still reads "This slice serves only `GET /healthz`; later slices mount their
  own route groups here." That was true of slice 1. This slice mounts
  `POST /api/v1/auth/google` fifteen lines below the comment that denies it exists. The file is the
  first thing anyone opens to find out what the process serves.
- `main.go` logs config failures as `log.Fatalf("config: %v", err)` while every error from
  `config.Load` already begins with `config: `. The observed output on a boot with no environment is
  `config: config: DATABASE_URL is required`. The same doubling applies to the `postgres:`, `redis:`
  and `migrate:` prefixes, whose errors are already wrapped as `store: ...` — those read
  `postgres: store: parsing DATABASE_URL: ...`.

## Expected output
- The package comment describes the routes actually mounted, or stops enumerating them and points at
  CODEMAP instead so it cannot go stale again.
- The `log.Fatalf` calls drop the redundant prefix (`log.Fatal(err)`), or the wrapped errors drop
  theirs — one or the other, consistently. First boot output should read
  `config: DATABASE_URL is required`.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
- `backend/cmd/api/main.go:1-2` — "This slice serves only GET /healthz" vs `:57` — `v1.POST("/auth/google", auth.Handler(authSvc))`.
- `backend/cmd/api/main.go:23` — `log.Fatalf("config: %v", err)`; `backend/internal/config/config.go:30,33,49` — errors already prefixed `config: `.
- Observed in the plan's worktree: running the built binary with no environment printed
  `config: config: DATABASE_URL is required`.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Reject — folded.** Both cosmetic; both in the file the cmd/api hardening plan rewrites.
