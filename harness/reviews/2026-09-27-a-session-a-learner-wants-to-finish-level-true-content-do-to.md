---
plan: harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/onboarding-declares-cefrorder-in-both-the-regenerate-and-the.md, harness/ideas/_inbox/level-guidance-asks-c2-for-300-380-word-passages-but-parsero.md, harness/ideas/_inbox/testassessraisestheaileveltothefloor-s-prompt-check-is-vacuo.md]
---
# Review — Level-true content: the roadmap prompt carries a CEFR descriptor and the goal's register, and a deterministic placement floor stops a 10/10 learner being graded A2

**Plan:** `harness/plans/2026-09-26-a-session-a-learner-wants-to-finish-level-true-content-do-to.md`
**Branch/worktree:** `harness/2026-09-26-high-a-session-a-learner-wants-to-finish-level-true-content-do-to` @ `4bb2c9f`. Reviewed in a fresh detached worktree (`.worktrees/rv-ai-4`, removed afterwards).
**Diff:** base `67ad0c0`: 9 files, +280/−14 (`airouter/level.go`+test, `prompt.go`+test, `onboarding/grade.go`+test, `service.go`+test, CODEMAP).

_Reviewer, 2026-09-27 (unattended daily-review, id rv-ai)._

## Plan vs idea
The plan owns cause (3) of the idea, and it delivers it:
- The roadmap user turn now carries a per-level CEFR descriptor and a goal-register rule ("never generic greetings… unless A1").
- A deterministic `GradeFloor` stops a 10/10 learner being graded below C1.
- `RoadmapSystemPrompt` is untouched (the pin test still passes).
- The regenerate route (from the other branch) will inherit the guidance automatically, since both call `RoadmapUserPrompt`.

## Code vs plan
Tasks 1–3 were followed:
- `LevelGuidance` has six entries plus a B1 fallback.
- The prompt adds the two sentences before the schema.
- `GradeFloor` walks A1..C1 requiring both items of each level.
- `maxLevel`/`cefrIndex` are added, and the floor is applied inside the `level == ""` branch before `StageLevel`.

The executor's deviation (two existing tests now expect C1 because the fixture answers 10/10) was anticipated by Review Focus 4 and is justified.

Re-run (`backend/`):
```
go build ./... ; gofmt -l internal/airouter internal/onboarding -> (empty) ; go vet ./... -> clean
go test -timeout 180s ./... -count=1 -race -> all ok
--- PASS: TestLevelGuidanceCoversEveryLevel / TestLevelGuidanceUnknownLevelFallsBackToB1 / TestRoadmapUserPromptCarriesLevelAndGoalGuidance
--- PASS: TestGradeFloor (6 rows) / TestMaxLevel / TestAssessRaisesTheAILevelToTheFloor / TestAssessKeepsAHigherAILevel
gh run list --branch <branch> --limit 1 -> 4bb2c9f completed success
```
Runtime proof, re-run. I used the branch binary on the rv-ai stack (PG 5441, Redis 6391), with an OpenAI-compatible stub as the only provider that grades every placement `B1`, and a real JWT and session:
```
GET /api/v1/onboarding/quiz -> 200
POST /api/v1/onboarding/assessment (Business English, 10/10) ->
  {"status":"success","assessed_level":"C1","roadmap_id":"3baa…","pet_state":{…}} 201
users.cefr_current -> C1
api log: "onboarding: placement B1 raised to C1 by the answer floor for 4a2a…"
roadmap prompt the stub received:
  - Current CEFR level: C1
  Write every task for a C1 learner. C1: use a vocabulary of roughly 6,000-8,000 word families, …
  Every task must use the language of the learner's goal ("Business English"): its situations, vocabulary and register. …
```
Reproduced; there is no executor gate failure.

## Quality
- **Correctness.** The floor only raises and never exceeds C1 (the bank has no C2 items). Duplicate answers are already rejected by `validate`, so the floor cannot be gamed by repeating an item. Answer keys never leave the server (`bank.go`), so the floor cannot be read off the client.
- **Cross-branch compile break (BLOCKER).** `grade.go:56` declares `var cefrOrder`, and so does the regenerate branch's `service.go:30`. The two merge cleanly but `go build` fails with `cefrOrder redeclared in this block`, reproduced in a scratch merge of both onto `origin/main`. The plan asked for one definition; it could not see the other branch, but the daily PR cannot carry both as they are.
- **Cross-branch prompt conflict (medium).** C2's "300–380 words" (and upper C1) versus the typed-content branch's 2000-character passage cap. Once both are on main, C2 roadmaps will often fail `ParseRoadmap`.
- **Test honesty (low).** `TestAssessRaisesTheAILevelToTheFloor`'s `strings.Contains(gen, "C1")` is vacuous, because `RoadmapSchema` always contains `A1|A2|B1|B2|C1|C2`.
- **Merge.** Textual conflicts: `prompt_test.go` with the typed-content branch, and `harness/CODEMAP.md` with every branch. None with `origin/main`.
- CODEMAP airouter/onboarding sentences are accurate.

## Bugs filed
- **BLOCKER** (high, `blocks` this plan): `harness/ideas/_inbox/onboarding-declares-cefrorder-in-both-the-regenerate-and-the.md`. The daily merge with the regenerate branch does not compile.
- medium: `harness/ideas/_inbox/level-guidance-asks-c2-for-300-380-word-passages-but-parsero.md`.
- low: `harness/ideas/_inbox/testassessraisestheaileveltothefloor-s-prompt-check-is-vacuo.md`.

## Verdict
**pass-with-bugs, with one blocker.** The idea's cause (3) is delivered, verified live and CI is green on its own. It must not go into the same daily integration branch as the regenerate branch until the duplicate `cefrOrder` is resolved on this branch (an `amends:` plan). On its own, without the regenerate branch, it builds, but the blocker still holds it per the rules.
