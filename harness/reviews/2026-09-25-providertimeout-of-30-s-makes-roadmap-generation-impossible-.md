---
plan: harness/plans/2026-09-25-providertimeout-of-30-s-makes-roadmap-generation-impossible-.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md, harness/ideas/_inbox/assessment-request-has-no-overall-cap-malformed-output-retry.md]
---
# Review — Per-task AI deadlines (180 s roadmap / 30 s others), the graded level survives a failed roadmap step, Gemini default model off the retired name, one retry on 503, per-call provider logs

**Plan:** `harness/plans/2026-09-25-providertimeout-of-30-s-makes-roadmap-generation-impossible-.md`
**Branch/worktree:** `harness/2026-09-25-high-providertimeout-of-30-s-makes-roadmap-generation-impossible-` / `.worktrees/providertimeout-of-30-s-makes-roadmap-generation-impossible-`
**Diff:** `git diff main...harness/2026-09-25-high-providertimeout-of-30-s-makes-roadmap-generation-impossible- --stat`

## Plan vs idea
Both ideas' *Expected output* is delivered:
- **Per-task timeouts.** `airouter.TaskTimeout`: 180 s roadmap, 30 s otherwise. The deadline travels in the context, set per call by `onboarding.route()`, with `Route` as the backstop. The `http.Server` has no WriteTimeout (comment updated). Constants are documented in CLAUDE.md and in a §6.2 addendum in the 1st-thinking doc.
- **The grade survives a failed roadmap step.** The plan staged `_level` in the existing `quiz:placement:{user_id}` hash rather than persisting `cefr_current` or returning a 202. The §6.1 DTO is unchanged and the "nothing durable before the roadmap exists" invariant holds. This is the smaller change the idea asked for, and the plan says why.
- **Per-call provider logs.** Elapsed time, tokens, task, and on failure the upstream status + first 200 chars of the body. No key leaks: Gemini's key is in the `x-goog-api-key` header, not the URL.
- **Frontend waiting copy** is in place (there is no client fetch timeout to raise).
- **Test for the idea's timing case:** `budgeted` needs 31 s → roadmap succeeds, placement fails. It exists.
- **Gemini idea:** default `gemini-3.8-flash`, one retry on 429/502/503/504, the upstream message in the error/log.

The plan's shared-budget design (one deadline per `Route` covering the whole fallback chain) meets the idea's letter. It regresses fallback when the preferred provider is slow rather than down; see Bugs.

## Code vs plan
All 7 tasks were followed; the diff matches the plan's file table. Task 5 step 5 (the `deploy/README.md` row) was skipped per the plan's own conditional: `deploy/` is still not on the branch's base (`959cb5f`). **Heads-up for the daily PR:** `origin/main` has since merged the deploy PR (#27, `3f4242d`), so the evaluator should re-file the `GEMINI_MODEL` runbook row as the plan instructs. The one executor deviation (gofmt fix folded into Task 4) is justified.

