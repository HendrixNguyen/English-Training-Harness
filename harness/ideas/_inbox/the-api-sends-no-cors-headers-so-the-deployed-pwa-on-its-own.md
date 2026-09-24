---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
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
