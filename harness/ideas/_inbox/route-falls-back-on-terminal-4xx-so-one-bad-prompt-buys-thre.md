---
type: bug
status: selected
source: reviewer
run: _inbox
priority: low
---
# Route falls back on terminal 4xx so one bad prompt buys three paid provider calls

## Why
`Router.Route` treats every provider error as a reason to try the next provider
(`backend/internal/airouter/router.go:84-89`). That is exactly right for an outage — a 503, a
connection reset, a timeout, a vendor-side 429 — and it is what the idea's *Why* asks for. It is
wrong for a **terminal** error: a 400 "invalid request", a 422, a prompt over the model's context
window, a content-policy refusal. Those fail identically at every vendor, so the fallback loop buys
two extra paid calls and two extra round trips to learn what the first call already said.

Measured against three `httptest` fakes all answering 400:

```
PROBE 400 from preferred -> upstream calls gemini=1 openai=1 deepseek=1
```

The cost is bounded — three attempts per `Route`, and the §4 limiter caps a user at 5 `Route` calls
a minute, so the ceiling is 15 provider calls per user per minute rather than anything unbounded —
which is why this is not a merge blocker. But it is a 3× multiplier on the most expensive request
the product makes (a 28-day roadmap generation), paid precisely when the request was never going to
succeed. It also triples latency on the deterministic-failure path: the probe shows `Route` taking
the full sum of the per-provider timeouts.

The second cost is diagnostic. A single wrong `GEMINI_API_KEY` currently produces a successful
roadmap through the OpenAI fallback plus one log line, so the broken key is never noticed until the
bill arrives. The plan names this trade-off in its *Notes* ("it doubles cost in the failure case and
can mask a broken key") and leaves it open; this bug is that open question, filed.

## Expected output
`Route` classifies provider errors before falling back. The provider layer returns a typed error
carrying the upstream status (an `*ProviderError{Provider, StatusCode, err}`, or sentinels
`ErrProviderTerminal` / `ErrProviderRetryable`), and `Route` continues the loop only on the
retryable class:

- **retryable** → transport errors, `context.DeadlineExceeded` from the client timeout, 408, 429, and any 5xx;
- **terminal** → 400, 401, 403, 404, 422 and the rest of 4xx. A terminal error from the *preferred*
  provider returns immediately, wrapped so the caller sees which provider and which status.

401/403 deserve a deliberate decision rather than a default: a revoked key is terminal for that
vendor but not for the others, so the cleanest rule is "401/403 skips to the next provider **and**
logs loudly that the provider's credentials look bad", while 400/422 stops the loop.

Tests: one case per class asserting the number of upstream calls (1 for a terminal 400, 3 for a 503),
using the call-counting `httptest` fakes the package already has.

## Evidence
- Plan under review: `harness/plans/2026-09-22-ai-router-multi-llm-providers-task-strategies-and-rate-limit.md` (Task 1; the trade-off is named in *Notes and open questions*, first bullet).
- `backend/internal/airouter/router.go:84-89` — `if err == nil { return }` … `errs = append(...)`; no inspection of the error.
- `backend/internal/airouter/gemini.go:106-108` — every non-2xx becomes the same opaque `fmt.Errorf("status %d: %s", …)`, so no classification is even possible today.
- `backend/internal/airouter/router_test.go:68-80,82-99` — the two fallback tests use `errors.New("503 overloaded")` / `errors.New("gemini down")`; nothing exercises a 4xx.
- Reviewer probe (scratch test, removed): three fakes answering 400 → one request each, `Route` returns `ErrAllProvidersFailed` joined with all three.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — low (was medium).** Real but bounded: 3× cost only on deterministic failures, capped by the 5/min limiter, no user-visible symptom. Worth doing when `airouter` errors become typed (the Gemini multi-part plan introduces `finishReason` errors — a natural moment to add `*ProviderError{Status}` and stop the loop on 4xx).