Re-run by the reviewer. The branch was checked out, clean at `c4c8873`, in another session's nested worktree, so the review used a detached scratch worktree at the same commit; removed afterwards.
```
cd backend && make check
  gofmt/vet clean; ok for cmd/api, airouter, auth, config, google, health, middleware, notify, onboarding, pet, quests, secrets, store
grep -rn ProviderTimeout backend ; exit=1 (no matches)
grep -rn gemini-2.5-flash backend ../CLAUDE.md → only "retired" comments/fixtures (gemini.go:17, gemini_test.go:102,161,169, .env.example:34, CLAUDE.md:52)
go test ./internal/airouter/ ./internal/onboarding/ -run 'Timeout|Deadline|Retr|KeepsTheGrade|GradesAgain|Logs' -race -v
  PASS x11 (TestGeminiRetriesOnceOn503AndLogsElapsedAndTokens, …NoRetryA404…, …RetryBackoffRespectsTheDeadline, TestTaskTimeoutsMatchTheMeasuredProviders,
  TestRouteKeepsACallerDeadlineInsteadOfWideningIt, TestProvidersObeyTheContextDeadlineNotAClientTimeout, TestAssessRetriesOnceOnABadGrade…,
  TestAssessGivesEachAICallItsOwnDeadline, TestAssessReportsADeadlineHitAsAITimeoutWithoutWriting, TestAssessKeepsTheGradeWhen…, TestAssessGradesAgainWhen…)
frontend: npm ci && npm run lint && npm run test:unit → eslint clean; 16 files, 79/79 passed
git diff --stat origin/main...HEAD -- harness/ → only harness/CODEMAP.md
deploy/README.md absent on branch → Task 5 step 5 skipped (as recorded)
```
Runtime proof re-run: `COMPOSE_PROJECT_NAME=rev-providertimeout`, PG 15533, Redis 16480, API :18082, Python fake OpenAI on :18091, `GEMINI_API_KEY` unset. Torn down with `docker compose down -v`.
```
1. roadmap sleeps 35 s  → 201 35.048s, assessed_level B1, roadmap_id; log: fallback gemini→openai (placement), "openai-compat(gpt-4o-mini) task=placement_test ok 0.0s tokens prompt=1 completion=1",
                          "task=roadmap_generation ok 35.0s tokens prompt=100 completion=4500"
2. roadmap 503 once     → 201 2.02s; log "status 503: {"error": {"message": "high demand"}}; retrying once in 2s" then ok
3. roadmap 503 always   → 502 {"error":"ai_upstream_failed"} in 2.01s; log "openai failed for task roadmap_generation after 2.0s: … status 503: …";
                          HGETALL quiz:placement:<uid> → q1 B _level B1
4. same request, healthy → 201 0.009s; log "onboarding: reusing the staged level B1 for <uid>", no placement_test call; hash empty afterwards
```
Everything the executor claimed reproduced. Not re-proven live: the 504 `ai_timeout` path (unit-tested only, as the executor also noted), and whether `gemini-3.8-flash` exists (the reviewer made no live Gemini call; the name rests on Google's 404 message quoted in the idea). The owner's post-merge live onboarding check in the plan covers both.

CI: run 36120925682 on head `c4c8873` is green (backend-unit, backend-integration, frontend, harness-tooling). backend-integration runs the new `TEST_REDIS_URL`-gated quiz-store test.

## Quality
- **Design — shared deadline across the fallback chain (bug, medium).** `Route` passes one context to every provider, so a preferred provider that hangs consumes the whole task budget. Reviewer probe (throwaway test, deleted): hanging `ProviderGemini` + instant `ProviderOpenAI`, 200 ms deadline → `err=context deadline exceeded fallbackCalls=0`. On `main` a hanging Gemini on the placement test cost 30 s, then OpenAI got its own 30 s. On the branch it is a 504 with no fallback. No test covers "preferred hangs, fallback succeeds", because `budgeted` returns instantly instead of consuming time.
- **Unbounded worst case (bug, low).** `routeJSON`'s malformed-body retry gets a fresh `TaskTimeout` per attempt: up to ~7 min per assessment against copy that says 1–2 min, with no overall cap.
- **Staged-level correctness checked.** `StageAnswers` does `DEL` + `HSET` in one TxPipeline, so a changed-answer re-submit cannot inherit a stale `_level`. `StagedLevel` compares by field and counts `len-1` for `_level`, and `levelField` cannot collide with bank ids (`validate` rejects non-`q1..q10`). `StageLevel` re-applies the TTL. A failed `StageLevel` is logged and only costs a re-grade: correct degrade.
- **Concurrency.** A double-submit during the longer wait runs two AI chains. `SaveAssessment` updates the users row first inside its tx, so the row lock serializes the deactivate/insert and only one roadmap stays active. Only tokens are wasted (noted in the low bug).
- **Error mapping.** `route()` maps only *its own* deadline to `ErrAITimeout` (`ctx.Err()==nil` check), so client disconnects are not reported as 504. `Route` still returns a bare `ctx.Err()` when the deadline hits after fallbacks, which drops the per-provider errors from the error value. This predates the plan, and the new per-failure log line now covers it for operators.
- **Retry.** Bounded to one, runs under the same ctx, and the backoff select respects cancellation (tested). Retrying 502/504 on a non-idempotent paid POST can double-bill a request the upstream actually served. That is acceptable for one retry; noted, not filed.
- **Docs.** CLAUDE.md says "each driver … logs … on failure the upstream status". The failure line is logged by the router (and the retry line by `postJSON`), not the driver. The imprecision is minor and not filed. CODEMAP bullets are accurate.
- **Boundaries.** onboarding talks to airouter through `Generator`/`TaskTimeout`; the new code adds no cross-package table access and does not touch the daily loop.

## Bugs filed
- `harness/ideas/_inbox/per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md` (medium): a slow or hanging preferred provider starves the fallback; regression vs per-provider 30 s.
- `harness/ideas/_inbox/assessment-request-has-no-overall-cap-malformed-output-retry.md` (low): no overall cap on the assessment; the malformed-body retry doubles the 180 s budget.

No blockers.

## Verdict
**pass-with-bugs.** The plan and both ideas are delivered. Verification, runtime proof and CI all reproduce, and the live-incident path (Gemini 503 → fallback → 35 s+ roadmap → 201; grade survives a failed roadmap) is fixed. The two filed bugs are follow-ups, not merge blockers.
