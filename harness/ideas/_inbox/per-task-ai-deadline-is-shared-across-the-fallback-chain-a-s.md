---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-27-per-task-ai-deadline-is-shared-across-the-fallback-chain-a-s.md
---
# Per-task AI deadline is shared across the fallback chain; a slow preferred provider starves the fallback

## Why
The providertimeout plan moved the AI deadline out of each driver's `http.Client{Timeout: 30s}` and into one context per `Route` call (`airouter.TaskTimeout`: 30 s placement, 180 s roadmap). `Route` hands that single context to every provider in the fallback chain, so time spent on the preferred provider is subtracted from the fallback's budget. A preferred provider that is slow or hangs (as opposed to failing fast with a 503) now uses up the whole budget, and the fallback is never tried.

On `main` today a hanging Gemini on the placement test costs 30 s, and then OpenAI gets its own 30 s (per-provider client timeout; onboarding passed a context with no deadline). On the branch the same request returns **504 `ai_timeout` after 30 s, and OpenAI is never called**. The roadmap is the same with 180 s. This is a regression of the router's fallback guarantee on the §5.1 onboarding happy path whenever Gemini is slow instead of down. It will also hit exercise generation and essay grading once they route through `Route`. `timeouts.go` names the assumption ("room for one fast-failing provider plus one slow success"), but nothing enforces it.

## Expected output
- A preferred provider that hangs cannot starve the fallback. For example, each attempt inside `Route` gets a per-attempt cap derived from the remaining budget (such as remaining / providers left, or a fixed per-attempt ceiling below `TaskTimeout`) and the fallback runs with what is left. Alternatively, restore a per-provider attempt timeout alongside the overall task deadline.
- A caller's deadline is still never widened.
- Router test: preferred provider blocks until `ctx.Done()`, fallback answers immediately → `Route` returns the fallback's answer within the task budget, and the fallback was called once.

## Evidence
- Plan: `harness/plans/2026-09-25-providertimeout-of-30-s-makes-roadmap-generation-impossible-.md` (Task 1, `timeouts.go` / `Route`).
- `backend/internal/airouter/router.go:67-99` on branch `harness/2026-09-25-high-providertimeout-of-30-s-makes-roadmap-generation-impossible-` (`c4c8873`): one `ensureDeadline` ctx passed to every `provider.GenerateContent(ctx, …)`. `backend/internal/onboarding/service.go:155-165` `route()` sets one `TaskTimeout` ctx per `Route` call.
- Reviewer probe (throwaway test, deleted): `hang` provider as `ProviderGemini` (blocks on `<-ctx.Done()`), immediate `{}` provider as `ProviderOpenAI`, `Route(ctx 200ms, TaskPlacementTest)` → `out="" err=context deadline exceeded fallbackCalls=0`.
- No existing test covers "preferred hangs, fallback succeeds". `TestRouteGivesTheRoadmapMoreThan30SecondsAndOtherTasksExactly30` uses a single provider, and `budgeted` returns instantly instead of consuming the budget.

## Evaluation
_Evaluator, 2026-09-26 — **deferred** (bug cap of 5 reached; status left `proposed`)._ Real regression of the router's fallback guarantee, reproduced by the reviewer — but production runs a single provider (OpenRouter), so the fallback never runs today. Tomorrow's bug queue, top with the Gemini thinking item; fix = per-attempt cap derived from the remaining budget plus the `preferred hangs, fallback answers` router test.

_Evaluator, 2026-09-27 — daily decide (bug queue, planned today as **B1**)._

**Select — medium; bug still present on `origin/main`.** Root cause: `backend/internal/airouter/router.go:67-91` — `ensureDeadline` builds one context per `Route` call and the loop hands that same `ctx` to every `provider.GenerateContent`, so a preferred provider that hangs spends the whole `TaskTimeout` and the fallback is never called; nothing in `timeouts.go` enforces its own "room for one fast-failing provider plus one slow success" comment (`timeouts.go:15-16`). Checked: `router_test.go` has no test where the preferred provider consumes time (`scripted` ignores its context, `budgeted` returns instantly); `onboarding/service.go:154-165` `route()` wraps each `Route` in a `TaskTimeout` context and maps a `context.DeadlineExceeded` to `ErrAITimeout` (504) — that mapping is untouched by the fix below because an all-attempts timeout still joins `DeadlineExceeded`. Production is single-provider (OpenRouter), so nobody is hit today; it matters the day a second key is set.

**Fix decision:** per-attempt cap inside `Route`: before each attempt, `attemptBudget(time.Until(deadline), providersLeft)` = the remaining budget split evenly over the configured providers not yet tried (this one included); the attempt runs under `context.WithTimeout(ctx, budget)`, a child of the caller's context, so a caller's deadline is never widened and a single-provider router is unchanged (share = everything). Tests: the idea's "preferred hangs until `ctx.Done()`, fallback answers → fallback's answer inside the task budget, each called once", a `deadlineProbe` asserting the ≈60 s / 90 s / 180 s re-split on the roadmap task, and a table for `attemptBudget`. Plan also carries the Gemini `thinkingConfig.thinkingBudget` and Gemini/OpenAI empty-answer bugs (same package, same safe files). The terminal-4xx classification stays out (see its own Evaluation).
