---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
plan: harness/plans/2026-09-24-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md
---
# The API sends no CORS headers, so the deployed PWA on its own Railway domain cannot call a single endpoint

## Why
Spec §8 deploys two separate Railway services: the Nuxt PWA and the Go API, on two different origins. The browser therefore makes every `/api/v1/*` call cross-origin, and every guarded call carries an `Authorization` header, which forces a CORS preflight. The backend has no CORS middleware and no `Access-Control-*` header anywhere: `grep -rni 'cors|Access-Control' backend/internal backend/cmd --include='*.go'` returns nothing, and `cmd/api/main.go` builds a bare `gin.Default()` with no such middleware between it and the route groups.

Against the running binary this is not theoretical: `OPTIONS /api/v1/onboarding/quiz` with `Origin:` and `Access-Control-Request-Method: GET` answers **`404 page not found`** (gin has no OPTIONS route), and a plain `GET` with an `Origin` header returns 200 with **no `Access-Control-Allow-Origin`**. A browser would block both — the preflight outright, and the simple request's response from being read. The result is that the shipped PWA gets zero data from the API: sign-in, `/quests/daily`, `/pet/status`, `/onboarding/*`, all of it. Nothing caught this because every browser proof to date (this review included) has been driven same-origin or through a local shim, and CI never puts a browser in front of the Go binary.

This is pre-existing and touches every endpoint, not any one slice.

## Expected output
The API answers cross-origin browser requests from the configured frontend origin: `Access-Control-Allow-Origin` (an explicit allow-list from an env var such as `FRONTEND_ORIGIN`, never `*` alongside credentials), `Access-Control-Allow-Headers: Authorization, Content-Type`, `Access-Control-Allow-Methods: GET, POST, OPTIONS`, and a `204` on preflight `OPTIONS` for every `/api/v1/*` route. A handler test asserts the preflight and the allow-origin header; the Railway checklist (backend spec §9) lists the new variable.

## Evidence
- Found while reviewing `harness/plans/2026-09-23-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md`; the browser proof there only worked because the reviewer put a same-origin proxy in front of the API.
- `backend/cmd/api/main.go` — `r := gin.Default()`, then `r.GET("/healthz", …)` and the `/api/v1` group; no middleware in between.
- `grep -rni "cors\|Access-Control" backend/internal backend/cmd --include='*.go'` → no output.
- Live against the merged binary (API on 8107): `curl -X OPTIONS -H 'Origin: https://app.example.com' -H 'Access-Control-Request-Method: GET' .../api/v1/onboarding/quiz` → `HTTP/1.1 404 Not Found`; authenticated `GET` with `Origin:` → `HTTP/1.1 200 OK` with no `Access-Control-Allow-Origin` in the response headers.
- Spec §8 (two Railway deployables) and `frontend/nuxt.config.ts` `apiBase` / `NUXT_PUBLIC_API_BASE` — the browser is pointed at the API's own origin.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — high. Planned today as the head of `harness/plans/2026-09-24-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md`.**

*Is the Why real?* Yes, and it is the gate on every frontend idea in production. Spec §8 / backend spec §9 deploy the Nuxt PWA and the Go binary as two Railway services on two origins; every `/api/v1/*` call is cross-origin and every guarded call carries `Authorization`, so the browser preflights. The 2026-09-24 ideation run itself notes "every frontend idea is inert in production until the CORS inbox bug is fixed".

*Root cause (read on this branch).* `backend/cmd/api/main.go` builds `r := gin.Default()` and mounts `/healthz` and the `/api/v1` group with no middleware in between; `grep -rni 'cors\|Access-Control' backend/` is empty and `backend/go.mod` has no CORS library. Gin registers no `OPTIONS` routes, so a preflight falls to the NoRoute chain and is answered `404`. Gin rebuilds that NoRoute chain from the global middleware on every `engine.Use`, so one global middleware that answers a preflight with `204` and echoes an allow-listed `Origin` fixes every present and future route, including the ones that do not exist yet.

*Decision.* Hand-rolled middleware in a new `backend/internal/middleware` package (≈ 60 lines; no dependency): allow-list from `FRONTEND_ORIGIN` (comma-separated exact origins; default `http://localhost:3000`, the Nuxt dev server), `Access-Control-Allow-Origin` echoes the matched origin (never `*`), `Vary: Origin`, preflight → `204` with `Allow-Methods: GET, POST, OPTIONS`, `Allow-Headers: Authorization, Content-Type`, `Max-Age: 600`; a preflight from an origin not on the list → `403`. No `Allow-Credentials`: the session is a bearer header, not a cookie. Spec §9's Railway list gains `FRONTEND_ORIGIN`.

*Also planned in the same branch* (same file, same class — request-edge hardening on the one router): `no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md` (one `MaxBytesReader` middleware on the `/api/v1` group) and `healthz-leaks-postgres-and-redis-driver-error-strings-public.md` (fixed `"unavailable"` markers, per-ping deadline). *Dependencies:* none. *Conflict:* the approved cmd/api shutdown plan also edits `main.go`, `config.go` and `.env.example`; the plan writes region edits and a merge-order note.
