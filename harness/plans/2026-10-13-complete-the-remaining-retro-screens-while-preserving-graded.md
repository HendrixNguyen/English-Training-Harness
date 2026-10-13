---
idea: harness/ideas/_inbox/complete-the-remaining-retro-screens-while-preserving-graded.md
status: approved
priority: high
merged: false
design: harness/designs/retro-screens-compatibility.md
branch: harness/2026-10-13-high-retro-screens
worktree: .worktrees/retro-screens
---
# Complete the remaining retro screens — Plan

**Design:** `harness/designs/retro-screens-compatibility.md` supplements `retro-hub.md`, `retro-learning-room.md`, `retro-onboarding.md`, `retro-roadmap.md`, `retro-revive.md`, `retro-settings.md`; `harness/UI-KIT.md` v2 is binding.
**Goal:** finish all remaining screens with real data while preserving the released learning lifecycle.

## Global Constraints
- Work only in `.worktrees/retro-screens`, branch `harness/2026-10-13-high-retro-screens`; never edit main checkout app code.
- Preserve generated content hydration, glossary, answer → feedback → Continue, unsaved summary → explicit save → server result, resume, session guards and account-scoped caches. No automatic completion or offline POST queue.
- Use the existing retro kit; radius at most 2 px, no blur/gradients/scale/eased motion, body text at least 14 px; labels and controls in Vietnamese; reduced motion produces visible static states.
- Real saved data only: no inferred past completion, invented roadmap modules, fake successful sync or notifications. Missing/denied push support still permits saving time with honest status.
- Use APIs behind auth, scoped to the authenticated user. No model/provider changes, new secret requirements, schema migration unless demonstrably necessary, or unrelated backend refactor.
- Tests assert user behavior and meaningful error/ownership cases; keep existing regression coverage. Commit per task; controller dispatches reviews. Do not spawn helpers or reviewers from implementer agents.

## Tasks

### Task 1: Supply roadmap and saved reminder data
Files: `backend/internal/quests/` roadmap read DTO/repo/service/handler and tests; `backend/internal/notify/` saved settings read handler/service and tests; `backend/cmd/api/main.go`; CODEMAP and backend API spec.
- Follow compatibility addendum API contracts and existing `roadmap-tree.md` contract. GET `/api/v1/roadmap` reads active owned roadmap metadata, 4 modules/28 days, task summaries and real daily_progress; dates use learner timezone. Today beyond day28 must not stay current forever. Return no exercise answer keys/content; no generation/read-side mutation.
- GET `/api/v1/settings` reads persisted time/timezone and computed next reminder; include honest push capability signal if supported by service wiring. Do not expose subscriptions or credentials. Existing POST behavior remains compatible.
- First write failing tests for unknown/no roadmap, ownership isolation, skipped/partial/met days, dates across timezone/DST and settings reload. Implement minimal changes. Unit/race checks with service vars unset; real integration tested in final task.
- Commit, self-review, report test commands/results and exact DTOs.

### Task 2: Complete hub and shared retro shell
Files: `frontend/pages/index.vue`, `components/AppHeader.vue`, `app.vue`, `assets/css/main.css`, `composables/useGrowthMoment.ts` if needed, focused tests.
- Implement `retro-hub.md` states/signature: companion/HP, real minutes/day bar, quest path, bottom next task action, missing-roadmap CTA, wilted recovery, growth moment. Reuse current store semantics and nextTaskId; keep extra study available. Shared header/navigation should serve other screens without changing auth cleanup.
- Add behavior tests before code; retain completion return delta and first-time/no-roadmap/error cases. Lint/typecheck/focused tests. Commit and report.

### Task 3: Complete retro learning room
Files: `frontend/pages/learn/[id].vue`, `components/learn/ContentViewer.vue`, existing question/flashcard/passage/gloss components as needed, focused learning tests.
- Implement `retro-learning-room.md` through compatibility addendum. Retro panels/options/feedback and honest saved chest; passage remains reachable during questions; words/examples and vocabulary quiz retained. Completed review shows server result without re-post. Unsaved/offline/failure states remain retryable; XP cosmetic only if implemented and never authoritative.
- Preserve explicit save, all learningSession/exerciseHydration/cache/session tests. Add UI behavior coverage without weakening existing behavior. Keyboard/focus controls, no pre-submit answer leaks. Lint/typecheck/focused tests. Commit and report.

