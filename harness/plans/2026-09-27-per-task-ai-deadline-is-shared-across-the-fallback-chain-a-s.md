---
idea: harness/ideas/_inbox/per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md
status: done
priority: medium
merged: true
branch: harness/2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s
worktree: .worktrees/per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/53"
---
# Per-task AI deadline is shared across the fallback chain; a slow preferred provider starves the fallback — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B1** of 2026-09-27. **Estimate:** 3 h. **Branch:** `harness/2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s`.

**Idea:** `harness/ideas/_inbox/per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md` — plus two folded `airouter` bugs from the same inbox: `harness/ideas/_inbox/gemini-thinking-tokens-share-maxoutputtokens-so-the-32768-bu.md` (Task 2) and `harness/ideas/_inbox/geminiprovider-returns-an-empty-string-as-success-when-a-sto.md` (Task 3). Backend only; **no design doc**.

**Depends on (must be on `origin/main` before execution):** nothing — every file below is on `origin/main` today and untouched by the 21 unmerged `done` branches. **Stay inside the file list in *File structure*.** In particular do not edit `airouter/prompt.go`, `content.go`, `level.go`, `roadmap.go` or their tests, anything under `internal/onboarding/`, `cmd/api/`, or either `.env.example` (all edited by unmerged branches; the daily integration merge cannot resolve conflicts).

**Goal:** Inside one `Route` call a provider that hangs can no longer spend the fallbacks' time — each attempt runs under its own share of what is left of the task budget, a caller's deadline is never widened and a single-provider router is unchanged; and the Gemini driver stops two silent failure shapes: thinking tokens eating `maxOutputTokens`, and a STOP answer whose text is empty being returned as success.

**Architecture:** all in `backend/internal/airouter`. `timeouts.go` gains the pure `attemptBudget(remaining, providersLeft)`; `Route` (`router.go`) computes it before each attempt from `time.Until(deadline)` (the deadline `ensureDeadline` guarantees) and the number of *configured* providers not yet tried, and runs the attempt under `context.WithTimeout(ctx, budget)` — a child of the caller's context, so nothing widens. `gemini.go` sends `generationConfig.thinkingConfig.thinkingBudget` from a new `Config.GeminiThinkingBudget` (`GEMINI_THINKING_BUDGET`, default `0`) and returns `gemini: empty response` when the joined text is blank; `openai.go` gets the same blank guard. `LLMProvider`, `Route`'s signature, `TaskTimeout` values and onboarding's `ErrAITimeout` mapping (`onboarding/service.go:154-165`: `errors.Is(err, context.DeadlineExceeded)` on a joined error) are unchanged.

## Global Constraints
- Work in `.worktrees/<slug>`; Go from `backend/`; `rg`/`timeout` not installed (`grep -n`, `go test -timeout`). Unit tests only — nothing here needs Docker; do not run `make up`.
- Only the files in *File structure* may change. Before pushing, `git diff --name-only origin/main` must list nothing else.
- Never widen a budget: the per-attempt context is always derived from the `Route` context; `attemptBudget(remaining, 1) == remaining` so production's single-provider setup (OpenRouter only) behaves exactly as today.
- No new dependencies, no signature change on `LLMProvider` or `Route`. `NewGeminiProvider` may gain one parameter — its only callers are `config.go` and this package's tests.
- `gofmt -l internal/airouter` empty; `go vet ./internal/airouter` clean; the whole package passes with `-race`.
- Tests must not sleep for real budgets: use short caller deadlines (≤ 200 ms) for wall-clock tests and deadline-probing fakes for the 30 s / 180 s arithmetic (the existing `budgeted` pattern in `timeouts_test.go`).

