---
plan: harness/plans/2026-09-25-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/already-onboarded-learners-now-see-the-english-default-my-gr.md, harness/ideas/_inbox/plant-name-field-rejects-vietnamese-typed-in-decomposed-unic.md, harness/ideas/_inbox/growth-moment-plant-name-and-roadmap-tree-ui-is-built-on-kit.md]
---
# Review — Name your plant at onboarding and see it greet you by name on the hub

**Plan:** `harness/plans/2026-09-25-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md`
**Branch/worktree:** `harness/2026-09-26-medium-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o` / `.worktrees/name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o`
**Diff:** `git diff main...harness/2026-09-26-medium-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o --stat`

## Plan vs idea
Delivered for new learners: the goal step has an optional name field, `plant_name` travels in the §6.1 request only when set, the server trims it, bounds it to 30 runes, defaults a blank to "Mầm Non" and stores it once (`pet.EnsureNamed`, idempotent, 1:1), and the result step, hub caption, both wilted banners and two bubble lines say it. The idea's "fix the visible 'My Green Buddy' wart" is **not** delivered for learners who onboarded before this ships: their rows still hold the DDL default and the branch now prints it in five places (bug, medium).

## Code vs plan
Branch head `55650d5` (base `67ad0c0`); CI run 36229005157 green on that head. All eight tasks followed; the only deviations are stale expected counts in the plan's own Verification (onboarding tests 11 not 8, `index.vue` `plant_name` 4 not 3, `TestIntegration*` 14 not 13) and the `My Green Buddy` grep hitting the plan-prescribed doc comment — all explained correctly in the summary.

Re-run in a detached worktree at the branch head, isolated stack `COMPOSE_PROJECT_NAME=rv-frontend` (Postgres 5445, Redis 6395):
```
backend: env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check -> fmt/vet silent, all packages ok (-race), exit 0
backend: TEST_DATABASE_URL=…:5445 TEST_REDIS_URL=…:6395 go test ./... -count=1 -p 1 -run TestIntegration -v
  -> 14 --- PASS incl. TestIntegrationEnsureNamedWritesOnceAndKeepsTheNameOnRepeat, TestIntegrationEnsureCreatesExactlyOnePetRow; 0 SKIP, 0 FAIL
grep -rhn '^func TestIntegration' --include='*_test.go' . | wc -l -> 14
grep -n EnsureNamed internal/pet/repo.go internal/pet/service.go cmd/api/main.go | wc -l -> 9
DefaultPlantName / MaxPlantNameRunes defined (service.go:30, :33); 'My Green Buddy' in onboarding non-test code: only the doc comment (service.go:28)
frontend: npm ci && lint && typecheck && test:unit && build -> 0 / 0 / 16 files, 85 tests passed / 0
grep -c plant_name onboarding.vue index.vue revive.vue useOnboardingApi.ts -> 3 4 1 2 ; 'name?: string' in utils/plant.ts -> 1
spec plant_name 6, 1st-thinking plant\_name 3, DEFAULT 'My Green Buddy' lines 2 (DDL untouched in both)
CODEMAP matches -> 3 ; harness/ diff outside CODEMAP -> none ; cli.py validate -> 0
```
Merge with `origin/main`: `frontend/pages/revive.vue` conflicts (main changed the alarm's `text-white` → `text-ground-0` on the same element this branch edits — keep both) and `harness/CODEMAP.md`. Merge with the growth-moment branch: `utils/plant.ts` and `pages/index.vue` conflict (see that review).

**Screen walk (design `harness/designs/plant-name.md`).** Branch `.output` on :3105 against a stub API:
- Goal step: label "Đặt tên cho cây của bạn (không bắt buộc)", placeholder "Mầm Non" (not a value), `maxlength` 30 ✓. States: empty → start enabled, no note ✓; "Mầm Non!" → disabled, `aria-invalid`, note ✓; 31 chars (paste) → disabled + note ✓; "  Lá Xanh  " → enabled ✓; whitespace-only → enabled (tests) ✓. **NFD "Mầm Non" → disabled + note ✗** (bug, low).
- Hub (health 10, name "Bé Lá"): caption "Bé Lá" above the plant ✓; bubble "Bé Lá sắp héo mất! Học một chút nhé?" ✓. Wilted: banner "⚠️ Bé Lá đang bị héo rũ!" ✓; `/revive` alarm "⚠️ Bé Lá ĐANG BỊ HÉO RŨ!" with the name span `text-transform: none` ✓ (design §2: the name is not shouted).
- Result step name from the response and the `invalid_request` copy are pinned by `onboardingPage.test.ts` (not re-driven: needs the quiz flow).

## Quality
- Boundaries: `onboarding` still never imports `pet` (adapter in `cmd/api`), `pet` gains a sibling method instead of widening `Ensure`'s four callers — clean. The `IS DISTINCT FROM` guard makes a repeat a no-op write (integration test proves `RowsAffected() == 0`).
- Correctness: server-side the name is length-only by decision, so a direct API call can store control or bidi characters (rendered as text, never HTML — no injection; noted, not filed). The client class rejects combining marks (bug, low). Existing learners keep the English default (bug, medium).
- Tests: honest; the rune-vs-byte table case and the exact-body "only when set" case would catch the obvious regressions (mutations recorded).
- UI kit: built on v1 per its design (`font-display` caption, `rounded-btn` field); kit v2 drift is in the shared low bug.
- Docs: both specs updated in their own style; §3.2 DDL unchanged; CODEMAP `onboarding`/`pet`/`shell` accurate.

## Bugs filed
- `harness/ideas/_inbox/already-onboarded-learners-now-see-the-english-default-my-gr.md` (medium) — existing rows show "My Green Buddy" in five places.
- `harness/ideas/_inbox/plant-name-field-rejects-vietnamese-typed-in-decomposed-unic.md` (low) — NFD input rejected.
- `harness/ideas/_inbox/growth-moment-plant-name-and-roadmap-tree-ui-is-built-on-kit.md` (low, shared) — kit v2 drift.

## Verdict
**pass-with-bugs.** Delivered and verified end to end (backend unit + real-Postgres integration, frontend suite, live screen). No blocker; resolve the `revive.vue`/CODEMAP conflicts with main and the `speechLine`/`index.vue` conflict with the growth-moment branch when building the daily branch.
