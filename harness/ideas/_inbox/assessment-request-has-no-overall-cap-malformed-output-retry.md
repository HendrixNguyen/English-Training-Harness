---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Assessment request has no overall cap; malformed-output retry doubles the 180 s roadmap budget

## Why
`onboarding.routeJSON` retries a malformed AI body once. `route()` gives **each** attempt a fresh `airouter.TaskTimeout` budget, and `Route`'s 2 s-backoff 429/5xx retry runs inside each one. The worst case for one `POST /api/v1/onboarding/assessment` is 2×30 s placement + 2×180 s roadmap ≈ **7 minutes**, with no overall request cap (`http.Server` has no WriteTimeout by design). The new onboarding copy promises "thường mất 1–2 phút". A learner who gives up and re-submits starts a second concurrent AI chain (the users-row lock in `SaveAssessment` keeps only one roadmap active, but both chains burn provider tokens and the 5 req/min AI rate limit). Whether Railway's edge proxy cuts a request this long is unverified (inference, not checked). Low: this needs a malformed first roadmap *and* a slow second one.

## Expected output
- One overall deadline for `Assess` (for example `RoadmapTimeout` + `DefaultTaskTimeout` + slack), under which the per-call budgets nest, so the endpoint's worst case is documented and bounded. Or: the malformed-body retry reuses the remaining budget of the call it retries.
- `cmd/api/server.go`'s WriteTimeout comment and CLAUDE.md state that worst case.
- Unit test: a scripted provider returning a malformed roadmap and then a slow one finishes, or times out as `ErrAITimeout`, inside the overall cap.

## Evidence
- Plan: `harness/plans/2026-09-25-providertimeout-of-30-s-makes-roadmap-generation-impossible-.md` (Task 2).
- `backend/internal/onboarding/service.go:131-165` on `c4c8873`: `routeJSON` loops `attempt < 2`, and each attempt calls `route()`, which does `context.WithTimeout(ctx, airouter.TaskTimeout(task))`.
- `backend/cmd/api/server.go:26-31` (comment acknowledges "one malformed-body retry each").
- `frontend/pages/onboarding.vue` waiting copy: "thường mất 1–2 phút".

## Evaluation
_Evaluator, 2026-09-26 — **deferred** (bug cap of 5 reached; status left `proposed`)._ Needs a malformed first roadmap and a slow second one; the copy promise (1–2 min) is the user-visible part. Next free bug slot after the provider-timeout items, planned as one overall `Assess` deadline.
