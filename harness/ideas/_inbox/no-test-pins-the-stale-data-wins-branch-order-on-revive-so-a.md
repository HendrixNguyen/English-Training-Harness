---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# No test pins the stale-data-wins branch order on /revive, so a reorder silently regresses the cached-status case

## Why
The shipped code is correct: `frontend/pages/revive.vue:69` (`v-else-if="pet.status"`) sits **above** `:109` (`v-else-if="pet.error"`), so a cached/stale `pet.status` with a failed refresh still shows the last-known plant instead of an error card. That is exactly what the fixed idea's *Expected output* demanded ("Offline with a cached status, the last-known state is shown"), and it is what stops a failed `pet.revive()` — which also writes `pet.error` (`stores/pet.ts:95`) — from replacing the wilted screen with a load-error card while the user is standing in it.

Nothing in the suite enforces that order. `tests/unit/revivePage.test.ts` only ever exercises `status` and `error` as mutually exclusive (error with `status === null`, or `status` with `error === null`), so the one combination the ordering exists to resolve is untested. A later editor who groups the two "failure-ish" branches together loses a documented behaviour with a green suite.

## Expected output
A fourth case in `frontend/tests/unit/revivePage.test.ts`: load succeeds with `WILTED` (or a healthy body), then a second `pet.load()` rejects; assert the plant/alarm is still rendered and `[role="status"]` (the error card) is absent. Optionally a fifth: inside the wilted branch, a rejected `POST /pet/revive` leaves the wilted UI up and shows only the page's own inline `role="alert"` message, not the load-error card.

## Evidence
- Plan under review: `harness/plans/2026-09-23-revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md` (`status: done`).
- `frontend/pages/revive.vue:69` and `:109` — the two branches whose relative order carries the behaviour.
- `frontend/stores/pet.ts:45-55` — `load()` sets `error` and leaves a previous `status` in place; `:95` — `revive()` writes the same `error` field.
- Surviving mutation, run in `.worktrees/revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p/frontend`: moved the `v-else-if="pet.error"` block above `v-else-if="pet.status"`, then `npx vitest run tests/unit/revivePage.test.ts` → `Test Files 1 passed (1) / Tests 3 passed (3)`. Restored afterwards; `git diff --quiet pages/revive.vue` clean.
- For contrast, two mutations on the true-wilted render path were both killed (2 of 3 cases failed each time): `PlantSvg stage="wilted"` → `"sprout"`, and `v-else-if="pet.status"` → `v-else-if="pet.status && pet.error === null && false"`.
