---
plan: harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/deploy-compose-yml-drops-the-ai-provider-base-url-and-model-.md, harness/ideas/_inbox/caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md, harness/ideas/_inbox/smoke-api-sh-stops-at-the-first-unreachable-check-instead-of.md, harness/ideas/_inbox/web-image-runs-caddy-as-root-and-reinstalls-npm-deps-on-ever.md]
---
# Review — Containerised deploy: Dockerfiles, production compose, runbook and CI image build

**Plan:** `harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md`
**Branch/worktree:** `harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook-` / `/Users/hendrixnguyen/Workspaces/self/Learning-English-Project/.worktrees/containerised-deploy-dockerfiles-production-compose-runbook-`
**Diff:** `git diff main...harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook- --stat`

**Reviewed at:** `c46b1df` (origin branch head). The plan worktree had an untracked `.infisical.json`, so I left it untouched and reviewed in a fresh detached worktree in the reviewer scratchpad. Docker project `aelp-rev-deploy`, ports 28080/28081. Everything was torn down with `down -v`; no containers or volumes remain.

## Plan vs idea
The plan delivers the idea's technical Expected output: both Dockerfiles, the production compose, the runbook (env table, Railway Free, Dokploy, owner checklist, smoke check), the CI `docker-images` job, and the doc updates. The plan makes three deliberate corrections to the idea, and all three are justified and checked against the code:
- There is no `GET /api/v1/auth/google` redirect. The smoke check posts `{}` and expects 400.
- The OAuth redirect is `<origin>/login`, not `/auth/callback`.
- The final image is alpine, not distroless, so the `HEALTHCHECK` has a shell and `wget`.

Wiring the smoke check into the 20:00 routine was dropped in the idea's own `## Evaluation`. Registry push and CD are parked in `harness/BACKLOG.md`. The user-visible line (a public PWA) was the owner's checklist; the owner already deployed it by hand on 2026-09-25.

There is one gap against the idea's "one env table (every variable)". The six `*_BASE_URL` / `*_MODEL` variables that `internal/config` reads are missing, and the live OpenRouter setup depends on two of them (bug 1).

## Code vs plan
All seven tasks were followed. The executor's three deviations are justified:
1. `for _ in` satisfies actionlint SC2034.
2. Compose names the first missing `:?` variable in a nondeterministic order. My run named `POSTGRES_PASSWORD`.
3. Compose v2.39 expands the port syntax in `config` output.

No file under `backend/internal`, `backend/cmd` or the frontend source changed; the diff adds only new files plus doc and CI edits (16 files, +433/−3).

Re-run verification (my output):
```
1. docker build backend  -> ok (aelp-api:rev 55MB); docker build frontend (API_BASE build arg) -> ok (aelp-web:rev 86.3MB)
   inspect: api map[8080/tcp:{}] [CMD-SHELL wget -qO- "http://127.0.0.1:${PORT}/healthz" || exit 1]
2. docker run --rm aelp-api:rev -> "config: DATABASE_URL is required", exit=1
3. rm deploy/.env; compose config -q -> "required variable POSTGRES_PASSWORD is missing a value…", exit=1
   scratch deploy/.env (ignored: `git status --ignored` shows `!! deploy/.env`) -> config ok
4. up -d --build --wait: api/postgres/redis/web all "Up (healthy)"
   logs: migrations applied: [0001_init 0002_google_sync 0003_pet_verdict_dates] / cors: allowing [http://localhost:28081] / listening on [::]:8080 (GIN_MODE=release)
   smoke-api.sh: 6x ok (healthz 200 + body, auth/google 400, preflight 204 + allow-origin, foreign 403) exit=0
   smoke-web.sh: 7x ok exit=0
5. redis-cli config get appendonly -> appendonly yes
6. down -v -> no aelp-rev containers/volumes left; deploy/.env removed
7. actionlint ok; CI "Production compose file is valid" block run locally -> exit 0 (and the no-env guard fails as required)
9. gh run 36096793213 (headSha c46b1df = branch head): success — backend-unit, backend-integration, harness-tooling, frontend, docker-images all green
```
For item 8 (the existing suites) I relied on the green CI run on the same SHA and did not re-run them locally. No backend or frontend source changed.

