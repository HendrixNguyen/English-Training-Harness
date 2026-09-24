---
plan: harness/plans/2026-09-24-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/a-slow-request-body-still-holds-its-goroutine-indefinitely-n.md, harness/ideas/_inbox/parseorigins-accepts-frontend-origin-entries-no-browser-send.md]
---
# Review — API edge: a CORS allow-list for the PWA's origin, bounded request bodies, and a `/healthz` that names no secrets

**Plan:** `harness/plans/2026-09-24-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md`
**Branch/worktree:** `harness/2026-09-24-high-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own` / `.worktrees/the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own`
**Diff:** `git diff origin/main...harness/2026-09-24-high-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own --stat`

## Plan vs idea
**CORS (head idea): delivered in full.** Every item in *Expected output* is there and was observed on the live binary. The allow-list comes from `FRONTEND_ORIGIN` and is never `*`; there is no `Allow-Credentials`. The headers are `Allow-Headers: Authorization, Content-Type` and `Allow-Methods: GET, POST, OPTIONS`. A preflight gets `204` on every `/api/v1/*` path, including a path with no GET route. Handler tests assert the preflight and the `Access-Control-Allow-Origin` header, and spec §9 lists `FRONTEND_ORIGIN`. The allow-lists also match what the PWA actually sends. `frontend/utils/apiClient.ts` uses only `GET`/`POST` and only `Accept` (CORS-safelisted), `Authorization` and `Content-Type`, and `main.go` registers only GET/POST routes.

**healthz (folded idea): delivered.** Each dependency reports a fixed `"unavailable"` marker, the endpoint still returns `503`, the driver error is `log.Printf`ed on the server, and the tests are rewritten. The folded shared-deadline finding is fixed as well: each ping gets its own `PingTimeout`.

**Body bound (folded idea): the size half is delivered, the time half is not.** One `MaxBytesReader` middleware sits on the `/api/v1` group, oversized bodies get `400 invalid_request`, and a test covers it. That idea's *Expected output* also lists server `ReadTimeout`/`WriteTimeout`. The evaluator deliberately moved those to the cmd/api shutdown plan, and leaving `WriteTimeout` unset is justified there. But the shutdown plan sets only `ReadHeaderTimeout` and `IdleTimeout`, so **no plan sets `ReadTimeout`**. A slow body sender still holds a goroutine. This plan's *Notes* also say `MaxBytesReader` closes the connection, which it does not do through gin's writer. Filed as bug #1 (medium, not a blocker: it predates this branch, and this branch strictly narrows it).

