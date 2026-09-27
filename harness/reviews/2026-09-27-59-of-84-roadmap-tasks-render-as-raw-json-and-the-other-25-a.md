---
plan: harness/plans/2026-09-26-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/post-roadmaps-regenerate-answers-400-to-a-chunked-request-wi.md, harness/ideas/_inbox/onboarding-declares-cefrorder-in-both-the-regenerate-and-the.md]
---
# Review — Roadmap regeneration: `POST /api/v1/roadmaps/regenerate` replaces the active roadmap (optionally one CEFR step up or down) so a roadmap stored before the typed-content contract can be re-made

**Plan:** `harness/plans/2026-09-26-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md`
**Branch/worktree:** `harness/2026-09-26-high-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a` @ `c74ac85`. Reviewed in a fresh detached worktree (`.worktrees/rv-ai-3`, removed afterwards).
**Diff:** base `67ad0c0`: 12 files, +537/−19 (onboarding repo/service/handler/types plus tests, `cmd/api/main.go`, both specs, CODEMAP).

_Reviewer, 2026-09-27 (unattended daily-review, id rv-ai)._

## Plan vs idea
The plan owns the "regenerate stored roadmaps" part of the idea, and it delivers it. `POST /api/v1/roadmaps/regenerate` replaces the active roadmap in one transaction, at the current level or one CEFR step away, reusing onboarding's generator, parser and limiter. One dependency to note: the route only produces *typed* roadmaps once the typed-content branch is on `main`, and that branch is blocked today (see its review). So the owner's one-off regenerate would currently produce another free-form roadmap. The route itself does not need to wait.

## Code vs plan
Tasks 1–4 were followed as written:
- `insertActiveRoadmap` is shared by `SaveAssessment` and `ReplaceRoadmap`.
- `updateLevelSQL` runs first (row lock).
- `ErrNoActiveRoadmap` is checked before the limiter or any AI call.
- `stepAllowed` is case-sensitive.
- The handler `switch` order matches `AssessmentHandler`.
- The route is mounted under `guarded`.
- The spec §6.1.3 entry, the 1st-thinking §7 row and CODEMAP are updated.

Re-run in the review worktree (`backend/`, stack `COMPOSE_PROJECT_NAME=rv-ai`, PG 5441, Redis 6391):
```
go build ./... ; gofmt -l internal/onboarding cmd/api -> (empty) ; go vet ./... -> clean
go test -timeout 180s ./... -count=1 -race            -> all ok
go test ./... -run Integration -p 1 -count=1 -v       -> every Integration test PASS, incl.
  --- PASS: TestIntegrationSaveAssessmentPersists84ExercisesAndDeactivatesPrevious
  --- PASS: TestIntegrationReplaceRoadmapDeactivatesPreviousAndKeepsHistory
gh run list --branch <branch> --limit 1               -> c74ac85 completed success
```
Runtime proof, re-run with a stronger path than the executor's. I used the branch binary against the rv-ai stack, with an OpenAI-compatible stub on 127.0.0.1 as the only provider, real JWTs minted with `auth.TokenIssuer`, and sessions in Redis:
```
no token                       -> {"error":"unauthorized"} 401
never onboarded                -> {"error":"no_active_roadmap"} 404
{"cefr_level":"C1"} from B1    -> {"error":"invalid_request"} 400
malformed JSON                 -> {"error":"invalid_request"} 400
stub calls so far              -> 0   (no AI call before validation)
no body                        -> {"status":"success","assessed_level":"B1","roadmap_id":"a52b…"} 201
{"cefr_level":"B2"}            -> {"status":"success","assessed_level":"B2","roadmap_id":"9d07…"} 201
DB: cefr_current|active|roadmaps|exercises -> B2|1|3|168
stub saw prompts: "Current CEFR level: B1 / Target goal: Business English", then "…B2 / Business English"
chunked empty body             -> {"error":"invalid_request"} 400   (filed, low)
```
Everything the executor claimed reproduced, and the 201 path, which the executor could not reach without a key, works end to end.

## Quality
- **Boundaries.** Everything stays in `onboarding`, which owns `roadmaps`/`exercises`/`users.cefr_current` per CODEMAP. No cross-package table access.
- **Concurrency.** `Profile` and the step check read `cefr_current` outside the transaction, so two concurrent requests are each validated against the pre-request level. Both still land within one step of it, and the users-row lock keeps exactly one active roadmap. Acceptable. By design, successive calls can walk the level up one step each (B1 → C2 in three calls, bounded by 5/min). This is the learner's own self-placement and low-stakes, so it is not filed.
- **Error handling.** No write happens before parse. AI errors map exactly as in `Assess`.
- **Edge input (low).** A chunked request with an empty body gets 400 because of the `ContentLength != 0` gate.
- **Cross-branch compile break (blocker, filed against the level-true-content plan).** This branch's `var cefrOrder` in `service.go` collides with the same name in that branch's `grade.go`. The merge is textually clean but does not build. The blocker is set on the level-true plan, the later one that promised a single definition. This branch is not blocked by it, but whichever of the two lands second must fix it.
- **Merge.** Conflicts with `origin/main`: none. With other branches: `harness/CODEMAP.md` only, except the level-true branch, which also has the compile break above.
- CODEMAP onboarding sentence is accurate.

## Bugs filed
- low: `harness/ideas/_inbox/post-roadmaps-regenerate-answers-400-to-a-chunked-request-wi.md`. A chunked empty body gets 400 instead of 201.
- high (blocks the level-true-content plan, listed here because this branch is one side): `harness/ideas/_inbox/onboarding-declares-cefrorder-in-both-the-regenerate-and-the.md`.

## Verdict
**pass-with-bugs.** The route is delivered, verified live including 201, and CI is green. It can go into the daily PR. If the level-true-content branch goes in the same day, the duplicate `cefrOrder` must be resolved on that branch first.
