# Real-backend reading proof

Run from the reading worktree. Use a disposable Compose project: integration tests drop tables. The test helper refuses PostgreSQL/Redis ports other than 55432/56379 and always names project `reading-flow`. Keep the main development services on their existing ports. Do not run destructive integration tests while the browser fixture API is running.

```sh
# From repository root. Existing pinned Go install must be available here.
export PATH=/workspace/toolchains/go/bin:$PATH
export GOCACHE=/workspace/.cache/go-build
export GOPATH=/workspace/.cache/go
POSTGRES_PORT=55432 REDIS_PORT=56379 docker compose -p reading-flow -f backend/docker-compose.yml up -d --wait

# Integration suite first; all TestIntegration declarations must PASS, no SKIPs.
cd backend
env -u DATABASE_URL -u REDIS_URL TEST_DATABASE_URL='postgres://english:english@localhost:55432/english?sslmode=disable' TEST_REDIS_URL='redis://localhost:56379/0' make test-integration > /tmp/reading-flow-task4-integration.log 2>&1
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check > /tmp/reading-flow-task4-check.log 2>&1
go build -o /tmp/reading-flow-task4-api ./cmd/api
cd ../frontend
npm run lint
npm run typecheck
npm run test:unit
npm run build
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config playwright.reading.config.ts > /tmp/reading-flow-task4-playwright.log 2>&1
cd ..
python3 tools/harness/cli.py validate
python3 -m unittest discover -s tools/harness/tests
docker compose -p reading-flow -f backend/docker-compose.yml down
```

Omit `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` to use Playwright's installed Chromium (`npx playwright install chromium`). The browser uses viewport 390×844. Existing stub-based login tests use `playwright.config.ts`; this dedicated config runs only `reading.spec.ts`, enables the real service worker and starts frontend :3100 plus actual `cmd/api` :3199. Never reuse a previously running proof server.

The Node-only helper at `helpers/reading-server.mjs` starts a local OpenAI-compatible provider on :3299, seeds unique users/active-roadmaps/day-one reading outlines through `psql`, and signs ephemeral HS256 JWTs with the existing `sub`/`iat`/`exp` and 24-hour session contracts. It stores each JWT under `sess:{user_id}:token` in Redis and injects the frontend's normal persisted auth shape. There is no production auth bypass, intercepted daily/exercise/progress response or Google OAuth call. Fixture secrets exist only in process memory and are never printed. Repository migrations run on actual API boot. The helper kills only its own API process when Playwright stops it; Compose cleanup is the final command above. Fixture rows remain in the disposable volume for diagnosis until it is removed.

Tests prove lazy loading, lossless passage display before questions, Vietnamese cards, native keyboard activation/Escape/focus, ongoing timer ticks, question look-back continuity, all answers in the actual progress POST, persisted schema/content reuse with one provider call, SQL completion/minutes and Redis seconds, completion after reload, upstream failure/retry, and offline in-app re-entry backed by actual service-worker daily cache plus owner-scoped typed-content storage. A resumed timer anchor is deliberately set to 65 seconds before the current time so durable minutes are nonzero without sleeping one minute. No browser clock is mocked; the posted duration is checked against Redis and its floored minutes against SQL. Answers are accepted but currently not saved or scored.

Evidence is outside tracked files: `/tmp/reading-flow-task4-{api,provider,playwright}.log`, `/tmp/reading-flow-task4-gloss-390x844.png`, `/tmp/reading-flow-task4-db-evidence.json` (SQL/Redis/POST assertions without tokens), and `/tmp/reading-flow-playwright/` (runner output). The proof also attaches its JSON to Playwright's test result for reporters that retain attachments. Preserve command exit codes when capturing logs. Check declared integration count against `--- PASS: TestIntegration` and reject any `--- SKIP`.

This deterministic compatible provider proves the application transport/persistence path, not live AI latency, quality or token feasibility. Genuine 9Router proof is separate and requires securely injected `NINE_ROUTER_API_KEY`; use it through supported `OPENAI_API_KEY`/`OPENAI_BASE_URL=https://9router.huyndg.io.vn/v1`/`OPENAI_MODEL=English-bot-free` configuration. Inspect credential presence only, never echo or put the key in commands, files or logs. Record genuine generation duration/token usage/schema and a second persisted cached read only when executed. No genuine-provider claim follows from fixture token counts.

Offline full-document reload is not covered as passing: the existing service worker has no navigation fallback and the server build has no precached HTML. The initial reload experiment returned `ERR_INTERNET_DISCONNECTED`; the controller scoped this task to reading cache re-entry, so no unrelated app-shell change was added. The final test reads the actual cached daily response while offline and re-enters reading through Nuxt links.
