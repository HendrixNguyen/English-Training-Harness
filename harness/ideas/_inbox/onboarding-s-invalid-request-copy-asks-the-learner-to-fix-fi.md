---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# Onboarding's invalid_request copy asks the learner to fix fields the quiz step no longer shows

## Why
The new `invalid_request` copy (`frontend/pages/onboarding.vue:16`) reads "Kiểm tra lại mục tiêu và giờ nhắc học rồi thử lại." It only ever renders at the end of the quiz, where `step === 'quiz'` — and the goal cards and the time input exist only in the `step === 'goal'` card (`onboarding.vue:91-106`). No control returns to the goal step, so the learner is told to change two fields they cannot reach; retry posts the identical body and fails identically. The loop the head idea described is unchanged for every remaining cause of `invalid_request` — only the message is different.

After this branch's gate, the time cause is unreachable, so what is left is causes the learner does not control: `backend/internal/onboarding/service.go` `validate()` also returns `ErrInvalidRequest` for a timezone `time.LoadLocation` rejects (a browser whose `Intl` zone the server's tzdata lacks; the backend imports no `time/tzdata` and the repo has no Dockerfile pinning tzdata — inference, not reproduced), an unknown `question_id` or option (a quiz bank changed between `GET /onboarding/quiz` and the POST), or a duplicate answer; plus `handler.go:38` for a body that fails to bind. For those, the copy points at the wrong fields.

Low: rare once the gate exists, but when it happens it is the same dead end after ten answers.

## Expected output
Either (a) the error card on the quiz step offers "Sửa mục tiêu / giờ nhắc" that returns to the goal step with `answers` kept, so the copy's instruction is actionable; or (b) the `invalid_request` copy stops naming goal and reminder time and says the request could not be accepted and to reload, without implying a fix the UI cannot perform. A component test covers whichever is chosen (for (a): reject with `400 invalid_request`, click the action, assert the goal step renders with the time input and that answers survive a second submit).

## Evidence
- Plan under review: `harness/plans/2026-09-24-a-cleared-reminder-time-field-dead-ends-onboarding-on-an-opa.md` (Design decision 2, Task 1).
- `frontend/pages/onboarding.vue:16` (copy), `:91-106` (goal step is the only place goal/time render), `:108-135` (quiz step: no back control).
- `backend/internal/onboarding/service.go:134-164` `validate()` — timezone, answers, question_id/option checks all map to `invalid_request`.

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — low. Not planned today.** Confirmed on `main`: the `invalid_request` copy in `frontend/pages/onboarding.vue` names the goal and reminder time, which render only on the goal step, and the quiz step has no way back. Decision: option (b) — the copy stops naming fields and says the request could not be accepted, reload and try again — is the honest minimum; a back-to-goal control is a design change for the ideator. One copy line and one test assertion; rides with the next frontend plan.
