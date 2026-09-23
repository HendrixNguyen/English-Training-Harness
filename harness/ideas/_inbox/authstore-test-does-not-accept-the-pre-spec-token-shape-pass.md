---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# authStore test "does not accept the pre-spec {token} shape" passes even when the store reads res.token

## Why
This slice's central auth claim is that the frontend reads `access_token` and never the merged handler's `token`. One test is named for exactly that claim and does not test it: it passes for an unrelated reason, so it would not catch the regression it exists to prevent.

`stores/auth.ts:59` guards on two fields — `typeof res.access_token !== 'string' || typeof res.expires_in !== 'number'`. The test's fixture is `{ token: 'legacy', user }`, which omits `expires_in`, so the guard rejects it on the *expiry* half regardless of which token key the store reads. The assertion is satisfied by the wrong branch.

## Expected output
The test fails when the store reads the wrong key. Give the fixture a valid `expires_in` — `{ token: 'legacy', token_type: 'Bearer', expires_in: 86400, user }` — so the only reason `signIn` can reject it is the missing `access_token`. Worth adding, separately, a case that the guard rejects a response with `access_token` but no `expires_in`, which is the assertion the current fixture is accidentally making.

## Evidence
- Plan: `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` (Task 3).
- `frontend/tests/unit/authStore.test.ts:31-37` — the test; its fixture at line 34 is `{ token: 'legacy', user: spec61.user }`.
- `frontend/stores/auth.ts:59` — the two-part guard.
- Mutation test performed during review: `stores/auth.ts` changed so `signIn` reads `(res as unknown as {token: string}).token` in both the guard and the assignment. `npx vitest run tests/unit/authStore.test.ts` → **1 failed | 4 passed**. The failure was line 21 (`expect(auth.accessToken).toBe('eyJ.test')`) in "signIn stores the §6.1 access_token…"; the `{token}` test at line 31 **still passed** under the mutation. Change reverted; `git diff --quiet stores/auth.ts` clean.
- The neighbouring test at line 18 does carry the claim honestly, so the behaviour is covered — only this test's name overstates what it checks.
