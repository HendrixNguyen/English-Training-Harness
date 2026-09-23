---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
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
