---
idea: harness/ideas/_inbox/caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md
status: done
priority: medium
merged: false
branch: harness/2026-09-26-medium-caddyfile-serves-index-html-with-a-one-year-immutable-cache-
worktree: .worktrees/caddyfile-serves-index-html-with-a-one-year-immutable-cache-
---
# Dokploy target hardening: Caddy answers 404 for a missing chunk and `no-cache` for the shell, compose passes the AI base-URL/model variables, smoke-api reports every check — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B5** of 2026-09-26. **Estimate:** 2.5 h. **Branch:** `harness/2026-09-26-medium-caddyfile-serves-index-html-with-a-one-year-immutable-cache-`.

**Idea (head):** `harness/ideas/_inbox/caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md`
**Also planned here (their frontmatter points at this plan):**
- `harness/ideas/_inbox/deploy-compose-yml-drops-the-ai-provider-base-url-and-model-.md` → Task 2
- `harness/ideas/_inbox/smoke-api-sh-stops-at-the-first-unreachable-check-instead-of.md` → Task 3

**Goal:** On the Dokploy target a request for a `/_nuxt/` file that no longer exists is a `404` (never HTML with a one-year header), the app shell is `Cache-Control: no-cache`, the six `*_BASE_URL`/`*_MODEL` variables the live deployment depends on reach the `api` container, and `deploy/smoke-api.sh` prints an `ok`/`FAIL` line for every check even when the host is down.

**Architecture:** `frontend/Caddyfile`, `frontend/public/_headers` (one rule so Pages matches), `deploy/compose.yml`, `deploy/.env.example`, `deploy/README.md`, `deploy/smoke-*.sh`. No app code; the Caddyfile is a server config, not a screen — **no design doc**.

**⚠ Same-day conflict note:** ticket B4 (`harness/plans/2026-09-26-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-.md`) edits the middle of `deploy/smoke-web.sh` (SPA block, login check) and the runbook's Target A Pages paragraph + *Smoke check* bullet. This plan **appends** its two `smoke-web.sh` checks at the end (before `exit $fail`) and edits only the *Environment* table, the Supabase-free Target B paragraph and `smoke-api.sh`. If `origin/main` already has B4, merge it first and keep both sides.

