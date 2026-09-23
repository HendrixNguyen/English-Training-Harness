---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Every Google 409 now becomes 500 internal_error instead of 502 google_unavailable

## Why
`doJSON` maps 409 to `ErrAlreadyExists` for **every** Google service, not just the Calendar insert that
needs it, and `SyncHandler` has no case for `ErrAlreadyExists`. Any 409 the insert path does not
consume therefore falls through to `case err != nil` and is answered `500 {"error":"internal_error"}`.
Before this branch the same response was an `UpstreamError`, i.e. `502 {"error":"google_unavailable"}`.

Reachable today: `events.patch` on a stored id can return 409 on a concurrent modification —
`service.go:73-80` returns that straight out, so a transient Google conflict is reported to the client
as an internal server error. Any 409 from `tasklists.insert` / `tasklists.delete` / `tasks.insert`
does the same. 500 tells the client the fault is ours and is not a retry signal; 502 is. It also
contradicts the CODEMAP contract for this route: "other Google failures ... -> 502 `google_unavailable`".

This is adjacent to the already-selected `every-google-403-becomes-409-reauth-required-so-a-quota-erro`
finding — both are the same class: a status-code mapping in `doJSON` that is too broad for the one
call site that motivated it.

## Expected output
A 409 is interpreted as "already exists" only where that is the meaning — the Calendar `events.insert`
call — and every other 409 stays an `UpstreamError` mapped to `502 google_unavailable`. Either scope
the mapping to `service == "calendar"` plus the insert, or leave `doJSON` generic and have
`SyncHandler` map `ErrAlreadyExists` to 502 so an unconsumed 409 never surfaces as 500. A client test
pins a 409 from `events.patch` and a 409 from `tasks` to the chosen status.

## Evidence
- Plan under review: `harness/plans/2026-09-23-a-failed-savesyncstate-orphans-the-google-object-just-create.md`
- `backend/internal/google/client.go:82-83` — `case resp.StatusCode == http.StatusConflict:` in `doJSON`, applied to all three services.
- `backend/internal/google/handler.go:32-43` — the switch has `ErrReauthRequired`, `UpstreamError`/`DeadlineExceeded`, then a catch-all 500; no `ErrAlreadyExists`.
- `backend/internal/google/service.go:73-80` — the stored-id patch returns any non-`ErrNotFound` error unchanged.
- `harness/CODEMAP.md` `google` bullet — "other Google failures or the 60 s `SyncTimeout` -> 502 `google_unavailable`".
- No test asserts a 409 outside `TestCalendarInsertSendsTheClientIDAndMaps409ToAlreadyExists` (`backend/internal/google/calendar_test.go:113-145`).
- Related: `harness/ideas/_inbox/every-google-403-becomes-409-reauth-required-so-a-quota-erro.md` (selected, medium).
