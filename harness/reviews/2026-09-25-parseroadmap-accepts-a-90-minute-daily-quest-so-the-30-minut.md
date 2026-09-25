---
plan: harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md, harness/ideas/_inbox/testparseroadmaprejects-checks-only-that-an-error-occurred-s.md, harness/ideas/_inbox/week-2-provenance-assertion-in-testexercisesflattens-repeats.md]
---
# Review — ParseRoadmap enforces the 30-minute day, non-empty titles at every level, and `week` = position

**Plan:** `harness/plans/2026-09-24-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut.md`
**Branch/worktree:** `harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut` / `.worktrees/parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut`
**Diff:** `git diff main...harness/2026-09-24-medium-parseroadmap-accepts-a-90-minute-daily-quest-so-the-30-minut --stat`

_Reviewer, 2026-09-25 (unattended daily-review). The plan's worktree `.worktrees/parseroadmap-…` no longer existed, so the review ran in a fresh detached worktree at `origin/<branch>` @ `42f8fd6` in the reviewer scratchpad, which has since been removed. Diff is `origin/main...origin/<branch>`: 3 files, +91/−8 (`roadmap.go`, `roadmap_test.go`, `CODEMAP.md`)._

## Plan vs idea
Both ideas are covered in production code.
- **Head idea (90-minute day).** Tasks limited to 5..15 minutes, days to 20..40, as named constants with the §6.1 quote above them. A missing duration still defaults to 10, and the default is applied before the sum. Roadmap, module and day titles are required (whitespace-trimmed). CODEMAP now says "empty roadmap/module/day/task titles" and states the bands. The idea asked for 3×30, 3×3, empty-roadmap-title and empty-day-title rows; all exist. **But the two duration rows do not test the day budget** (see Bugs).
- **Folded idea (week ≠ position).** Module *i* must declare `week == i+1`; reversed, duplicate and zero-based weeks are rejected, as the evaluator decided (reject, do not sort). The requested flatten-test assertion is present but proves nothing new (low bug).
- `Module.Focus` stays unchecked. That is a deliberate plan choice and fine.

## Code vs plan
Tasks 1–4 were followed as written, with no deviation in production code. The extra `1f3a7b3` gofmt fixup commit is harmless. Re-run in the review worktree:

```
gofmt -l .                     -> internal/quests/handler_test.go, internal/quests/repo.go (already on origin/main, not touched here; airouter clean)
go vet ./...                   -> clean
go test ./internal/airouter ./internal/onboarding -count=1 -v | grep ...
                               -> ok airouter, ok onboarding
go test -timeout 300s ./...    -> ok for all 10 packages (cmd/api has no tests)
CI run 36026829877             -> success @ headSha 42f8fd6 (frontend, backend-unit, backend-integration, harness-tooling)
```

Executor runtime proof, re-run with `COMPOSE_PROJECT_NAME=rvb2roadmap`, PG 15493, Redis 16493, API :18593:
```
docker compose up --wait       -> postgres, redis Healthy
go test ./... -run Integration -p 1 -count=1 -> all PASS, incl. TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious
API boot                       -> migrations applied [0001 0002 0003], listening on :18593
GET /healthz                   -> {"postgres":"ok","redis":"ok","status":"ok"} 200
GET /api/v1/onboarding/quiz    -> {"error":"unauthorized"} 401
docker compose down; .env removed; no rvb2roadmap containers; worktree clean
```
Everything the executor claimed reproduced. My first boot attempt reported `unavailable` because another reviewer's API ("revcors-a") was already listening on port 18471; on a free port it was clean. There was no executor gate failure.

## Quality
- **Test honesty (blocker).** `"three thirty-minute tasks (90-minute day)"` and `"three three-minute tasks (9-minute day)"` are rejected by the **task** band (30 > 15, 3 < 5), never by the day sum. With tasks limited to 5..15, the day sum can only fail at totals of 15..19 or 41..45, and no row uses those. Mutation probe: disabling the day-sum `if` (`if false && (…)`) keeps `go test ./internal/airouter` green. A scratch probe confirmed the rule works in code (`5+5+5` → "adds up to 15 minutes, want 20..40"; `15+15+15` → "adds up to 45"). So the behaviour is right but has no test protecting it, and the rows are misnamed. The underlying cause is that the rejection table only asserts `err != nil`/`errors.Is`, never the reason (filed separately, low).
- **Correctness on untried inputs.** Negative durations are rejected by the task band. A zero duration becomes 10, which matches the plan's review-focus item 1. The order of checks (week → module title → day count → day title → tasks → day sum) gives the model a precise error message for its one retry.
- **Boundaries / conventions.** All changes stay inside `airouter.ParseRoadmap`. No other package depends on the old `maxTaskMinutes = 30` (grep: only `quests.DefaultTaskMinutes`, a separate constant still at 10). `onboarding.DailyMinutes = 30` matches the band. `ParseRoadmap` runs only on fresh model output (`onboarding/service.go:83`), never on stored `roadmap_json`, so roadmaps saved under the old 1..30 rule are not affected. Error messages follow the existing 1-based `invalid(...)` style. The doc comment is accurate.
- **Performance.** Negligible: one int accumulator per day.
- **Prompt alignment (note, not a bug).** `RoadmapSystemPrompt` already says "approximately 30 minutes … (10 mins each)" and the schema shows `10`. The plan's notes cover what to do if models overshoot: strengthen the prompt, do not widen the band.
- **Merge note.** `git merge-tree HEAD origin/main` gives one conflict in `harness/CODEMAP.md`, on the `airouter` bullet, which main also edited (provider-timeout plan). It is doc-only; keep both sides when building the daily integration branch. There is no code conflict. F2 has not landed on main, so the F2 overlap the plan warned about has not happened yet.
- **CODEMAP.** Accurate for this branch; no reviewer correction needed.

## Bugs filed
- **BLOCKER** (high, `blocks` this plan): `harness/ideas/_inbox/parseroadmap-day-sum-rejection-has-no-test-the-90-and-9-minu.md`. The day-sum rule has no test; the "90-minute" and "9-minute day" rows are caught by the task band; mutation survives.
- low: `harness/ideas/_inbox/testparseroadmaprejects-checks-only-that-an-error-occurred-s.md`. Rejection rows don't assert the reason, so they can pass for the wrong rule.
- low: `harness/ideas/_inbox/week-2-provenance-assertion-in-testexercisesflattens-repeats.md`. The week-2 assertion repeats the day-number check; fixture titles are identical across modules, so provenance is never proved.

## Verdict
**pass-with-bugs, with one blocker.** Both ideas are delivered in production code, verified and CI is green. The branch must not go into the daily integration branch until the blocker is fixed: add 5+5+5 and 15+15+15 rows that assert the "adds up to" reason, landing on this same branch as an `amends:` plan. The two low bugs can wait for normal triage.
