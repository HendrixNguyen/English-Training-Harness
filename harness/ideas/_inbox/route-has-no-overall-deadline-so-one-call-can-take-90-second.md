---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# Route has no overall deadline so one call can take 90 seconds

## Why
Each provider owns a 30-second `http.Client` timeout (§6.2, `gemini.go:15`), and `Route` tries up to
three of them sequentially. Nothing budgets the whole operation, so a `Route` call on a context
without a deadline can take 3 × 30 = 90 seconds before returning an error. Verified proportionally
with a 100 ms client timeout and three slow fakes:

```
PROBE all-timeout Route took 300ms across 3 attempts (per-provider timeout 100ms)
```

`Route` does the right thing with a context that *is* done — it checks `ctx.Err()` before each
attempt (`router.go:78-80`) and the test at `router_test.go:111-124` pins it — so a caller who sets a
deadline is safe. The problem is that nothing says they must. The only consumer wired so far is
`cmd/api/main.go`, and a Gin handler's `c.Request.Context()` carries no deadline unless the server
sets one; this repo's `cmd/api` has no `ReadTimeout`/`WriteTimeout` (and no graceful shutdown — see
`cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md`). Onboarding, the next slice, is
the one that will call `Route` from a request handler.

The symptom is not a wrong answer, it is a request that hangs long past any client's patience while
holding a goroutine, a connection and — if the caller took a slot first — a rate-limit slot, three
times over. 30 seconds is already at the edge of what an onboarding spinner can survive; 90 is not
recoverable UX.

## Expected output
Either `Route` enforces a whole-operation budget of its own — a `RouteTimeout` constant (or a
`Router.Timeout` field, default 30–45 s) applied with `context.WithTimeout` around the attempt loop,
so the sum of fallbacks is bounded regardless of the caller's context — or, if the budget is
deliberately the caller's to set, `Route`'s doc comment and `harness/CODEMAP.md` state plainly that
the worst case is `len(FallbackOrder) × ProviderTimeout` and that callers **must** pass a context
with a deadline.

The first is the safer default for a package whose whole purpose is surviving a vendor being slow;
the second is acceptable only with the documentation actually written. A test asserts the bound: three
fakes that never answer, a `Route` on `context.Background()`, and an elapsed time under the budget.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Task 1).
- `backend/internal/airouter/gemini.go:15` — `ProviderTimeout = 30 * time.Second`, per client.
- `backend/internal/airouter/router.go:56-93` — sequential loop, no `context.WithTimeout`, no total budget; `:78-80` honours a caller-supplied deadline.
- `backend/internal/airouter/router_test.go:111-124` — covers a *cancelled* context, not a missing deadline.
- Reviewer probe (scratch test, removed): three slow fakes behind a 100 ms client → `Route` returned after ~300 ms having made 3 attempts.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low.** Real: onboarding's handler sets no deadline (`grep -n WithTimeout internal/onboarding/*.go` → none), so a slow-then-failing provider chain can hold the assessment request for 90 s per `Route` call. Failure-path only. A `Router.Timeout` default is a few lines; take it in the same `airouter` plan as the 4xx classification.
