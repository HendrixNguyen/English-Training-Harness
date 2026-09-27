---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Level guidance asks C2 for 300-380-word passages but ParseRoadmap caps passages at 2000 characters

## Why
When the level-guidance branch and the typed-content branch are both on `main`, the roadmap prompt gives the model two length rules for the same passage. `RoadmapSchema` says "one passage of 200-2000 characters", and `LevelGuidance("C2")` says "Reading passages run 300-380 words" (C1 says 250-320). English prose averages about 5.5-6 characters per word including spaces, so 380 words is about 2,100-2,300 characters. The upper part of the C2 range, and C1 prose with long words, therefore exceeds `maxPassageRunes = 2000`. One such passage among a roadmap's 28 makes `ParseRoadmap` reject the whole answer. After the single retry the learner gets 502 `ai_bad_output`. A C2 learner (reachable through the AI grader) would fail onboarding and regenerate most of the time. The contradictory instructions also waste output tokens at every level.

## Expected output
The two rules agree. Either `levelGuidance` quotes lengths that fit `minPassageRunes..maxPassageRunes` (for example C2 ≤ 300 words), or the passage bounds become per level and come from one table that both `RoadmapSchema` and `LevelGuidance` read. A test asserts that the top of each level's word range × 6 is ≤ `maxPassageRunes`, and the bottom × 5 is ≥ `minPassageRunes`.

## Evidence
- `harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md`: `backend/internal/airouter/level.go` (C1 "250-320 words", C2 "300-380 words").
- `harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md`: `backend/internal/airouter/content.go` `minPassageRunes, maxPassageRunes = 200, 2000`; `prompt.go` `roadmapSchemaTemplate` ("%d-%d characters").
- Arithmetic: 380 × 5.5 = 2090, 380 × 6.0 = 2280 characters. This is an estimate; no live model output was measured (no provider key in this run).
