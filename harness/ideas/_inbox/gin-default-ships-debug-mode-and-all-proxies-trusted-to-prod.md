---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# gin.Default ships debug mode and all-proxies-trusted to production

## Why
`backend/cmd/api/main.go:44` builds the router with a bare `gin.Default()` and nothing anywhere in
the repo ever calls `gin.SetMode(gin.ReleaseMode)` or sets `GIN_MODE`:

```
$ grep -rn 'GIN_MODE\|SetMode\|TrustedProxies' backend/ harness/CODEMAP.md
backend/internal/health/health_test.go:19:	gin.SetMode(gin.TestMode)
```

The test file is the only caller, so the deployed binary runs in Gin's **debug** mode. The reviewer
booted the branch's binary against the dev stack and Gin said so itself:

```
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
```

Two distinct problems, one of them security-relevant:

1. **Debug mode in production.** Gin dumps its route table and per-request debug lines to stdout,
   and its recovery middleware is more verbose. Spec §8 deploys this binary to Railway, where stdout
   is the log stream, so every request is logged twice with no way to turn it down.
2. **All proxies trusted.** `gin.Default()` leaves `TrustedProxies` at its default of
   `0.0.0.0/0, ::/0`, which means `c.ClientIP()` returns whatever the caller puts in
   `X-Forwarded-For`. Nothing reads `ClientIP()` in this slice, but this is the first application
   code in the repo and every later slice mounts its handlers on this same engine — the auth slice
   (login attempts) and the AI rate-limit slice (spec §4 `ratelimit:ai:{user_id}`, 5 req/min) are
   both natural `ClientIP()` callers, and on Railway there is exactly one real proxy in front of the
   app, so the correct trusted set is knowable today. Fixing it after handlers depend on it is
   strictly harder than fixing it now.

This is a foundation defect rather than a behaviour bug: `/healthz` answers 200 correctly either way.

## Expected output
`cmd/api` selects Gin's mode and trusted-proxy set from configuration rather than accepting the
library defaults:

- `gin.SetMode(...)` is called before the engine is built, defaulting to `gin.ReleaseMode` and
  switchable to debug for local work through a config value (`GIN_MODE`, read via
  `internal/config` alongside the other spec §8 variables, not by scattering `os.Getenv` in `main`).
- `r.SetTrustedProxies(...)` is called explicitly with the platform's proxy set, or
  `r.SetTrustedProxies(nil)` where `ClientIP()` is not to be trusted at all. The
  `You trusted all proxies` warning no longer appears at boot.
- `internal/config` gains the new variable with a test covering its default, matching the existing
  `PORT` pattern in `internal/config/config_test.go`.
- `backend/.env.example` and the `harness/CODEMAP.md` `store`/`cmd/api` entry name the new variable.
- A boot in a clean shell prints no `[GIN-debug]` lines.

## Evidence
- Plan: `harness/plans/2026-09-22-store-go-module-postgres-and-redis-clients-migration-0001.md`
  (Task 8 — `main.go` is the plan's verbatim code block, so this originates in the plan text).
- `backend/cmd/api/main.go:44` — `r := gin.Default()`.
- `backend/internal/config/config.go:11-15` — `Config` has no mode or proxy field.
- Reviewer runtime reproduction: built `./cmd/api` and ran it against the dev stack on
  `POSTGRES_PORT=5433 / REDIS_PORT=6380`; both `[GIN-debug] [WARNING]` lines above appear at boot,
  and `curl /healthz` returned `200 {"postgres":"ok","redis":"ok","status":"ok"}`.
- Spec §8 — Railway deployment, stdout is the log stream and a platform proxy fronts the app.
