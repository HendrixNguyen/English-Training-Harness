# Task 3 report: retro learning room

## Changes
Removed the legacy AppCard wrapper from the learning page; applied the shared dark-native screen and safe-area bottom action bar. Camp/settings links have 48px targets. Completed review hides the countdown and remains read-only. Unsaved summaries show a closed chest; known completed results open it. The result copy remains separate body text from authoritative quiz_result, including unknown historical score/time and ungraded results. No minutes, HP, streak or XP rewards are invented.

ContentViewer keeps words/definitions/examples followed by vocabulary questions, reading/glossary/look-back and the existing study/selection/feedback/summary engine. Optional speaker and companion presentation props place the learner's known companion in static 48px portraits, avoiding added motion. Question feedback appears inside the original dialogue panel with growth/ember tone, textual correctness and one intentional polite status region. Speaker panels use announce=false. Radio rows now have a single roving tab stop alongside their existing arrow/Home/End behavior. Answer keys/explanations remain absent before submission.

## Interfaces preserved
No stores, auth, API clients, content hydration, learningSession, timers or persistence helpers changed. All original ContentViewer events and session contracts remain. Added optional props: speaker:string; companion:{stage:string,health:number}. Companion data comes from existing pet.status. POST is still only the explicit summary completion action; final Continue never saves. Offline save is disabled without a queue; ambiguous outcome offers authoritative check; confirmed failure allows explicit retry. Completed tasks never POST again.

## Validation
- TDD: added two behavior tests first; learnPage suite failed exactly for missing closed chest and missing roving tabindex (22 existing tests passed).
- Focused learning verification: npm run test:unit -- tests/unit/learnPage.test.ts tests/unit/readingViewer.test.ts tests/unit/learningSession.test.ts tests/unit/exerciseHydration.test.ts tests/unit/completionCache.test.ts — 58 tests / 5 files passed.
- npm run lint — passed.
- npm run typecheck — passed after correcting redundant nested v-if narrowing.
- npm run test:unit — 407 tests / 45 files passed on final post-self-review code (04:26 UTC). Existing unrelated NuxtLink stub warnings are unchanged.
- git diff --check — passed.

## Self-review
Checked that the lifecycle script is unchanged apart from a Chest import; store/account/revision ownership guards and authoritative saved result remain intact. Feedback branches disclose correct answer only in feedback. One study/submit/Continue action bar and one summary/save/return bar are mutually exclusive. Speaker containers do not add live regions. Static portraits avoid competing motion; chest has the existing reduced-motion implementation. Body text is >=14px, display <=24px, options >=56px and primary actions >=48px; the parent reserves the shared bar's safe-area space.

## Concerns and final scope
Browser 320/375 layout, real-service regression, production build and broad route screenshots remain assigned to controller Task 7. No browser claim is made here. XP is omitted. The existing Hoàn thành label is retained for the explicit save action, with Chưa lưu kết quả directly above it. The fallback speaker label is Mầm Non when pet data is absent; a portrait is rendered only with known pet status. No frontmatter changed; no helpers/reviewers, push or PR created.
