---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# GET onboarding quiz is shipped but absent from backend spec 6.1 and 1st-thinking 7

## Why
The onboarding slice ships `GET /api/v1/onboarding/quiz`
(`cmd/api/main.go:125`, `onboarding/handler.go:16`), returning
`{questions:[{id, prompt, options{A..D}}]}`. Neither spec lists it: backend
§6.1 has `POST /auth/google` and `POST /onboarding/assessment` only, and
1st-thinking §7 enumerates the same set. The endpoint is necessary — §6.1's
assessment request references `question_id` values (`"q1"`, `"q2"`) that the
client has to obtain from somewhere — so this is a gap in the spec, not a wrong
implementation. AGENTS.md and CLAUDE.md both say the §7 endpoint list must stay
in sync with the code, so the specs are now out of date.

The plan flagged this itself and asked the reviewer to file it if the spec was
not updated (*Notes and open questions*, "`GET /onboarding/quiz` is a spec
addition"). It was not; only `harness/CODEMAP.md` records it.

## Expected output
- Backend spec §6.1 gains a `GET /api/v1/onboarding/quiz` entry with its
  request headers (`Authorization: Bearer <JWT>`) and its 200 body, matching
  what `onboarding.QuizResponse` marshals.
- 1st-thinking §7's endpoint list gains the same route.
- The specs' note records that correct options and CEFR levels are server-only
  and never appear in the response, which is what
  `TestPublicBankNeverLeaksAnswersOrLevels` enforces.

## Evidence
- Plan: `harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md`, *Notes and open questions* — "the spec owner should add the
  endpoint to backend §6.1 and 1st-thinking §7 (a low spec bug for the reviewer
  to file if not)".
- Shipped route: `backend/cmd/api/main.go:125`;
  handler `backend/internal/onboarding/handler.go:16-24`; body shape
  `backend/internal/onboarding/bank.go` (`QuizResponse`, `PublicQuestion`).
- Spec §6.1 as written:
  `project-base/Adaptive English Learning Platform - Backend Technical
  Specification.md:254` lists only `POST /api/v1/onboarding/assessment` for this
  area.
- Rule: AGENTS.md / CLAUDE.md — "REST under `/api/v1`, enumerated in §7. Keep
  that list in sync with the code."

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low.** Doc-only: the endpoint is right, the two spec documents are behind. AGENTS.md makes the §7 list a contract; an executor can add the entry to both files in minutes. Batch with the next spec-touching plan (the 0003 migration plan also edits the backend spec's DDL).