Nothing failed to reproduce. The executor gate holds.

## Quality
- **Correctness under untried inputs:**
  - A missing `/_nuxt/*.js` returns `200 text/html` with a one-year `immutable` header. `index.html` is served with no `Cache-Control`. That is a stale-chunk hazard after every redeploy on the Dokploy target (bug 2).
  - The web smoke script run read-only against the live Pages site fails on the deep-link check (HTTP 404). The runbook's "Pages does the SPA fallback by itself" does not hold when `nuxi generate` emits `404.html`, so owner checklist step 8 fails. This duplicates existing inbox items, listed under *Bugs filed*.
- **Env contract:** compose only forwards the keys it lists, so `OPENAI_BASE_URL`/`OPENAI_MODEL` are silently dropped. I reproduced this with `docker compose exec api env` (bug 1).
- **Failure modes:** `smoke-api.sh` aborts at the first unreachable curl because of `set -e` on the assignment. It exits 7 with no FAIL lines, unlike `smoke-web.sh` (bug 4). Neither script can falsely pass: `smoke-web.sh` against nothing exits 1.
- **Security and resource use:** the web image runs as root, and `COPY . .` before `npm ci` defeats layer caching (bug 5). The API image is good: static, non-root, ca-certificates only, and the healthcheck reflects the dependencies.
- **Test honesty:** the CI `docker-images` job asserts real behaviour: the binary boots to a config error, the build arg is baked in, the cache headers are right, and the compose guards fail without an env file. It does not cover the Caddy fallback edge (bug 2 asks for a check).
- **Docs and CODEMAP:** accurate apart from bug 1 and the Pages paragraph (already filed). The AGENTS.md CI bullet now lists all five jobs. No CODEMAP correction needed.
- **Live impact once merged (note, not a bug):** the Railway service is rooted at `backend/`, so it will pick up `backend/Dockerfile` on the next redeploy from `main` instead of its current builder. `PORT` injection and `/healthz` fit that. The owner should watch the first redeploy.
- The diff is mostly config, shell and docs (about 300 lines of Dockerfile/YAML/sh), so I did not spawn a separate test-gap agent. I probed the smoke scripts' edge cases by hand instead.

## Bugs filed
1. `harness/ideas/_inbox/deploy-compose-yml-drops-the-ai-provider-base-url-and-model-.md`: medium. compose/.env.example/runbook omit `*_BASE_URL`/`*_MODEL`, and the live OpenRouter setup needs them.
2. `harness/ideas/_inbox/caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md`: medium. Missing hashed assets fall back to HTML with an immutable header, and `index.html` has no Cache-Control.
3. `harness/ideas/_inbox/smoke-api-sh-stops-at-the-first-unreachable-check-instead-of.md`: low.
4. `harness/ideas/_inbox/web-image-runs-caddy-as-root-and-reinstalls-npm-deps-on-ever.md`: low.

Not re-filed, because they already exist: the Pages deep-link 404 is `harness/ideas/_inbox/pages-ignores-the-redirects-spa-rewrite-while-404-html-exist.md`, already on `main`. The runbook's stale "Pages needs no `_headers`" paragraph and the missing `/login` smoke check are `harness/ideas/_inbox/task-4-follow-up-smoke-web-has-no-login-return-trip-check-an.md`, filed today by the review of `2026-09-25-nobody-can-sign-in-on-cloudflare-pages-login-is-308-redirect`. My read-only live run of `smoke-web.sh` (FAIL on `/learn/abc`, 404) is further evidence for the first.

None are blockers. The branch adds no app-code change, every check passes, and none of these regresses anything already on `main`.

## Verdict
**pass-with-bugs.** Plan and idea are delivered, the executor's runtime proof reproduces in full, and CI is green on the branch head. Two medium and two low bugs are filed to the inbox. Merge: `/harness merge harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md` (via the daily PR).
