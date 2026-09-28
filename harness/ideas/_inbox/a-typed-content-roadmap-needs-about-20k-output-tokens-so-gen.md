---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md
---
# A typed-content roadmap needs about 20k output tokens so generation cannot finish inside the 180 s budget or gpt-4o-mini's 16k cap

## Why
**Blocker for the typed-content branch.** Once this branch merges, every roadmap the model returns must carry 84 fully written tasks: word lists, 200-2000 character passages and 280-560 multiple-choice questions, each with four options and an explanation. That is several times the size of today's answer. CLAUDE.md and `timeouts.go` record today's free-form roadmap at about 4.5k output tokens, taking 53-78 s on OpenAI/DeepSeek. Production (OpenRouter, nemotron free) takes about 107 s for it, which works out to roughly 42 tokens/s.

Even the smallest roadmap the new validator accepts is 78,100 bytes of compact JSON. That was measured by marshalling `validRoadmapJSON`, which uses `SampleContent` at the minimum bounds with one-line strings. At about 4 bytes/token that is roughly 20k output tokens. This is an estimate: I counted bytes, not tokens with a real tokenizer. Real content, with passages of 130-380 words (the level-guidance branch), is larger still. At the measured throughput:
- production: 20k / 42 tok/s ≈ 475 s, against a `RoadmapTimeout` of 180 s → 504 `ai_timeout` for every onboarding and every `POST /roadmaps/regenerate`;
- `gpt-4o-mini` caps completions at 16,384 tokens and DeepSeek-chat at 8k by default, so the answer is truncated → `ParseRoadmap` fails twice → 502 `ai_bad_output`;
- Gemini's `maxOutputTokens` 32768 leaves little headroom.

The result is that no new learner can finish onboarding, which is the product's front door. The plan's own note ("if ai_bad_output rates rise … the first knob is MaxPracticeQuestions") underestimates this by an order of magnitude. The executor could not run the plan's optional live check because no provider key was available.

## Expected output
A typed roadmap reliably generates inside the task budget on the production provider. One way: keep the 28-day skeleton (titles, durations) in the `TaskRoadmapGen` call, and generate each day's typed `content` separately with `TaskExerciseGeneration`, lazily on the first `GET /quests/daily` for that day or in a small batch. Each call is then about 1k tokens and is validated by the same `validateContent`. Alternatively, cut the bounds until a live measurement fits, but only with a recorded measurement. Either way, before merge: one live `POST /onboarding/assessment` (or a scratch-DB run) against the production-class model, with its elapsed time and output token count recorded in the plan's Execution summary.

## Evidence
- Plan under review: `harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md` (branch `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` @ 3c1376e).
- Size probe (scratch test in the review worktree, deleted afterwards): `ParseRoadmap(validRoadmapJSON(t, nil))` passes and `len(...) == 78100` bytes.
- `backend/internal/airouter/content.go:48-55` (bounds), `prompt.go` `roadmapSchemaTemplate` (asks for all 84 tasks' content in one answer).
- `origin/main:backend/internal/airouter/timeouts.go:9,17` (~4.5k tokens, 53 s; `RoadmapTimeout = 180s`), `origin/main:backend/internal/airouter/gemini.go:28` (`GeminiMaxOutputTokens = 32768`); the OpenAI-compatible driver sends no `max_tokens`, so the model's default cap applies.
- Production throughput: owner's deployment notes (OpenRouter nemotron free, roadmap ≈ 107 s for the ~4.5k-token free-form answer).