## Code vs plan
Diff base is `origin/main` (fetched). The branch is 5 commits on `fe2c29a`. The two commits `origin/main` has gained since (#12, #13) touch no `backend/` file, so there is no conflict.

| Task | Result |
| --- | --- |
| 1 config `FRONTEND_ORIGIN` + default | Followed verbatim |
| 2 `middleware.CORS` / `ParseOrigins` | Followed verbatim, including the redundant scheme condition the plan said could be simplified (bug #2) |
| 3 `middleware.BodyLimit` | Followed verbatim |
| 4 health markers, logging, per-ping budget | Followed verbatim |
| 5 wiring, `.env.example`, spec §9, CODEMAP | Followed. CORS `Use` is at L88, before `/healthz` at L91. `BodyLimit` is at L147, right after `r.Group` (L146) and before `v1.Group("", auth.Require…)`, so the guarded subgroup inherits it |

The executor's one deviation (file edits via Bash instead of the Edit tool) is tooling only; the content matches the plan.

**Re-run evidence (reviewer, in the worktree; everything reproduced, so no executor gate failure):**
```
$ gofmt -l ./internal/middleware ./internal/health ./internal/config ./cmd/api     → (no output)
  (gofmt -l . also lists internal/quests/{handler_test,repo}.go: already on main, and a sibling gofmt plan owns them)
$ go build ./... && go vet ./... && go test ./... -count=1                          → ok ×11 packages
$ go test ./internal/middleware/ -v                                                 → PASS ×11
$ go test ./internal/health/ -v                                                     → PASS ×6
$ go test ./internal/config/ -v                                                     → PASS ×7 (incl. DefaultsFrontendOrigin)
$ grep -n 'err.Error()' internal/health/health.go                                   → (no output)
$ grep MaxBytesReader (non-test)                                                    → bodylimit.go only (doc line + call)
$ grep -c FRONTEND_ORIGIN .env.example CODEMAP / spec                               → 1 / 2 / 1
$ git log origin/main..HEAD                                                         → 5 commits, all with Co-Authored-By
$ cli.py validate                                                                   → exit 0
$ gh run view 35958578061 (head 1875b89 = branch tip)                               → success: harness-tooling, frontend, backend-unit, backend-integration
```

**Runtime proof** (isolated: `COMPOSE_PROJECT_NAME=rev-edge`, pg 55443, redis 56393, API :18092, `FRONTEND_ORIGIN=https://app.example.com`; torn down with `down -v` afterwards):
```
boot log: cors: allowing [https://app.example.com]
OPTIONS quiz  Origin app, ACRM GET, ACRH authorization → 204; ACAO app; Allow-Methods GET, POST, OPTIONS;
                                                        Allow-Headers Authorization, Content-Type; Max-Age 600; Vary Origin
OPTIONS quests/progress (no GET route), ACRM POST     → 204
GET quiz Origin app (no token)                         → 401 with ACAO app + Vary Origin (browser can read the 401)
OPTIONS from evil.example | null | app.example.com.evil.com | evilapp.example.com | http://app… |
             https://app…:443 | HTTPS://APP.EXAMPLE.COM | https://app…/        → 403, no ACAO (all 8)
GET quiz Origin null                                   → 401, no ACAO, Vary Origin
GET /healthz, no Origin                                → 200, no CORS headers, no Vary
POST auth/google 70 000-byte body (Content-Length)     → 400 {"error":"invalid_request"}  (no Connection: close; see bug #1)
POST auth/google 70 000-byte body (chunked)            → 400
Guarded POSTs with a minted session: small / 60 KiB / 70 KB body
  quests/progress           500 / –   / 400     pet/revive               500 / –   / 400
  settings/notifications    404 user_not_found / 404 / 400
  onboarding/assessment     503 ai_unavailable / 503 / 400
  → the limit reaches every body-binding POST route, and a 60 KiB body still gets through to the handler
healthz up                                             → {"postgres":"ok","redis":"ok","status":"ok"}
healthz, postgres stopped                              → 503 {"postgres":"unavailable","redis":"ok","status":"unavailable"} in 4 ms
server log: health: postgres ping failed: failed to connect to `user=english database=english`: …
```
Side observation: the binary ignores SIGTERM (a plain `kill` left it serving). That is the known bug the approved shutdown plan fixes, not something this branch introduced.

## Quality
**Security (CORS).** No reflection holes. The match is an exact `map[string]bool` lookup on the raw `Origin`, with no prefix, suffix or case folding on the request side, and the eight lookalike origins above were all refused live. `null` cannot be allow-listed: `ParseOrigins("null")` has no host and is rejected. `*` is rejected, and `Allow-Credentials` is never sent. `Vary: Origin` goes on every response to a request that carries `Origin`, allowed or not. It is not sent when there is no `Origin`, which is what the plan tested. That is safe here: all the data routes need `Authorization`, which shared caches do not store, and the PWA's service-worker requests are cross-origin, so they always carry `Origin`. A disallowed origin's actual request still runs the handler (by design, and documented). That is not a CSRF path, because a cross-site simple request cannot attach the bearer header. Bug #2 is the config footgun: `ParseOrigins` accepts origins no browser sends (`:443`, `*.host`, trailing dot), so the boot check does not catch the silent outage it exists for. It also covers the thin lookalike-origin test coverage.

**Security (body limit).** The limit applies to every POST route: all five `ShouldBindJSON` call sites sit under `/api/v1`, and each maps a bind error to `400 invalid_request`. `pet/revive` binds only when `ContentLength != 0`; a chunked body reports -1, so it is still bound and still limited (proved live). `google/sync` never reads its body. `/healthz` is GET-only and outside the group. The CODEMAP note about registration order is accurate and matters: `Use` precedes both the routes and the `guarded` subgroup.

**healthz.** The design is clean. `ping` gives each dependency its own `WithTimeout` from the request context. Worst case is 4 s, which fits within Railway's probe timeout (the plan's reasoning). `PingTimeout` is an exported mutable package var only so tests can shrink it. That is a small idiom cost: a `Handler` option or an unexported var set from the test would keep the tunable internal. Not filed. `TestHealthzLogsTheDriverErrorServerSide` restores `log` output to `os.Stderr` rather than to the previous writer. That is harmless now, but it would clobber another test's redirect. Not filed.

**Test honesty.** The tests assert behaviour, not just that code runs. The CORS tests check exact header values and the absence of `Allow-Credentials` and `Access-Control-Allow-Origin`. The body test checks both the 400 and that the handler saw `request body too large` and decoded nothing. The health tests look for specific leak substrings. The executor reports all six of the plan's mutations turning red. The router in the tests mirrors `cmd/api`, and I confirmed the real wiring live on every route. Gap (in bug #2): only one disallowed origin is tested.

**Boundaries and conventions.** The new `middleware` package has no knowledge of routes, users or tables. `config` carries only the raw string. `main.go` gains exactly the three planned lines. No new dependency. The code matches the repo's style: plain `gin.HandlerFunc`s, and doc comments that cite the spec.

**Docs.** The CODEMAP `middleware` and `health` bullets and the `shell` cross-reference are accurate, and CODEMAP needs no correction. `.env.example` and spec §9 are updated in the document's escaped style. The one inaccurate statement is in the plan's *Notes* (the connection-close claim), filed in bug #1.

## Bugs filed
1. `harness/ideas/_inbox/a-slow-request-body-still-holds-its-goroutine-indefinitely-n.md` (medium, not a blocker): no plan sets `http.Server.ReadTimeout`, so a slow body still pins a goroutine, and `MaxBytesReader`'s connection-close hook never fires through `c.Writer`, contrary to the plan note.
2. `harness/ideas/_inbox/parseorigins-accepts-frontend-origin-entries-no-browser-send.md` (low, not a blocker): `:443`, `*.host` and trailing-dot entries pass boot validation but can never match. Only one lookalike-origin test exists. The redundant scheme condition remains.

## Verdict
**pass-with-bugs.** The head idea is fully delivered and verified live: the deployed PWA can now call the API cross-origin, with no origin-reflection hole. The healthz leak is closed, and body size is bounded on every POST route. Two non-blocking bugs are filed (medium: a slow-body read timeout that no plan owns; low: `ParseOrigins` config footguns and test coverage). There are no unresolved blockers (`cli.py blockers --plan` exits 0) and CI is green on the branch tip. Merge: `/harness merge harness/plans/2026-09-24-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md` (daily order per the plan: shutdown plan, then this plan, then the secrets plan).
