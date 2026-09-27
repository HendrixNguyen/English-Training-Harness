---
plan: harness/plans/2026-09-26-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md
verdict: fail
bugs: [harness/ideas/_inbox/session-renewal-signs-the-learner-out-when-a-non-2xx-or-a-pa.md, harness/ideas/_inbox/sw-header-strip-comment-and-codemap-say-a-cached-token-is-ne.md]
---
# Review — Stay signed in: `auth.Require` renews a session below half-life and the PWA adopts the new token silently; an expired session says why on `/login`

**Plan:** `harness/plans/2026-09-26-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md`
**Branch/worktree:** `harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve` / `.worktrees/stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve`
**Diff:** `git diff main...harness/2026-09-26-high-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve --stat`

## Plan vs idea
**Not delivered.** The idea's first user-visible line is "A learner who opens the app at least once every 24 hours is never asked to sign in again". Its concurrency note requires that "a `401` immediately after a renewal is retried once with the newest stored token before signing out". The branch builds every piece the plan lists, but the server and the client disagree about when a session rotates:
- **Deterministic.** The server renews on every status (the header is set in `Require` before the handler), while the client adopts it only on a 2xx. A 404/409/5xx at renewal time leaves the client holding a revoked token, and the next request signs out.
- **Intermittent.** The hub always sends `pet/status` and `quests/daily` in parallel. One renews, and the other's `Get` sees the new token and returns 401. That 401 usually returns before the renewing response, so the one-retry rule never fires.

Reproduced in a real browser: the first hub load past half-life ended on `/login` with `aelp.auth` cleared. So the feature moves the forced sign-out from 24 h to about 12 h for affected learners instead of removing it. Everything else the idea asked for exists: the `/login?reason=expired` copy, unchanged revocation and supersession, the CORS expose, the tests, and the spec and CODEMAP.

## Code vs plan
Reviewed at origin head `60eed1a` in a detached reviewer worktree, with merge base `67ad0c0`. The diff is 22 files, +628/−15.
- Tasks 1–5 were followed. The three deviations are justified:
  - `apiStateCache.ts` split out for testability;
  - one test function with half-life subtests;
  - an own fake store to isolate a failing `Put`.
- Merge: against `origin/main` only `harness/CODEMAP.md` conflicts. Against the auth-5xx branch, `backend/internal/auth/middleware.go` also conflicts, as the plan predicted. The renewal call goes after the 503 branch.

Verification, re-run by the reviewer:
```
backend: go build ./... && gofmt -l . && go vet ./...   -> build-ok, gofmt silent
env -u … go test -timeout 120s ./... -count=1 -race    -> all packages ok
go test ./internal/auth ./internal/middleware -run 'TestRequire|TestCORS' -v
  12 TestRequire* PASS (incl. the 6 renewal tests), TestCORSExposesTheSessionHeaders PASS
integration (rv-auth stack): go test ./... -run Integration -p 1 -count=1   -> all ok
frontend: npm ci; npm run lint (clean); npm run typecheck (clean)
npm run test:unit      -> Test Files 18 passed, Tests 92 passed
npm run build          -> ✨ Build complete
npx playwright test tests/e2e/login.spec.ts -> 5 passed
gh run list --branch <branch> --limit 1 -> completed success (run 36227582887)
```
The executor's evidence reproduces, so there is no executor gate failure. The defect is in the design of the renewal handshake, and the tests do not cover it: the unit test `ignores X-Session-Token on an error response` pins the mismatched behaviour.

Runtime, re-run by the reviewer. The API was built from the branch on the isolated stack, with a token minted with `RenewBelow + 1m` elapsed and seeded into Redis:
- `GET /quests/daily` for a learner without a roadmap → `404 {"error":"no_active_roadmap"}` **with** `X-Session-Token`. The same old token on `GET /pet/status` then → `401`.
- Two concurrent curls (`pet/status` and `quests/daily`, like `pages/index.vue:11`), 15 trials: in `trial 1` pet got `401` in 5 ms and the renewing quests response came back at 11 ms. The other 14 trials had both requests renew within one second, which gives byte-identical tokens and no failure.
- Real browser (built Nuxt on :3614 → the API): `/login?reason=expired` shows the `⏳` card between the tagline and the Google button. The text is exactly the §7 sentence, with no `aria-live`/`role` and the glyph `aria-hidden`. `/login`, `?reason=foo`, `?reason=` and a repeated param show no card. A stale `aelp.auth` with `/` → `/login?reason=expired`, storage cleared. `/roadmap` with no token → `/login`, no card. The sign-out paths (`AppHeader`, `useApi` 401) navigate to plain `/login`.
- Seeding the below-half-life token and loading `/` gave network `quests/daily → 401` and `pet/status → 500` (aborted by the navigation). The page ended on `/login`: **signed out**.

Design §8 acceptance:
- Silent adoption on 2xx: ✓ (unit).
- Ignored on non-2xx: ✓ as designed, but that design choice is the blocker.
- One retry: ✓ only when the renewal response arrives first. ✗ in the observed ordering.
- Redirect with `reason=expired`: ✓.
- Card placement and copy: ✓.
- No notice after sign-out: ✓.
- SW strip: ✓ (unit).
- Named tests exist and pass: ✓.

UI-KIT: v1 `AppCard` as the design decided, and no `StateBlock`. No kit violation.

## Quality
- Boundaries: `middleware` uses a string literal instead of importing `auth`, and `Require` has a single call site. Clean.
- Correctness: the rotation handshake has the two failure modes above (blocker). Also, same-second concurrent renewals are only safe by accident, because `Issue` has no `jti` and so produces identical tokens. Tokens issued a second apart diverge, and the client adopts whichever response arrives last.
- Test honesty: the backend renewal tests are sequential only. There is no concurrent test and no non-2xx-status test, and the frontend test enshrines the mismatch.
- Docs: the SW strip comment and CODEMAP call the cached token "newer" when it is older (low).
- Performance: one `SET` per half-life per learner, as planned.

## Bugs filed
- **BLOCKER** `harness/ideas/_inbox/session-renewal-signs-the-learner-out-when-a-non-2xx-or-a-pa.md` (high, `blocks` this plan): a non-2xx rotation is ignored by the client, and the parallel-request race returns a 401 before the renewal response. Either way the learner is signed out at ≥12 h.
- `harness/ideas/_inbox/sw-header-strip-comment-and-codemap-say-a-cached-token-is-ne.md` (low): the comment/CODEMAP direction is wrong.

## Verdict
**fail.** The plan was executed faithfully and CI is green, but the idea's core promise does not hold. The first hub load past half-life can sign the learner out, and any non-2xx at renewal time always does, which is earlier than today's 24 h. Keep this branch out of today's daily PR until the blocker lands on it (`amends:`).