## Global Constraints
- `rg`/`timeout` not installed: `grep -n`, `curl --max-time`, `docker compose --wait-timeout`. Docker Compose in a worktree: `COMPOSE_PROJECT_NAME=<slug>` (or `-p <slug>`) and free host ports; `make down`/`compose down` the same project after.
- Scripts stay POSIX `sh`; `set -eu` stays — the fix is `|| true` on the curl assignments, not dropping `-e`.
- Pages behaviour must not regress: `_headers` gains only the `/` + `/index.html` `no-cache` rule; the missing-asset `404` check is **Caddy-only** (Pages' implicit SPA mode answers 200 for any miss) and is therefore guarded by `SMOKE_WEB_ASSET_404=1`, which only the `docker-images` CI job and the local Docker check set.

## Review Focus
1. Caddy: `handle /_nuxt/* { header Cache-Control "public, max-age=31536000, immutable"; file_server }` — inside a `handle`, a miss falls to Caddy's default 404 with no `try_files`; the immutable header is set only when the file is served (verify with `curl -i` on a missing chunk: `404`, no `Cache-Control: … immutable`).
2. `index.html` and every SPA-fallback response carry `Cache-Control: no-cache` (`curl -I /` and `curl -I /learn/abc`); `sw.js`/manifest rules unchanged.
3. `docker compose -f deploy/compose.yml config` shows all six variables under `api.environment`; `docker compose exec api env | grep -c 'OPENAI_BASE_URL\|OPENAI_MODEL'` → `2` when set in `deploy/.env`.
4. `VAPID_SUBJECT` no longer defaults to `mailto:admin@example.com` in compose or `.env.example` (blank; `config.DefaultVAPIDSubject` still applies in the binary — that default and its comment are out of scope here; note it in the runbook row).
5. `deploy/smoke-api.sh http://127.0.0.1:1 http://x` prints five `FAIL` lines and exits `1` — no bare `curl: (7)` abort.

## File structure

| Path | Change |
| --- | --- |
| `frontend/Caddyfile` | `handle /_nuxt/*` with `file_server`, no fallback; `header` `no-cache` for `/` and the fallback; `handle` for everything else with `try_files` |
| `frontend/public/_headers` | `/` and `/index.html` → `Cache-Control: no-cache` |
| `deploy/smoke-web.sh` | appended: `/` is `no-cache`; guarded missing-asset `404` |
| `deploy/compose.yml` | `api.environment` gains the six `${VAR:-}`; `VAPID_SUBJECT: ${VAPID_SUBJECT:-}` |
| `deploy/.env.example` | six blank vars with the OpenRouter note; `VAPID_SUBJECT=` blank |
| `deploy/README.md` | *Environment* table: six rows + `VAPID_SUBJECT` note; Target B one sentence |
| `deploy/smoke-api.sh` | `|| true` on the curl assignments |
| `.github/workflows/ci.yml` | `docker-images` job: `SMOKE_WEB_ASSET_404=1` on its `smoke-web.sh` call |
| `harness/CODEMAP.md` | Deploy bullet |

## Tasks

### Task 1: Caddyfile and the shell headers

**Files:** `frontend/Caddyfile`, `frontend/public/_headers`, `deploy/smoke-web.sh`, `.github/workflows/ci.yml`.

- [ ] **Step 1 (checks first):** append to `deploy/smoke-web.sh` before `exit $fail`:
```sh
# The app shell must never be cached by heuristics (a redeploy would strand an old index.html).
check "index cache-control" "no-cache" "$(cache_control "$web/")"
# Caddy only (SMOKE_WEB_ASSET_404=1): a chunk that no longer exists is a 404, never index.html
# with a one-year header. Pages' implicit SPA mode answers 200 for any miss, so it is not checked there.
[ "${SMOKE_WEB_ASSET_404:-}" = "1" ] && check "missing asset is 404 (/_nuxt/does-not-exist.js)" "404" "$(status "$web/_nuxt/does-not-exist.js")"
```
  (`[ … ] && check …` under `set -e`: a false test on the last line would exit 1 — put it as `if [ … ]; then check …; fi`.)
- [ ] **Step 2:** Rewrite the `Caddyfile` site block:
```caddyfile
:80 {
	root * /srv
	encode gzip zstd
	respond /healthz 200

	@fresh path /sw.js /manifest.webmanifest
	header @fresh Cache-Control "no-cache"

	# Content-hashed assets: immutable when they exist, a plain 404 when they do not
	# (never index.html under a chunk URL — a stale shell would cache HTML for a year).
	handle /_nuxt/* {
		header Cache-Control "public, max-age=31536000, immutable"
		file_server
	}

	# Everything else: the SPA shell, never cached by heuristics.
	handle {
		header Cache-Control "no-cache"
		try_files {path} /index.html
		file_server
	}
}
```
  Check Caddy's `header` semantics for a 404 inside `handle /_nuxt/*` (the header directive runs before `file_server`, so a 404 could still carry the immutable header): if `curl -i` on a missing chunk shows the immutable header, use `header @exists` with `@exists file` (or `route { header … ; file_server }` with `@missing not file` → `respond 404`) — the test in Step 1 is the arbiter.
- [ ] **Step 3:** `_headers`: add
```
/
  Cache-Control: no-cache

/index.html
  Cache-Control: no-cache
```
- [ ] **Step 4:** `ci.yml` `docker-images` job: `SMOKE_WEB_ASSET_404=1 deploy/smoke-web.sh http://127.0.0.1:18081`.
- [ ] **Step 5:** Local proof: `docker build -t aelp-web:b5 --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend && docker run -d --rm --name web-b5 -p 18083:80 aelp-web:b5 && sleep 2 && SMOKE_WEB_ASSET_404=1 deploy/smoke-web.sh http://127.0.0.1:18083; curl -i http://127.0.0.1:18083/_nuxt/old-deleted-chunk.js | head -8; curl -I http://127.0.0.1:18083/ | grep -i cache-control; docker stop web-b5`. Commit: `web: Caddy answers 404 for a missing /_nuxt chunk and no-cache for the shell; Pages _headers matches`.

### Task 2: The six AI variables through compose and the runbook

**Files:** `deploy/compose.yml`, `deploy/.env.example`, `deploy/README.md`.

- [ ] **Step 1:** `compose.yml` `api.environment`: after `DEEPSEEK_API_KEY`, add `GEMINI_BASE_URL: ${GEMINI_BASE_URL:-}`, `OPENAI_BASE_URL: ${OPENAI_BASE_URL:-}`, `DEEPSEEK_BASE_URL: ${DEEPSEEK_BASE_URL:-}`, `GEMINI_MODEL: ${GEMINI_MODEL:-}`, `OPENAI_MODEL: ${OPENAI_MODEL:-}`, `DEEPSEEK_MODEL: ${DEEPSEEK_MODEL:-}`; change `VAPID_SUBJECT: ${VAPID_SUBJECT:-}`.
- [ ] **Step 2:** `.env.example`: under the AI providers block add the six, blank, with the comment `# Base URL + model per provider; blank = the provider's own default. The OPENAI_* pair works for any OpenAI-compatible endpoint (the live deployment runs OpenRouter there).` `VAPID_SUBJECT=` blank with `# a mailto:/https: contact; required in production when the VAPID keys are set`.
- [ ] **Step 3:** `README.md` *Environment* table: six rows (`optional | provider base URL / model | set on the service if used | set in deploy/.env if used`) and the `VAPID_SUBJECT` row's note: "the binary falls back to `config.DefaultVAPIDSubject` when unset — set a real contact in production". Target B paragraph: one sentence that `deploy/.env` must carry the same `OPENAI_BASE_URL`/`OPENAI_MODEL` the Railway service has.
- [ ] **Step 4:** Proof: scratch `deploy/.env` (copy of `.env.example` with the `:?` values filled and `OPENAI_BASE_URL=https://openrouter.ai/api/v1`, `OPENAI_MODEL=x`): `docker compose -p <slug> -f deploy/compose.yml config | grep -c 'OPENAI_BASE_URL\|OPENAI_MODEL\|GEMINI_BASE_URL\|DEEPSEEK_MODEL'` → 4 lines at least; optionally `up -d api --wait --wait-timeout 60` and `exec api env | grep OPENAI_`; `down`. Delete the scratch env. Commit: `deploy: compose and the runbook carry the AI base-URL/model variables; VAPID_SUBJECT has no fake default`.

### Task 3: `smoke-api.sh` reports every check

**Files:** `deploy/smoke-api.sh`, `harness/CODEMAP.md`.

- [ ] **Step 1:** Append `|| true` to the four `$(curl …)` assignments (`body=`, `code=` ×2, `hdr=`), so an unreachable host yields empty values and five `FAIL` lines; keep `set -eu` and `exit $fail`.
- [ ] **Step 2:** `deploy/smoke-api.sh http://127.0.0.1:1 http://x; echo exit=$?` → five `FAIL …` lines, `exit=1`; against the local API (`make up` + `go run ./cmd/api` or the compose stack) → five `ok`.
- [ ] **Step 3:** CODEMAP Deploy bullet: Caddy 404/no-cache rule, the six variables, `SMOKE_WEB_ASSET_404`. Commit: `deploy: smoke-api prints a FAIL line per check when the API is unreachable`.

## Verification
```
deploy/smoke-api.sh http://127.0.0.1:1 http://x; echo "exit=$?"                                   # 5 FAIL lines, exit=1
docker build -t aelp-web:b5 --build-arg NUXT_PUBLIC_API_BASE=https://api.example.test frontend
docker run -d --rm --name web-b5 -p 18083:80 aelp-web:b5 && sleep 2
SMOKE_WEB_ASSET_404=1 deploy/smoke-web.sh http://127.0.0.1:18083; echo "exit=$?"                # every line ok, exit=0
curl -sI http://127.0.0.1:18083/_nuxt/old-deleted-chunk.js | head -3                             # 404, no immutable header
curl -sI http://127.0.0.1:18083/ | grep -i '^cache-control'                                       # no-cache
docker stop web-b5
docker compose -p <slug> -f deploy/compose.yml --env-file <scratch .env> config | grep -E 'OPENAI_BASE_URL|OPENAI_MODEL|GEMINI_BASE_URL|GEMINI_MODEL|DEEPSEEK_BASE_URL|DEEPSEEK_MODEL|VAPID_SUBJECT'
grep -n 'BASE_URL\|_MODEL' deploy/.env.example deploy/README.md | wc -l                          # ≥ 12
git push -u origin harness/2026-09-26-medium-caddyfile-serves-index-html-with-a-one-year-immutable-cache-   # CI green incl. docker-images
```

## Execution summary

Built exactly the three tasks named in the plan; no deviations from the File structure table (`git diff --stat origin/main..HEAD` matches it file-for-file: `.github/workflows/ci.yml`, `deploy/.env.example`, `deploy/README.md`, `deploy/compose.yml`, `deploy/smoke-api.sh`, `deploy/smoke-web.sh`, `frontend/Caddyfile`, `frontend/public/_headers`, `harness/CODEMAP.md`).

- **Task 1 (Caddy 404 + no-cache shell):** `header @exists Cache-Control …` gated on an `@exists file` matcher inside `handle /_nuxt/*` (no `try_files` fallback in that block) was enough on the first try — a `curl -i` on a missing chunk came back `404` with no `Cache-Control` header at all, so the plan's fallback option (`route { header …; file_server }` + `@missing not file` → `respond 404`) was not needed.
- **Task 2 (AI vars + VAPID_SUBJECT):** as specified — six `${VAR:-}` entries added to `compose.yml` after `DEEPSEEK_API_KEY`, `VAPID_SUBJECT: ${VAPID_SUBJECT:-}` (was `:-mailto:admin@example.com`), matching `.env.example` and README rows, README's Target B paragraph gained the one sentence about `OPENAI_BASE_URL`/`OPENAI_MODEL` matching Railway.
- **Task 3 (smoke-api.sh):** `|| true` appended to all four curl-assignment lines; no other line changed.

**Runtime proof (in the worktree, all torn down afterward):**
- `deploy/smoke-api.sh http://127.0.0.1:1 http://x` → 5 `FAIL` lines, `exit=1` (was a bare `curl: (7) … exit=7` before the fix — confirmed red on the unpatched script first).
- Built `aelp-web:b5-red` from the unpatched Caddyfile first: `curl -i` on `/_nuxt/old-deleted-chunk.js` came back `200` with `Cache-Control: public, max-age=31536000, immutable`, and `curl -I /` had no `Cache-Control` header at all — red, reproducing both bugs.
- Rebuilt as `aelp-web:b5` with the fixed Caddyfile: `SMOKE_WEB_ASSET_404=1 deploy/smoke-web.sh http://127.0.0.1:18083` → all 9 checks `ok`, `exit=0`; `curl -i` on the missing chunk → `404` with no immutable header; `curl -I /` and `curl -I /learn/abc` → `Cache-Control: no-cache`.
- `docker compose -p caddy-harden -f deploy/compose.yml config` with a scratch `.env` (`OPENAI_BASE_URL=https://openrouter.ai/api/v1`, `OPENAI_MODEL=x`, rest blank) showed all six variables plus `VAPID_SUBJECT: ""` under `api.environment`.
- Full compose stack (`COMPOSE_PROJECT_NAME=caddy-harden`, `API_PORT=18182 WEB_PORT=18183`) brought up with `--wait --wait-timeout 240`: `deploy/smoke-api.sh http://127.0.0.1:18182 http://localhost:18183` → 6 `ok` lines against a real, reachable API; `SMOKE_WEB_ASSET_404=1 deploy/smoke-web.sh http://127.0.0.1:18183` → 9 `ok` lines. Torn down with `docker compose down -v`; scratch `deploy/.env` deleted; `docker ps -a --filter name=caddy-harden` empty afterward.
- `grep -n 'BASE_URL\|_MODEL' deploy/.env.example deploy/README.md | wc -l` → 15 (≥ 12 required).
- `python3 -m unittest discover -s tools/harness/tests` → 38 tests, OK (unaffected by this plan, run as a sanity check since `harness/CODEMAP.md` was touched).
- Pushed `harness/2026-09-26-medium-caddyfile-serves-index-html-with-a-one-year-immutable-cache-`; CI run [36216572852](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36216572852) — **success** (`docker-images`, `backend-unit`, `frontend`, `harness-tooling`, `backend-integration` all green).

**Deviations:** none from the plan's tasks or file list.

**Overlap with the deploy-smoke branch (B4, `harness/2026-09-26-high-deploy-smoke-checks-the-pwa-seconds-after-upload-when-pages-`, not yet in `origin/main`):** confirmed at start that `origin/main`'s `deploy/smoke-web.sh` still had the `SMOKE_WEB_SPA_WARN` knob (B4 not merged yet), so this plan's guidance to merge B4 first did not apply. This branch only *appended* two new checks to `smoke-web.sh` (`index cache-control`, the guarded `missing asset is 404`) right before `exit $fail`, and only touched `deploy/README.md`'s *Environment* table and the Target B paragraph — never the SPA block, the login check, or the Target A Pages paragraph that B4 edits. `.github/workflows/ci.yml`'s `docker-images` job line was also touched here (added `SMOKE_WEB_ASSET_404=1`); B4 should not need that same line, but the daily integration merge should check it doesn't collide.

Nothing left for the owner beyond the standard daily-PR merge; no production secrets or deploys were touched.
