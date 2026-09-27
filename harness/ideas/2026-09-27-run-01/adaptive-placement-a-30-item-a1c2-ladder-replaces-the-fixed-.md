---
type: feature
status: proposed
source: ideator
run: 2026-09-27-run-01
order: 2
---
# Adaptive placement: a 30-item A1–C2 ladder replaces the fixed ten questions every learner and every day-28 re-check sees

## Why
The level is the single most consequential number in the product: it is the input to all 84 tasks, and `docs/PRODUCT.md` lists "wrong level" as the first reason learners quit. Today that number comes from `onboarding.Bank` — **ten fixed items, two per level A1–C1, in a fixed order, identical for every learner** (`backend/internal/onboarding/bank.go`). Two four-option items per level cannot separate B1 from B2: one lucky guess (25 %) or one careless tap moves the learner a whole level, and C2 is never probed at all.

It gets worse with what is already planned. The day-28 checkpoint plan (`2026-09-26-day-28-checkpoint-…`, planned) re-grades the learner with "the quiz UI" — the same ten questions, 28 days later, with the level-true plan's deterministic floor for 10/10. Anyone who remembers "She ___ a teacher." is promoted. A re-check that cannot tell learning from memory undermines the CEFR-progression goal (1st-thinking §1) and the reason to start a second roadmap.

Computerised adaptive testing solves exactly this and is what every serious placement test in the category does: start in the middle, go harder on a correct answer and easier on a wrong one; the score depends on *which* items were answered correctly, not how many; a shorter test is as reliable as a longer fixed one, and no two learners need to see the same items. With a hand-written 30-item bank (5 per level, A1–C2) the ladder needs no AI and no new dependency, and the day-28 re-check draws a different form.

## Expected output
User-visible:
- The placement quiz becomes **8 questions instead of 10** and visibly listens: it starts with a B1 item; after each answer the next item is one level harder (correct) or easier (wrong), clamped A1..C2; no item repeats. A strong learner reaches C1/C2 items by question 4; a beginner is not shown eight questions they cannot read.
- The result step names the level as today. The re-check on day 28 (checkpoint plan) uses the same ladder and, because items are drawn at random within a level from those not yet seen by this user, is a different form.
- Onboarding UX is otherwise unchanged (goal, reminder time, the retro-onboarding steps). The quiz already shows one question at a time; it now fetches the next one after each answer instead of holding ten.

Technical (backend `onboarding`; frontend onboarding page):
- `Bank` grows to 30 items, 5 per level A1–C2, hand-written in `bank.go` (grammar + vocabulary, one clearly correct option each; a unit test asserts 5 per level, unique ids, `Correct ∈ options`).
- The ladder lives server-side in the existing `quiz:placement:{user_id}` hash (§4, 2 h TTL): `GET /api/v1/onboarding/quiz` starts (or resumes) a session and returns `{question{id, prompt, options}, number: 1..8, total: 8}`; new `POST /api/v1/onboarding/quiz/answer` `{question_id, selected_option}` records the answer, picks the next item (level ±1, random among unseen items at that level, falling back to the nearest level with items left), and returns the next question or `{done: true}`; a stale or unknown `question_id` → 400 `invalid_request`. Levels and correct options never leave the server — the client sees no ladder position. Per-user "seen" ids are kept in the hash for the session and, for the day-28 form, in `users.placement_seen_ids TEXT[]` (or the checkpoint plan's own store — evaluator's call).
- `POST /api/v1/onboarding/assessment` keeps its §6.1 body; `answers[]` becomes optional and, when absent, the eight answers are read from the hash (the staged-level and re-submit rules of `onboarding` are unchanged). Grading: the ladder's final position (with a two-item tie rule) is computed deterministically and passed to `TaskPlacementTest` alongside the items as today; the level-true plan's deterministic *floor* becomes a **band** — the AI grade is clamped to ladder ±1 — so the AI can shade but not contradict the evidence. Fewer than 8 answered → 400 `invalid_request` (`quiz_incomplete`).
- Specs: backend spec §6.1 gains the two quiz blocks (the shipped `GET /onboarding/quiz` is already flagged as missing from §6.1 by an inbox bug — this idea is the natural moment to write both); 1st-thinking §5.1 step 4 and §7 updated; CODEMAP `onboarding`.
- Frontend: `composables/useOnboardingApi.ts` gains the answer call; `pages/onboarding.vue` quiz step iterates on `number/total` and posts each answer; the existing error copy per code stays. Unit tests: the step-through, `done`, a 400 on a stale id, and that the start button is still gated on goal + reminder time.
- Tests: ladder table (all-correct → C2 items by Q4 and a C1/C2 result; all-wrong → A1; alternating → B1/B2), no-repeat, fallback when a level is exhausted, hash round-trip (`TEST_REDIS_URL`-gated), band clamping with a scripted provider.
- Estimate: one working day (bank authoring ~2 h, backend ~4 h, frontend ~2 h). Needs the designer role only if the quiz step's chrome changes (it should not).

## Evidence
- `backend/internal/onboarding/bank.go` — `Bank` is ten fixed items, two per level A1–C1; `PublicBank()` returns all ten in order.
- `harness/plans/2026-09-26-day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md` — re-grades with "the quiz UI"; `harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-…` — deterministic floor for a 10/10 learner.
- `harness/ideas/_inbox/get-onboarding-quiz-is-shipped-but-absent-from-backend-spec-…` (selected) — the quiz endpoint has no spec block yet.
- `docs/PRODUCT.md` "Wrong level"; 1st-thinking §1 (CEFR progression), §4 (`quiz:placement`), §5.1 step 4; backend spec §6.1.
- Adaptive placement in practice (start moderate, harder on correct, easier on wrong; score by item difficulty): https://duolingo.fandom.com/wiki/Placement_test ; Duolingo English Test scoring white paper: https://duolingo-papers.s3.amazonaws.com/reports/Duolingo_whitepaper_test_scoring_2024_v1.pdf ; CAT item selection and calibration: https://arxiv.org/pdf/2410.21033
