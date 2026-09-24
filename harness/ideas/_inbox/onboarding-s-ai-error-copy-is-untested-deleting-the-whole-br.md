---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md
---
# Onboarding's ai_* error copy is untested — deleting the whole branch leaves the suite green

## Why
The onboarding-stub plan's stated Test approach was that `/onboarding` renders *honest* copy per backend error code. `tests/unit/onboardingPage.test.ts` guards exactly one of the two mapped codes: `rate_limited`. The `ai_*` branch is unguarded, and it is the branch that fires most often in practice — the merged backend answers `503 ai_unavailable` for *any* deployment with no provider API key set (`cmd/api/main.go` logs "no provider API keys set; AI-backed routes will answer 503"), and `502 ai_bad_output` / `502 ai_upstream_failed` for a malformed or failing model response. A first Railway deploy that forgets `GEMINI_API_KEY` puts every new user on that path.

Mutation run (reviewer, on the branch): deleting line `pages/onboarding.vue:15` — the entire `e.code.startsWith('ai_')` branch, so all three AI codes collapse to the generic "Không tạo được lộ trình. Thử lại." — leaves `npx vitest run tests/unit/onboardingPage.test.ts` at **3 passed**. Nothing in CI would notice the honest copy being lost. The copy is correct today; the suite simply does not hold it in place, which is the same class of gap the plan set out to close.

## Expected output
A fourth case in `tests/unit/onboardingPage.test.ts` rejects the assessment with `new ApiError(503, 'ai_unavailable')` (and ideally `502 ai_bad_output`) and asserts the AI-specific message renders — so deleting `pages/onboarding.vue:15` fails the suite. A case for `400 invalid_request` belongs with it if that code gains its own copy (see the cleared-time-field bug).

## Evidence
- Plan: `harness/plans/2026-09-23-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md` (review: `harness/reviews/2026-09-23-nuxt-public-stub-onboarding-defaults-to-true-so-a-deployment.md`).
- `frontend/pages/onboarding.vue:13-17` — `assessErrorMessage`; line 15 is the unguarded `ai_*` branch.
- `frontend/tests/unit/onboardingPage.test.ts:102-111` — the only error case, `429 rate_limited`.
- `backend/internal/onboarding/handler.go:44-58` — `ai_unavailable` (503), `ai_bad_output` (502), `ai_upstream_failed` (502).
- All three confirmed live against the real Go backend (API 8107): keyless boot + valid body → `{"error":"ai_unavailable"} HTTP 503`; provider returning non-JSON → `{"error":"ai_bad_output"} HTTP 502`; provider returning 500 → `{"error":"ai_upstream_failed"} HTTP 502`. In a browser, the 502 rendered "Máy chủ AI đang bận, chưa chấm được bài. Thử lại sau ít phút." — correct, and unguarded.

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Planned today in `harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md` (Also planned here).**

*Confirmed (read on this branch).* `frontend/tests/unit/onboardingPage.test.ts` has exactly one error case (`429 rate_limited`, lines 102-111); `pages/onboarding.vue:15` — the whole `e.code.startsWith('ai_')` branch — is unguarded, and the reviewer's mutation (deleting the line) left the suite green. This is the branch a keyless first Railway deploy puts every new user on (`main.go` logs "no provider API keys set; AI-backed routes will answer 503").

*Fix.* Two cases in the same `describe`: `new ApiError(503, 'ai_unavailable')` and `new ApiError(502, 'ai_bad_output')` both render the AI-specific copy and keep the learner on question 10; deleting line 15 must turn them red. The `400 invalid_request` case lands with the head idea's new copy. Medium: a test gap, but on the funnel's most frequent error path.
