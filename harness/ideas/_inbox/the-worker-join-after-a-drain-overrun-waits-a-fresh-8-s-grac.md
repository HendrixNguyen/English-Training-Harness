---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# The worker join after a drain overrun waits a fresh 8 s grace so worst-case shutdown is 16 s past the Railway window

## Why
`backend/cmd/api/server.go:15-18` states the invariant: Railway sends SIGTERM and kills after its own grace window, "so this stays under 10 s" (`ShutdownGrace = 8 s`). After the drain, `main.go:222` joins the pet and notify workers with `waitWithin(&workers, ShutdownGrace)` — a *fresh* 8 s, not what is left of the first 8 s. On the path this plan exists for (a drain overrun: a 60 s Google sync or a long AI call in flight at redeploy), `serve` has already spent the full 8 s, so a worker that is also stuck in a store call keeps the process alive up to 16 s after SIGTERM. Railway (or `docker stop`'s 10 s default) SIGKILLs it first, and the deferred `pg.Close()`/`rdb.Close()` — the reason the plan added the join — never run. The join's own comment says its bound exists so a stuck worker "cannot hold the process past Railway's kill window"; the bound as written does not achieve that. The source idea asked for a WaitGroup "bounded by the **remaining** grace". Impact is small today (both workers return on `ctx.Done()`, which was cancelled at SIGTERM, so the join is normally instant) — hence low.

## Expected output
One shutdown deadline for the whole stop: e.g. compute `deadline := time.Now().Add(ShutdownGrace)` when the context is done (or have `serve` return the time it used) and join the workers with `time.Until(deadline)` (floored at a small minimum), so drain + join together stay under `ShutdownGrace`. A unit test on the helper that computes the remaining budget; the existing `waitWithin` tests stay.

## Evidence
- Plan: `harness/plans/2026-09-25-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md` (Task 2 Step 4; design decision 3 "bounded by the grace").
- Source idea: `harness/ideas/_inbox/cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md` Expected output bullet 2 ("bounded by the remaining grace").
- Branch `harness/2026-09-25-medium-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r` @ `ae207d3`: `backend/cmd/api/main.go:213` (`serve(..., ShutdownGrace)`), `:222` (`waitWithin(&workers, ShutdownGrace)`), `backend/cmd/api/server.go:15-18` (the < 10 s invariant).
- Reviewer live run 2026-09-27: held request + single SIGTERM → `exit=0 after 8.06s` (the drain alone consumed the whole grace before the join started).