## Review Focus
1. `Route` with three configured providers on `TaskRoadmapGen`: the first attempt sees ≈ 60 s, after a fast failure the second sees ≈ 90 s, the third ≈ 180 s (each within 1 s) — the split is over what is *left*, and time a fast failure did not use flows to the next provider.
2. Preferred provider blocks on `<-ctx.Done()`, fallback answers immediately, caller deadline 200 ms → `Route` returns the fallback's answer, well inside 200 ms, each provider called exactly once.
3. Missing (unconfigured) providers do not count in the split; one provider gets the whole remaining budget; a caller's 10 s deadline is split, never replaced by the task default.
4. When every attempt times out the returned error still satisfies `errors.Is(err, ErrAllProvidersFailed)` **and** `errors.Is(err, context.DeadlineExceeded)`, so onboarding's 504 mapping is untouched.
5. Gemini request body carries `generationConfig.thinkingConfig.thinkingBudget` (default `0`; `GEMINI_THINKING_BUDGET=-1` sends `-1`, Google's dynamic budget) and the `GeminiMaxOutputTokens` comment says the budget includes thinking tokens.
6. A `STOP` candidate whose parts are all empty/whitespace → `gemini: empty response`; the existing `SAFETY` row (empty text, non-STOP) still reports `finishReason SAFETY`. OpenAI-compatible: empty `choices[0].message.content` → an error naming "empty".

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/airouter/timeouts.go` | `attemptBudget`; header comment says the split now enforces "room for the fallback" |
| `backend/internal/airouter/router.go` | per-attempt context in the `Route` loop; doc comment; failure log names the attempt budget |
| `backend/internal/airouter/router_test.go` | `hang` fake, `deadlineProbe` fake, the three router tests |
| `backend/internal/airouter/timeouts_test.go` | `attemptBudget` table; `NewGeminiProvider` call site |
| `backend/internal/airouter/gemini.go` | `thinkingConfig`, `GeminiThinkingBudget` default, blank-answer guard, comments |
| `backend/internal/airouter/gemini_test.go` | request assertion, blank rows, call sites |
| `backend/internal/airouter/config.go`, `config_test.go` | `GeminiThinkingBudget` from `GEMINI_THINKING_BUDGET` |
| `backend/internal/airouter/openai.go`, `openai_test.go` | blank-answer guard + one row |
| `harness/CODEMAP.md` (airouter bullet only), `CLAUDE.md` (AI router paragraph + env list) | docs |

## Tasks

### Task 1: Per-attempt budget in `Route`

**Files:** `backend/internal/airouter/timeouts.go`, `router.go`, `timeouts_test.go`, `router_test.go`.

- [ ] **Step 1 (tests first):** `timeouts_test.go` — `TestAttemptBudgetSplitsWhatIsLeftOverTheProvidersLeft`: table `(180s,3)→60s`, `(180s,2)→90s`, `(180s,1)→180s`, `(30s,3)→10s`, `(0,3)→0`, `(5s,0)→5s` (0 or 1 left = everything). `router_test.go` — add two fakes: `hang` (`<-ctx.Done()`; returns `"", ctx.Err()`; counts calls) and `deadlineProbe` (records `time.Until(deadline)` of each context it is given, then returns a fixed `out`/`err`). Tests:
  - `TestRouteCutsAHangingPreferredProviderSoTheFallbackStillAnswers` — gemini `hang`, openai `scripted{out:"o"}`, `context.WithTimeout(…, 200ms)`, `TaskPlacementTest` → `out == "o"`, `err == nil`, `gemini.calls == 1`, `len(openai.calls) == 1`, elapsed `< 200ms` (Review Focus 2).
  - `TestRouteSplitsTheRemainingBudgetAcrossTheProvidersLeft` — three `deadlineProbe`s, gemini and openai `err: errors.New("down")`, deepseek `out: "d"`, `context.Background()`, `TaskRoadmapGen` → `out == "d"`; gemini saw `60s ± 1s`, openai `90s ± 1s`, deepseek `180s ± 1s` (Review Focus 1). Second case in the same test: only gemini configured → it sees `180s ± 1s` (Review Focus 3). Third case: gemini + deepseek configured (openai missing), caller deadline 10 s → gemini sees `5s ± 1s`.
  - `TestRouteReportsDeadlineExceededWhenEveryAttemptTimesOut` — two `hang` providers, caller deadline 100 ms → `errors.Is(err, ErrAllProvidersFailed) && errors.Is(err, context.DeadlineExceeded)`, both called once (Review Focus 4). Note the existing loop returns `ctx.Err()` alone when the *outer* context is done at the end — with per-attempt caps the outer context is usually not done here, so the joined error is what comes back; if it is done, `context.DeadlineExceeded` alone is also acceptable — assert `errors.Is(err, context.DeadlineExceeded)` and, only when `err != context.DeadlineExceeded`, `ErrAllProvidersFailed`.
  Run `go test ./internal/airouter -run 'AttemptBudget|Hanging|Splits|EveryAttempt' -count=1` — all four must fail (the first on the missing symbol).
- [ ] **Step 2:** `timeouts.go` — add
  ```go
  // attemptBudget is one provider's share of what is left of the Route budget:
  // remaining split evenly over the configured providers not yet tried (this
  // one included), so a preferred provider that hangs cannot spend the
  // fallbacks' time. With one provider left it is everything, so a
  // single-provider router behaves exactly as before.
  func attemptBudget(remaining time.Duration, providersLeft int) time.Duration {
  	if providersLeft <= 1 {
  		return remaining
  	}
  	return remaining / time.Duration(providersLeft)
  }
  ```
  and rewrite `RoadmapTimeout`'s comment: the budget covers fallbacks *because* `Route` splits what is left per attempt (point at `attemptBudget`).
- [ ] **Step 3:** `router.go` — after `ensureDeadline`/`withTask`: `deadline, _ := ctx.Deadline()`; count `left` = configured providers in `order`. In the loop, after the `ctx.Err()` check: `budget := attemptBudget(time.Until(deadline), left)`; `attemptCtx, cancelAttempt := context.WithTimeout(ctx, budget)`; call `provider.GenerateContent(attemptCtx, …)`; `cancelAttempt()` immediately after the call (no `defer` inside the loop); `left--`. Keep the success return and the failure `log.Printf` — extend the latter to `"airouter: %s failed for task %s after %.1fs of a %s attempt budget: %v"` (`budget.Round(time.Second)`; the substring `"<p> failed for task <task> after"` that `TestRouteJoinsEveryErrorWhenAllProvidersFail` checks must survive). Update `Route`'s doc comment: every attempt runs under its share of the remaining budget (`attemptBudget`), so a hanging provider cannot starve the fallback, and the caller's deadline is never widened.
- [ ] **Step 4:** `cd backend && go test ./internal/airouter -count=1 -race` — new tests pass, every existing test (`TestRouteGivesTheRoadmapMoreThan30SecondsAndOtherTasksExactly30`, `TestRouteKeepsACallerDeadlineInsteadOfWideningIt`, `TestRouteStopsFallingBackOnceTheContextIsDone`, …) still passes unchanged. `gofmt -l internal/airouter` empty. Commit: `airouter: Route gives each attempt its share of the remaining budget so a hanging provider cannot starve the fallback`.

### Task 2: Gemini `thinkingConfig.thinkingBudget`

**Files:** `backend/internal/airouter/gemini.go`, `config.go`, `config_test.go`, `gemini_test.go`, `timeouts_test.go` (one call site).

- [ ] **Step 1 (tests first):** `gemini_test.go` `TestGeminiSendsTheSpec62RequestAndReturnsTheText` — assert `gen["thinkingConfig"].(map[string]any)["thinkingBudget"] == float64(0)` for a provider built with the default budget, and add `TestGeminiSendsAConfiguredThinkingBudget` (provider built with `-1` → the body carries `-1`). `config_test.go` — `TestConfigFromEnvReadsTheSpec9VariablesAndDefaults`: `want` gains `GeminiThinkingBudget: 0` (unset); add rows: `GEMINI_THINKING_BUDGET=1024` → `1024`, `=-1` → `-1`, `=abc` → `0` (unparsable falls back to the default, and `ConfigFromEnv` logs one line naming the variable — assert with `captureLog`). `TestNewRouterRegistersOnlyProvidersWithAKey` unchanged. Run — must fail to compile / fail.
- [ ] **Step 2:** `config.go` — `GeminiThinkingBudget int` with comment (`GEMINI_THINKING_BUDGET`; `0` = no thinking, the default for this strict-JSON workload; `-1` = Google's dynamic budget; a positive number = a fixed cap). `ConfigFromEnv`: `strconv.Atoi` on the variable when set, default `DefaultGeminiThinkingBudget` (`0`, a const in `gemini.go` beside `GeminiMaxOutputTokens`), log and default on a parse error. `NewRouter` passes it. `gemini.go` — `NewGeminiProvider(apiKey, baseURL, model string, thinkingBudget int, client *http.Client)` storing it; `generationConfig` gains `"thinkingConfig": map[string]any{"thinkingBudget": g.thinkingBudget}`. Update every call site (`config.go`, `gemini_test.go`, `timeouts_test.go:82`) — pass `DefaultGeminiThinkingBudget` where the test does not care. Rewrite the `GeminiMaxOutputTokens` comment: on Flash-class models `maxOutputTokens` **includes** thinking tokens, and Flash's default thinking budget is dynamic (up to ~24 k), so without `thinkingConfig` the visible answer could shrink to a fraction of 32768; `thinkingBudget: 0` gives the whole budget to the answer. Say which models accept `0` (Flash / Flash-Lite) and that Pro-class models reject it with a 400 — those operators set `GEMINI_THINKING_BUDGET` to Google's minimum for that model. Before writing that sentence, confirm the current `thinkingConfig` contract against Google's public Gemini API reference (the executor's own reading; the `context7` MCP is available) and, if the current line spells the knob differently for `gemini-3.8-flash` (e.g. a `thinkingLevel`), keep `thinkingBudget` (Google keeps it for backwards compatibility) and record what you read in the Execution summary.
- [ ] **Step 3:** `go test ./internal/airouter -count=1`; `go build ./...` (proves `cmd/api`'s `NewRouter(ConfigFromEnv(os.Getenv))` still compiles untouched). Commit: `airouter: Gemini thinkingConfig.thinkingBudget (GEMINI_THINKING_BUDGET, default 0) so thinking no longer eats maxOutputTokens`.

### Task 3: A blank answer is an error, not a success

**Files:** `backend/internal/airouter/gemini.go`, `gemini_test.go`, `openai.go`, `openai_test.go`.

- [ ] **Step 1 (tests first):** `gemini_test.go` `TestGeminiRejectsNon2xxEmptyCandidatesAndBadJSON` — rows `"STOP with empty text": {200, `{"candidates":[{"content":{"parts":[{"text":""}]},"finishReason":"STOP"}]}`, "empty"}` and `"STOP with whitespace only": {200, `…"parts":[{"text":" \n"}]…,"finishReason":"STOP"`, "empty"}`; also `"no finishReason, empty text"`. `TestGeminiNamesANonStopFinishReason`'s `SAFETY` row (empty text) must keep naming `finishReason SAFETY`. `openai_test.go` — one test/row: `{"choices":[{"message":{"content":""}}]}` → error containing `"empty"`. Run — the new rows fail.
- [ ] **Step 2:** `gemini.go` — after the join: `text := strings.TrimSpace(sb.String()); if text == "" { return "", fmt.Errorf("gemini: empty response (finishReason %q)", parsed.Candidates[0].FinishReason) }`; return `sb.String()` untrimmed as before (callers parse JSON; do not change the successful payload). Keep the check *after* the finishReason check so `SAFETY` keeps its name. `openai.go` — `if strings.TrimSpace(parsed.Choices[0].Message.Content) == "" { return "", fmt.Errorf("openai-compat(%s): empty response", o.model) }` before `logCall`… (place it after `logCall` if you want the tokens line even for a blank answer — either is fine, say which in the comment).
- [ ] **Step 3:** `go test ./internal/airouter -count=1 -race`. Commit: `airouter: a blank Gemini/OpenAI answer is an error so Route falls back instead of handing "" to the parser`.

### Task 4: Docs

**Files:** `harness/CODEMAP.md` (the `airouter` bullet only — one paragraph; do not reflow other lines), `CLAUDE.md`.

- [ ] **Step 1:** CODEMAP `airouter` bullet: after "applied by `Route` when the caller set none" add that `Route` splits what is left of the budget evenly over the configured providers not yet tried (`attemptBudget`), so a hanging provider cannot starve the fallback and a single-provider router sees no change; `GeminiProvider` sends `thinkingConfig.thinkingBudget` (`GEMINI_THINKING_BUDGET`, default 0, `-1` dynamic) because Flash counts thinking against `maxOutputTokens`; a blank answer (Gemini STOP with empty parts, OpenAI empty content) is a provider error.
- [ ] **Step 2:** `CLAUDE.md` — the "Per-task deadlines…" paragraph: one sentence on the per-attempt split; the env-variable paragraph: add `GEMINI_THINKING_BUDGET` to "The router also reads…". (`backend/.env.example` and `deploy/.env.example` are edited by unmerged branches — leave them; list the missing `#GEMINI_THINKING_BUDGET=0` line under *Follow-ups* in the Execution summary for the reviewer.)
- [ ] **Step 3:** `python3 tools/harness/cli.py validate`. Commit: `docs: airouter per-attempt budget, GEMINI_THINKING_BUDGET, blank answers`.

## Verification
```
cd backend && go build ./... && gofmt -l internal/airouter && go vet ./internal/airouter && go test -timeout 120s ./internal/airouter -count=1 -race -v 2>&1 | grep -E '^(--- |ok|FAIL)'
go test -timeout 300s ./... -count=1        # integration tests skip without TEST_*_URL; everything else must pass
grep -n 'attemptBudget' internal/airouter/timeouts.go internal/airouter/router.go
grep -n 'thinkingBudget' internal/airouter/gemini.go internal/airouter/gemini_test.go && grep -n 'GEMINI_THINKING_BUDGET' internal/airouter/config.go ../CLAUDE.md ../harness/CODEMAP.md
grep -n 'empty response' internal/airouter/gemini.go internal/airouter/openai.go
git diff --name-only origin/main   # only the paths in File structure
git push -u origin harness/2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s
```

## Notes and open questions
- **Split rule.** Even split over the providers left is the simplest rule that guarantees every configured provider a floor of `budget / n`. On the roadmap (180 s, three providers) that is 60 s for a hanging Gemini, then 60 s for OpenAI (measured 53 s) and 60 s for DeepSeek (measured 78 s — would not finish as the *third* attempt after a hang, but does as the second after a fast failure with ≈ 90 s). That is the trade-off the reviewer named (one fast failure plus one slow success) and it only bites when two providers misbehave in one call; a weighted split is a follow-up if measurements ever show it matters.
- **Terminal 4xx classification** (`route-falls-back-on-terminal-4xx-…`, still `selected`) is deliberately not in this plan: it needs a typed provider error and a per-status policy, and this plan's own `thinkingBudget` is a fresh source of vendor-specific 400s (a Pro model rejecting `0`) that must keep falling back — see that idea's Evaluation.
- Production runs one provider (OpenRouter), so Task 1 changes nothing there by construction (`attemptBudget(remaining, 1) == remaining`); Tasks 2–3 only matter once a Gemini key is set.

## Execution summary

Built and all four tasks landed as separate commits on `harness/2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s`:

1. `afad9e3` — `attemptBudget(remaining, providersLeft)` in `timeouts.go`; `Route` (`router.go`) now runs each attempt under `context.WithTimeout(ctx, budget)` derived from `time.Until(deadline)` and the count of configured-but-untried providers, decrementing after each attempt. New fakes `hang` and `deadlineProbe` in `router_test.go`; three new `Route` tests plus the `attemptBudget` table in `timeouts_test.go`.
2. `e60d784` — `GeminiProvider.thinkingBudget` (new `NewGeminiProvider` parameter), sent as `generationConfig.thinkingConfig.thinkingBudget`; `Config.GeminiThinkingBudget` / `GEMINI_THINKING_BUDGET` (`strconv.Atoi`, default `DefaultGeminiThinkingBudget = 0`, logs and defaults on a parse error). Confirmed via context7 (Google's public Gemini API docs, 2026-09-27) that the classic `generateContent` endpoint this package calls still accepts `thinkingConfig.thinkingBudget`; Google's newer Interactions API (`v1beta/interactions`, gemini-3.x) has moved to a `thinking_level` enum, but its own docs say "`thinking_budget` is retained for backward compatibility" and that the two must not be combined on one request — so `thinkingBudget` was kept rather than switched to `thinkingLevel`. Recorded in `gemini.go`'s doc comment.
3. `5b0bfc6` — a Gemini STOP (or absent-finishReason) candidate whose joined text is blank now returns `gemini: empty response (finishReason %q)` (checked after the finishReason branch, so `SAFETY` still names itself); an OpenAI-compatible empty `choices[0].message.content` returns `openai-compat(<model>): empty response` (checked after `logCall` so the token-usage line still reaches the operator on a blank answer).
4. `b6f33af` — `harness/CODEMAP.md`'s airouter bullet and `CLAUDE.md`'s AI router paragraph + env-variable list updated. `backend/.env.example` / `deploy/.env.example` were left untouched (owned by unmerged branches per the plan) — see Follow-ups.

### Deviations from the plan
- **Plan-file bookkeeping done from the worktree, not ROOT.** The orchestrator's task explicitly said not to edit or stage anything in the main checkout for this run (it was mid-session on `harness/review-auto-merge` with another session's uncommitted work, and not even on the branch that carries this plan file). So `cli.py set status=executing/…` and this summary were applied to the plan's copy inside `.worktrees/per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s/harness/plans/...md` instead of ROOT's, and were **not committed to the pushed branch** (only `harness/CODEMAP.md` is a harness-file change carried by the branch, per the plan's own File structure). The owner/orchestrator will need to replay `status=done` (see below) onto ROOT's copy once it is back on a branch that has this file, or accept this worktree's copy as the record.
- No other deviations; all Tasks/Steps followed as written.

### Verification (plan's block, from `.worktrees/.../backend`)
```
$ go build ./... && gofmt -l internal/airouter && go vet ./internal/airouter
(clean)
$ go test -timeout 120s ./internal/airouter -count=1 -race -v 2>&1 | grep -E '^(--- |ok|FAIL)'
... 41 tests, all PASS or SKIP (TestIntegrationRateLimiterAllowsFiveThenBlocks skips locally) ...
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter	2.472s
$ go test -timeout 300s ./... -count=1
ok for every package (cmd/api, airouter, auth, config, google, health, middleware, notify, onboarding, pet, quests, secrets, store)
$ grep -n 'attemptBudget' internal/airouter/timeouts.go internal/airouter/router.go        → present
$ grep -n 'thinkingBudget' internal/airouter/gemini.go internal/airouter/gemini_test.go    → present
$ grep -n 'GEMINI_THINKING_BUDGET' internal/airouter/config.go ../CLAUDE.md ../harness/CODEMAP.md → present
$ grep -n 'empty response' internal/airouter/gemini.go internal/airouter/openai.go         → present
$ git diff --name-only origin/main
CLAUDE.md, backend/internal/airouter/{config,config_test,gemini,gemini_test,openai,openai_test,router,router_test,timeouts,timeouts_test}.go, harness/CODEMAP.md
(+ harness/STATE.md, harness/plans/<this file>.md — uncommitted local bookkeeping only, not part of the 4 pushed commits)
```

### Runtime proof
This plan changes only the AI-provider fallback/timeout logic inside `internal/airouter`, which has no HTTP routes of its own ("No routes, no tables" — CODEMAP) and, per the plan's own Global Constraints, needs no Docker services ("Unit tests only — nothing here needs Docker; do not run `make up`"). Runtime evidence within that scope:
- `go build -o /tmp/... ./cmd/api` succeeds — the binary that wires `airouter.NewRouter` still builds.
- Run without any env vars: `2026/09/27 10:27:04 config: DATABASE_URL is required` then a clean exit — the documented "no env vars" failure mode is safe (no panic, no partial startup), unchanged by this fix.
- The actual bug path — a hanging preferred provider starving the fallback — is exercised end-to-end through the real, unmodified `Router.Route` call path (not mocked out) by `TestRouteCutsAHangingPreferredProviderSoTheFallbackStillAnswers`: a `hang` provider that genuinely blocks on `<-ctx.Done()` under a real 200ms `context.WithTimeout`, alongside a fallback that answers — this reproduces the exact scenario in the idea's "Evidence" section (`hang` provider as `ProviderGemini`, immediate provider as `ProviderOpenAI`) and now returns the fallback's answer well within the deadline, where before this fix it returned `err=context deadline exceeded, fallbackCalls=0`.
- Docker: confirmed `docker ps` shows nothing created by this session (only pre-existing, unrelated containers from parallel work: `migrate-lock-*`, `scio3-*`); no compose stack was started for this plan.

### CI
Branch: `harness/2026-09-27-medium-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s`
Run: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36291451596 — conclusion: **success**. All jobs green: `docker-images`, `backend-unit`, `backend-integration`, `harness-tooling`, `frontend`.

### Follow-ups for the reviewer
- `backend/.env.example` and `deploy/.env.example` were left untouched (owned by unmerged branches, per the plan) — they should eventually get a `#GEMINI_THINKING_BUDGET=0` line.
- `status=done` needs to be applied to ROOT's copy of this plan file by the orchestrator/owner once it is on a branch that carries `harness/plans/2026-09-27-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md` (see Deviations above).
