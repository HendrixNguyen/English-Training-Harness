---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# TestAssessRaisesTheAILevelToTheFloor's prompt check is vacuous because RoadmapSchema always contains C1

## Why
`TestAssessRaisesTheAILevelToTheFloor` checks that the roadmap prompt sent to the fake generator carries the raised level with `strings.Contains(gen, "C1")`. `RoadmapUserPrompt` always embeds `RoadmapSchema`, which contains `"cefr_level": "A1|A2|B1|B2|C1|C2"`, so the assertion passes even if the prompt was built for A2. The level actually sent to the model is therefore untested. Only `AssessedLevel` and the saved level are, and the prompt level is the one that decides the content a learner gets.

## Expected output
Assert on the specific line: `strings.Contains(gen, "Current CEFR level: C1")` and `strings.Contains(gen, "Write every task for a C1 learner.")`. Mutation check: passing the un-floored `aiLevel` to `RoadmapUserPrompt` must fail the test.

## Evidence
- `harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md`, Task 2 Step 1.
- `backend/internal/onboarding/service_test.go:326-328` (branch `harness/2026-09-26-high-a-session-…` @ 4bb2c9f); `backend/internal/airouter/prompt.go` `RoadmapSchema` (`"cefr_level": "A1|A2|B1|B2|C1|C2"`).
