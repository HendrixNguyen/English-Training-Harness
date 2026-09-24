---
type: feature
status: selected
source: ideator
run: 2026-09-24-run-01
priority: medium
---
# AI-graded writing practice through the essay_grading route

## Why
Productive skills (writing, and later speaking) are what learners with an IELTS or fluency goal actually pay for. They are also the skill a fixed multiple-choice quiz cannot practise. The architecture already budgets for this. `airouter` routes `essay_grading` → OpenAI (1st-thinking §6.2, CODEMAP **airouter**), but no code path ever calls that strategy, so the router has a provider slot with nothing behind it. Meanwhile the third daily task is specified as "practice/interactive" (§6.1), and today the most it can be is a static prompt with no response from the app.

A short daily writing prompt with AI feedback turns ~10 of the 30 minutes into active production, with explicit corrective feedback. The corrective-feedback literature links that kind of feedback to measurable L2 gains. It is also the kind of feature a free flashcard app cannot copy.

## Expected output
User-visible:
- Some `practice` tasks (e.g. 2–3 per week, chosen by the roadmap prompt) are **writing tasks**: a prompt pitched at the learner's CEFR level and target goal, with a 40–150-word target.
- The learner types a response and submits it. Within a few seconds they see:
  - a band/score on a simple rubric (task achievement, grammar, vocabulary, coherence, each 0–4);
  - up to 3 concrete corrections, each shown as original → improved, with a one-line reason;
  - one improved model sentence.
- Completing the task still records its minutes through `POST /quests/progress` as before. The plant is watered by time spent, not by score.
- On `ai_unavailable` or `rate_limited`, the task can still be completed with a "feedback unavailable right now" note, so an AI outage never blocks the daily target.

Technical:
- A grading endpoint (e.g. `POST /api/v1/quests/writing/grade` with `{exercise_id, text}`) behind `auth.Require()`. It is bounded by text length and the existing `ratelimit:ai` slot, and calls `TaskEssayGrading` with a strict-JSON prompt at temperature 0.2. The response goes through a `ParseGrading` validator, using the existing retry-once → `ai_bad_output` convention.
- The grading result is stored with the exercise, so the learner can revisit it and so a future re-assessment can use it (new column or table — migration plus backend spec DDL).
- The roadmap content schema gains a `writing` practice variant. This pairs with the typed-content idea in this run; either can land first.
- Route errors never leak upstream bodies (see inbox *route-returns-upstream-provider-error-bodies-verbatim…*).
- Tests: parser cases, a rate-limit test, a fake-provider test through the real router, and frontend states for each outcome.

## Evidence
- 1st-thinking §6.1 (third task "practice/interactive"), §6.2 (`essay_grading` → OpenAI strategy); CODEMAP **airouter** ("`essay_grading` → openai", no caller).
- 1st-thinking §4 `ratelimit:ai:{user_id}` 5 req/min.
- Corrective feedback effect sizes (explicit feedback strongest on short-term posttests): https://onlinelibrary.wiley.com/doi/abs/10.1111/j.1467-9922.2010.00561.x , https://pmc.ncbi.nlm.nih.gov/articles/PMC9995700/
- Inbox noted: `route-returns-upstream-provider-error-bodies-verbatim-to-its.md`, `route-has-no-overall-deadline-so-one-call-can-take-90-second.md` (a synchronous grading call makes both user-visible).

## Evaluation
_Evaluator, 2026-09-24 — daily evaluate (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium. Not planned today.**

*Is the Why real?* Yes: `airouter` routes `essay_grading` → OpenAI with no caller, the third daily task is "practice/interactive" and is static today, and productive-skill feedback is what an IELTS learner pays for.

*Achievable in one plan?* No — two: (1) backend `POST /api/v1/quests/writing/grade` behind `auth.Require()`, bounded text length, the `ratelimit:ai` slot, `TaskEssayGrading` with a strict-JSON prompt and a `ParseGrading` validator (retry once → `ai_bad_output`), and storage of the result (migration + backend spec DDL); (2) the writing UI on `/learn/:id` with its outcome states, after a design note. *Dependencies:* the `writing` practice variant belongs to the content schema that `typed-task-content-…` (selected, medium) introduces — land that first so the roadmap prompt can emit writing tasks. The synchronous grading call also makes the selected `route-has-no-overall-deadline-so-one-call-can-take-90-second.md` user-visible; plan it before or alongside.
