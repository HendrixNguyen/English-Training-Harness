---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md
---
# Caddyfile serves index.html with a one-year immutable cache for missing /_nuxt assets

## Why
`frontend/Caddyfile` has `header /_nuxt/* Cache-Control "public, max-age=31536000, immutable"` followed by `try_files {path} /index.html`. A `/_nuxt/` path that does not exist therefore gets `200 text/html` with a one-year immutable header. After a redeploy, a client still holding the old `index.html` (or an old service-worker precache list) requests the old chunk names. It receives HTML with status 200 and fails with a MIME-type/module error, and the browser caches that HTML for a year under the chunk URL. Separately, `index.html` itself has no `Cache-Control` header (only ETag/Last-Modified), so browsers may cache it heuristically. That makes stale-shell-after-deploy more likely. This affects the Dokploy target, where this image is what users hit.

## Expected output
- A request under `/_nuxt/` for a file that does not exist returns `404`, and is not rewritten to `index.html`. For example, scope `try_files` to non-`/_nuxt` paths, or add a `handle /_nuxt/*` with `file_server` and no fallback.
- The immutable header applies only to files that exist.
- `index.html` (and the SPA-fallback responses) are served `Cache-Control: no-cache`.
- `deploy/smoke-web.sh` gains two checks: a made-up `/_nuxt/does-not-exist.js` is 404, and `/` is `no-cache`.

## Evidence
- Plan: `harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md` (branch origin/harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook- @ c46b1df).
- `frontend/Caddyfile` lines 15-19.
- Reproduced on the image built from the branch (`aelp-web:rev`, `-p 28081:80`). `curl -i http://127.0.0.1:28081/_nuxt/old-deleted-chunk.js` returned `HTTP/1.1 200 OK`, `Cache-Control: public, max-age=31536000, immutable` and `Content-Type: text/html; charset=utf-8`. `curl -I /` has no Cache-Control line, only `Etag` and `Last-Modified`.

## Evaluation
_Evaluator, 2026-09-26 — daily decide (bug queue, ranked #5; head of a three-item Dokploy plan)._

**Select — medium.** *Is the Why real?* Yes, reproduced by the reviewer on the built image: a missing `/_nuxt/` chunk answers `200 text/html` with a one-year immutable header, and `index.html` has no `Cache-Control` — a client holding an old shell after a redeploy caches HTML under a chunk URL for a year. It only bites the Dokploy target (Pages has its own `_headers`), so it ranks below the live-site items, but the trap is permanent once hit. *Root cause:* `frontend/Caddyfile` applies the `header /_nuxt/*` matcher before `try_files` rewrites the miss to `/index.html`. *Fix:* a `handle /_nuxt/*` block with `file_server` and no fallback; `Cache-Control: no-cache` on the shell. Folded with `deploy-compose-yml-drops-the-ai-provider-base-url-and-model-` (same target, same runbook table; the live OpenRouter setup would silently break on Dokploy) and `smoke-api-sh-stops-at-the-first-unreachable-check-instead-of` (three-line fix in the same `deploy/` set).
