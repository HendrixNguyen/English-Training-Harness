---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# NUXT_PUBLIC_STUB_ONBOARDING defaults to true, so a deployment that forgets it ships the fake placement quiz

## Why
The stub flag's *comparison* is correct — `useOnboardingApi.ts:6` uses `=== 'true'`, so `"false"`, `"0"` and `""` all disable the stub and there is no `Boolean("false")` truthy-string bug. The *default* is the problem: `nuxt.config.ts:34` sets `stubOnboarding: 'true'`, so the fail-open direction is "serve fake data". A production build with `NUXT_PUBLIC_STUB_ONBOARDING` unset — the state of any deploy that forgets one env var — runs real users through `stubs/onboarding.ts`: five hard-coded grammar questions, a CEFR level computed from a four-answer key in the client bundle, and `roadmap_id: 'stub-roadmap-0000-0000-000000000000'`. No request reaches the backend, so no roadmap row is ever created and the user lands on a dashboard whose `/quests/daily` returns `no_active_roadmap` forever, with `/onboarding` cheerfully "succeeding" each time they retry.

A stub that ships silently is worse than one that fails loudly: nothing in the UI distinguishes a stubbed assessment from a real one except a small note gated on `api.isStub` (`pages/onboarding.vue:93`).

## Expected output
`nuxt.config.ts` defaults `stubOnboarding` to `'false'`; `frontend/.env.example` keeps `NUXT_PUBLIC_STUB_ONBOARDING=true` for local development, and `playwright.config.ts` already sets it explicitly. A build that forgets the variable then calls the real `/api/v1/onboarding/*` endpoints and surfaces an honest error until the onboarding slice lands, instead of fabricating a level and a roadmap id. When the onboarding slice merges, the flag and `stubs/onboarding.ts` are deleted.

## Evidence
- Plan: `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` (Task 12).
- `frontend/nuxt.config.ts:34` — `stubOnboarding: 'true'`.
- `frontend/composables/useOnboardingApi.ts:6` — `const stub = useRuntimeConfig().public.stubOnboarding === 'true'` (strict, correct comparison).
- `frontend/stubs/onboarding.ts:37-47` — `STUB_KEY`, the client-side level ladder, and the fixed `roadmap_id`.
- No unit test covers the flag boundary: `tests/unit/onboardingStub.test.ts` asserts the stub payload shapes only, never that `'false'`/unset selects the real client. `grep -rn "stubOnboarding" frontend/tests` → no hits.
