---
plan: harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md
verdict: fail
bugs: [harness/ideas/_inbox/a-typed-content-roadmap-needs-about-20k-output-tokens-so-gen.md, harness/ideas/_inbox/typed-reading-tasks-show-questions-without-their-passage-and.md, harness/ideas/_inbox/testvalidatecontentrejects-asserts-only-the-task-location-so.md, harness/ideas/_inbox/level-guidance-asks-c2-for-300-380-word-passages-but-parsero.md]
---
# Review — Typed task content (backend half): a per-type `content` schema with answer keys, asked for in the prompt and enforced by `ParseRoadmap`

**Plan:** `harness/plans/2026-09-24-typed-task-content-with-answer-keys-so-every-quest-renders-a.md`
**Branch/worktree:** `harness/2026-09-25-medium-typed-task-content-with-answer-keys-so-every-quest-renders-a` @ `3c1376e`. Reviewed in a fresh detached worktree (`.worktrees/rv-ai-1`, removed afterwards).
**Diff:** `git diff $(git merge-base origin/main origin/<branch>)..origin/<branch> --stat` (base `959cb5f`): 8 files, +331/−12. Changed: `content.go` (new), `content_test.go` (new), `prompt.go`, `prompt_test.go`, `roadmap.go`, `roadmap_test.go`, `onboarding/fakes_test.go`, `CODEMAP.md`.

_Reviewer, 2026-09-27 (unattended daily-review, id rv-ai)._

## Plan vs idea
This plan is part 1 of 2 (backend) of the idea. The contract matches the idea's technical Expected output. Per-type `content` shapes are stated in `RoadmapSchema` and enforced by `ParseRoadmap` via `validateContent`, including `answer ∈ options` and count bounds. Bad output goes down the existing one-retry → `ai_bad_output` path. Score persistence and frontend typed branches are deliberately deferred to part 2, which is a documented evaluator decision.

Two gaps between the plan and the idea:
- **The contract cannot be generated inside the budget (blocker).** The idea's first user-visible line is "every quest task renders as a real exercise". That needs a roadmap the model can actually produce. Even the smallest roadmap the validator accepts is **78,100 bytes** of compact JSON (measured), about 20k output tokens by estimate. Today's free-form answer is about 4.5k tokens and takes 53–107 s, and the budget is 180 s. The production model on OpenRouter would need about 8 minutes. `gpt-4o-mini` caps completions at 16k tokens and DeepSeek at 8k. Once this branch merges, onboarding and regenerate would fail for every new learner. The executor could not run the plan's optional live check (no provider key), so nothing in the evidence contradicts this. It is an estimate from byte size, not a tokenizer count; that is why it is filed as a blocker requiring a live measurement, not as a proven outage.
- **"Today's frontend renders all of it without a change" is not true (medium).** `classifyContent` picks `words` before `questions` and has no `passage` branch. Typed reading tasks would show questions about a passage the learner cannot see, and vocabulary questions would be hidden. Today's raw fallback at least shows the passage text.

## Code vs plan
Tasks 1–5 were followed as written. `content.go` matches the plan's code, and `RoadmapSchema` became the `fmt.Sprintf` `var` the plan chose. `ParseRoadmap` got exactly one call. Both fixtures use `SampleContent`. The extra gofmt commit is harmless. Re-run in the review worktree (`backend/`):
```
gofmt -l internal/airouter internal/onboarding   -> (empty)
go vet ./...                                     -> clean
go test -timeout 120s ./internal/airouter -count=1 -run 'Content|Schema|ParseRoadmap' -v | grep -E '^(--- FAIL|ok|FAIL)'
                                                 -> ok  .../internal/airouter 3.027s
go test -timeout 300s ./... -count=1             -> ok for all 13 packages
gh run list --branch <branch> --limit 1          -> 3c1376e completed success
size probe (scratch test, deleted): len(validRoadmapJSON(t,nil)) = 78100, ParseRoadmap ok
```
The executor's claims reproduced, so there is no executor gate failure. The blocker is a design/feasibility defect, not a false "done".

## Quality
- **Correctness on untried inputs.** `null` content passes the `{}` guard but then fails the count checks, so it is rejected, which is correct. Option keys outside A..D are caught by the length plus required-key checks. Duplicate ids are caught after trimming. `SampleContent` ships in the production binary as an exported fixture helper, which the plan chose and is acceptable.
- **Test honesty (low).** `TestValidateContentRejects` asserts only that the error names the task, not which rule fired. This is the same pattern the reviewer flagged on `TestParseRoadmapRejects`.
- **Cross-branch.** With the level-guidance branch, C2's "300–380 words" guidance can exceed `maxPassageRunes = 2000` (medium, filed; listed here too because the bound lives in this branch's `content.go`).
- **Merge.** This branch conflicts **textually** with the ParseRoadmap-day-sum branch (`roadmap.go`, `roadmap_test.go`: both edit the `validRoadmapJSON` task line and the rejects table) and with the level-guidance branch (`prompt_test.go`). It also conflicts with every other branch in `harness/CODEMAP.md` (airouter bullet). Against `origin/main`, only CODEMAP conflicts.
- **Boundaries/CODEMAP.** All changes are inside `airouter`, and the stored bytes are unchanged (the validator decodes a copy). The CODEMAP airouter paragraph is accurate for this branch.

## Bugs filed
- **BLOCKER** (high, `blocks` this plan): `harness/ideas/_inbox/a-typed-content-roadmap-needs-about-20k-output-tokens-so-gen.md`. The typed roadmap is about 20k output tokens, beyond the 180 s budget at measured throughput and beyond the 16k/8k completion caps.
- medium: `harness/ideas/_inbox/typed-reading-tasks-show-questions-without-their-passage-and.md`. The frontend hides the passage and vocabulary questions until part 2.
- medium: `harness/ideas/_inbox/level-guidance-asks-c2-for-300-380-word-passages-but-parsero.md`. This branch's 2000-character cap conflicts with the level guidance.
- low: `harness/ideas/_inbox/testvalidatecontentrejects-asserts-only-the-task-location-so.md`. The rejection rows do not assert the reason.

## Verdict
**fail.** The code does what the plan says and is tested, but merging it as it stands would, by the size measurement above, make roadmap generation time out or truncate for every learner on the production provider. That would mean the idea ("every quest renders a real exercise") is not delivered, and onboarding would be broken. The branch must stay out of the daily PR until the blocker is resolved, either by splitting content generation per day or by a recorded live measurement that fits the budget.