### Task 4: Complete login and onboarding
Files: `frontend/pages/login.vue`, `pages/onboarding.vue`, onboarding components and tests.
- Implement `retro-onboarding.md` preserving OAuth callback and real quiz/assessment. Use existing ItemQuestion where compatible. Profile fields/name/time and actual CEFR result; calibration only through existing POST regenerate JSON contract. Long generation/error/retry states honest; no unsupported background-generation assumptions.
- Test OAuth return handling, invalid time, assessment failure/retry, result navigation/calibration. Lint/typecheck/focused tests. Commit and report.

### Task 5: Complete real roadmap world map
Files: `frontend/pages/roadmap.vue`, new account-scoped roadmap store/types/utilities/components as needed, roadmap tests.
- Consume Task1 DTO; implement `retro-roadmap.md` actual modules/dates/completion and accessible expanded task summaries. Fetch on entry/focus/date change with stale owner/lifetime guards; never show cached previous-owner state. Past missed/partial days remain honest; day29+ no permanent today; navigate today action to current actual lesson/hub per binding design.
- Test real module/title data, missed/partial/met distinctions, today expansion, reload/regeneration/account switch, errors and no roadmap. Lint/typecheck/focused tests. Commit and report.

### Task 6: Complete revive and functioning settings
Files: `frontend/pages/revive.vue`, `pages/settings.vue`, reminder/push/Google composables and confirm sheet as needed, focused tests.
- Implement `retro-revive.md` preserving actual revive challenge and measured time; no fabricated success.
- Implement `retro-settings.md` using Task1 GET and existing POST notifications/Google sync. Load persisted time; browser permission/subscription user initiated, capability/denial honest; saving time must not reset unavailable push states. Google errors/reauth correctly labelled; no fake synced state after reload. Sign-out confirmation accessible, calls existing central auth cleanup.
- Test persisted reload, time save errors, unsupported/no-key/denied notifications, successful subscription path, Google 409/errors/success, confirmation focus/Escape, revive start/resume/complete. Lint/typecheck/focused tests. Commit and report.

### Task 7: Prove all screens and document final behavior
Files: `frontend/tests/e2e/` retro flow browser suite/config if needed, CODEMAP, compatibility/design acceptance evidence.
- Full frontend lint/typecheck/unit/build; backend build/vet/race and real service integration. Browser walk all screens at 375px and desktop, reduced motion, keyboard, loading/error/offline. Use actual services to rerun released reading/vocabulary/practice lifecycle, and route fixtures only for edge states clearly labelled.
- Complete recorded small acceptance gaps during proof: normal placement Back to profile retains answers/cursor; unchanged roadmap refresh retains a selected historical day; hub loading follows its muted companion/three-placeholder design. Remove duplicate progress copy if demonstrated. Verify actual 200% layout zoom, not only device pixel ratio; fix demonstrated overflow without hiding controls or reducing body text. Preserve readable future map nodes while evaluating fog intent.
- Capture screenshots for all primary screens and state evidence. No overlap/overflow, invisible text, blocked controls, unsupported fields or lost answers. Verify documented commands run; stop only processes/containers started here.
- Commit evidence docs, push harness branch, prove all five exact-head CI jobs green, then final independent whole-branch review and scoped fixes. No direct main/production write or deploy. Controller prepares single integration PR after pass/blocker-free.

## Verification
The per-screen acceptance lists in all six retro designs, as amended by compatibility doc, are the checklist. Full frontend suite + production build, backend unit/race and actual integration, actual learning browser flow, mobile/desktop visual and reduced-motion review, exact-head CI, harness validate and plan blockers. A skipped check is not a pass.
