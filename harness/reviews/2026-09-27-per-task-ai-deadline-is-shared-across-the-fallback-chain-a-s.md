---
plan: harness/plans/2026-09-27-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/gemini-thinking-budget-is-missing-from-both-env-examples-and.md]
---
# Review — Per-task AI deadline is shared across the fallback chain; a slow preferred provider starves the fallback

**Plan:** `harness/plans/2026-09-27-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md`
**Branch/worktree:** `harness/2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s` @ `b6f33af`. Reviewed in a fresh detached worktree (`.worktrees/rv-ai-5`, removed afterwards).
**Diff:** base `0392e5d`: 12 files (`airouter` config/gemini/openai/router/timeouts plus tests, `CLAUDE.md`, `harness/CODEMAP.md`). This is exactly the plan's File structure; `git diff --name-only origin/main...HEAD` lists nothing else.

_Reviewer, 2026-09-27 (unattended daily-review, id rv-ai)._

## Plan vs idea
Delivered:
- A hanging preferred provider can no longer spend the fallback's time. Each attempt runs under `attemptBudget(time.Until(deadline), providersLeft)` as a child context, so the caller's deadline is never widened and a single-provider router (production) is unchanged by construction.
- Both folded Gemini bugs are fixed: `thinkingConfig.thinkingBudget` (default 0) is sent, and a blank STOP answer is now an error so `Route` falls back.
- OpenAI-compatible gets the same blank guard.
- `LLMProvider`/`Route` signatures and onboarding's 504 mapping are unchanged.

## Code vs plan
Tasks 1–4 were followed. `left` counts only configured providers, and `left--` runs only after an attempt. `cancelAttempt()` is called right after the call, not deferred in the loop. The failure log keeps the substring the join test checks. The blank check sits after the finishReason branch, so `SAFETY` keeps its name. The executor recorded the thinkingBudget contract check, and the plan-file bookkeeping deviation is an orchestration matter only (the plan in the harness worktree reads `status: done`).

Re-run (`backend/`):
```
go build ./... ; gofmt -l internal/airouter -> (empty) ; go vet ./internal/airouter -> clean
go test -timeout 120s ./internal/airouter -count=1 -race -v | grep -E '^(--- FAIL|ok|FAIL)' -> ok 3.266s
--- PASS: TestRouteCutsAHangingPreferredProviderSoTheFallbackStillAnswers (0.10s)
--- PASS: TestRouteSplitsTheRemainingBudgetAcrossTheProvidersLeft
--- PASS: TestRouteReportsDeadlineExceededWhenEveryAttemptTimesOut (0.10s)
--- PASS: TestAttemptBudgetSplitsWhatIsLeftOverTheProvidersLeft / TestGeminiSendsAConfiguredThinkingBudget / TestConfigFromEnvReadsTheGeminiThinkingBudget / TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON / TestOpenAICompatibleRejectsNon2xxEmptyChoicesAndBadJSON / TestGeminiNamesANonStopFinishReason
go test -timeout 300s ./... -count=1 -> all ok
Mutation A (Route passes ctx, not attemptCtx) -> FAIL: Hanging…, Splits…/three_configured_providers, Splits…/gemini_and_deepseek…10s, EveryAttemptTimesOut
Mutation B (blank-text guard disabled)        -> FAIL: TestGeminiRejects…/STOP_with_empty_text, /STOP_with_whitespace_only, /no_finishReason,_empty_text
Both reverted; git status clean.
gh run list --branch <branch> --limit 1 -> b6f33af completed success
```
Runtime scope: `airouter` has no routes. The executor's build and boot-failure check plus the wall-clock hang test are the right proof, and they reproduced.

## Quality
- **Design.** An even split is the simplest rule that gives every configured provider a floor. The plan notes the trade-off honestly: a slow-but-working preferred provider is now cut at `budget/n`, for example 60 s of 180 s with three providers. That matters only in multi-provider setups. Production has one provider.
- **Error mapping.** When the preferred provider is cut by its attempt budget and a later provider fails with a non-timeout error, the joined error still `errors.Is(context.DeadlineExceeded)`, so onboarding reports 504 `ai_timeout` rather than 502. That is defensible (a provider did time out) and matches Review Focus 4, so it is not filed.
- **Test honesty.** Good. The wall-clock tests use ≤ 200 ms deadlines, the arithmetic uses deadline probes, and both mutations are caught.
- **Config/docs (low).** `GEMINI_THINKING_BUDGET` is absent from both `.env.example` files (the executor's own follow-up). `thinkingConfig` is always sent, with no value that omits it; the effect on non-thinking models is unverified, since there is no Gemini key here and production does not use Gemini.
- **Merge.** Clean against `origin/main` and against the regenerate branch. `harness/CODEMAP.md` conflicts with the other branches, and there are no code conflicts. `NewGeminiProvider` gained a parameter; none of the other four branches adds a new call to it (their diffs contain no added `NewGeminiProvider(` line; their existing calls are only in files this branch also edits), so there is no semantic break.
- CODEMAP airouter sentence and CLAUDE.md are accurate.

## Bugs filed
- low: `harness/ideas/_inbox/gemini-thinking-budget-is-missing-from-both-env-examples-and.md`.

## Verdict
**pass-with-bugs.** Delivered, mutation-proved and CI green. It can go into the daily PR.
