---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
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
