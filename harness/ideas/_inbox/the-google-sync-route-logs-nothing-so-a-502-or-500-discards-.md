---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-24-every-google-403-becomes-409-reauth-required-so-a-quota-erro.md
---
# The google sync route logs nothing, so a 502 or 500 discards Google's error reason entirely

## Why
`internal/google` contains not one log statement in production code (`grep -c 'log\.' internal/google/*.go`
excluding tests → 0). This is the first package in the backend that makes outbound network calls to a
third party — up to 30 per request — and it is the only one with no observability at all
(`quests` logs 2, `pet` 3, `airouter` 1).

The consequence is concrete: `SyncHandler` catches an error, answers
`502 {"error":"google_unavailable"}` or `500 {"error":"internal_error"}`, and **discards the error
value entirely** (`handler.go:31-44` — `err` is never logged, only switched on). `UpstreamError`
carries `Service` ("oauth"/"calendar"/"tasks"), `Status` and Google's verbatim error `Body`
(`client.go:25-33`) — the only thing that would tell an operator whether a sync is failing because
of a quota, a scope, a malformed recurrence or a dead Tasks API — and all of it is dropped on the
floor. When a user reports "sync does nothing", there is nothing to read.

Not returning Google's body to the client is correct (and matches the inbox bug
`route-returns-upstream-provider-error-bodies-verbatim-to-its.md`, which this slice did not repeat).
The problem is that it is not recorded anywhere either.

## Expected output
`SyncHandler` logs the failing error once, server-side, on the 502 and 500 branches, with the user
id and — for `*UpstreamError` — the service, status and body, while the client keeps receiving only
the opaque `google_unavailable` / `internal_error` code. The refresh token and access token must
never appear in that line; they do not today (they are never placed in an error value —
`token.go:39`, `oauth.go:47-83`, and the bearer token lives only in a header), and a test or a
comment should keep it that way.

A successful sync logging one line at info with the user id, event id and task count would make the
feature operable at all; that is a judgement call for the owner.

## Evidence
- Plan: `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` (reviewed 2026-09-23).
- `backend/internal/google/handler.go:31-44` — `res, err := svc.Sync(...)`; every non-nil `err` is mapped to a status code and then dropped.
- `backend/internal/google/client.go:25-33` — `UpstreamError{Service, Status, Body}`, the diagnostic that is discarded.
- `grep -rn 'log\.' backend/internal/google/*.go | grep -v _test` → no matches; the same grep on `internal/quests` → 2, `internal/pet` → 3, `internal/airouter` → 1.
- `harness/CODEMAP.md` → `quests`: "Hook and state errors are logged, never returned" — the convention this package does not follow.

## Evaluation
_Evaluator, 2026-09-23 — post-MVP inbox triage (AGENTS.md: rank on user impact)._

**Select — medium.** Confirmed: `handler.go:31-44` switches on `err` and drops it; `UpstreamError.Body` — the only diagnostic — never reaches a log. Operability of the one third-party integration. Two `log.Printf` lines plus a test that the tokens are absent; pair with the 403-mapping plan (same handler/client files).

## Evaluation — 2026-09-24 daily planning (evaluator)
_Owner instruction 2026-09-24: pick ≤ 5 one-day tickets from the `selected` backlog, split Bug team / Feature team, write and approve the plans, one planning PR._

**Planned today — folded into Bug team ticket B1** (`every-google-403-becomes-409-reauth-required-so-a-quota-erro.md`), Task 4 of its plan.
*Confirmed on `main`.* `handler.go:31-44` switches on `err` and never logs it; `UpstreamError{Service, Status, Body}` — the only diagnostic Google gives — is dropped. The plan adds one `log.Printf` on the 409/502/500 branches carrying user id, service, status and body (never a token: none is ever placed in an error value — `token.go`, `oauth.go`) and one info line on success, and a test that a redacted body pattern (`ya29.`) is asserted absent from the log output. The client keeps receiving only the opaque code.
