---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Service and Require failure paths are untested; the fakes' err fields are never set

## Why
`fakeRepo` declares an `err` field (`service_test.go`) and `fakeSessions` declares one
(`session_test.go`), and both are honoured inside the fakes' methods — but **no test ever sets
either one**. Only `fakeExchanger.err` is used. The machinery for testing the non-Google failure
paths was built and then never wired up.

The consequence is that the following are entirely unexercised:

- `Service.SignIn` when `UserRepo.UpsertByGoogleID` fails (database down, constraint violation);
- `Service.SignIn` when `SessionStore.Put` fails (Redis down) — including the ordering claim in the
  comment at `service.go:48-49`, that the Redis write is deliberately last "so a failure here must
  not leave a user holding a token with no session behind it";
- `Require` when `SessionStore.Get` returns a transport error rather than `ErrNoSession`.

Those are precisely the paths that the sibling bug about 401-for-everything is about, and precisely
the ones a maintainer will change first. Right now a change to any of them is invisible to the suite.
Unused fields on a test fake are also a small lie about coverage: they read as "the failure case is
handled here" to anyone skimming.

## Expected output
Either use the fields or delete them. Using them is better — three small tests:

- `SignIn` returns an error and writes no session when the repo fails;
- `SignIn` returns an error when `Put` fails, and the assertion that no partial state is left behind;
- `Require` rejects (and, once the sibling bug is fixed, returns 503 rather than 401) when the session
  store errors for a reason other than `ErrNoSession`.

`fakeSessions.Delete` also ignores its `err` field entirely, which should be made consistent either way.

## Evidence
- Plan: `harness/plans/2026-09-22-auth-google-oauth-code-exchange-and-jwt-sessions.md`
- `backend/internal/auth/service_test.go:31` — `err error` on `fakeRepo`, honoured at `:37-39`, never set by any test.
- `backend/internal/auth/session_test.go:15` — `err error` on `fakeSessions`, honoured at `:23-25` and `:32-34`, never set by any test.
- `backend/internal/auth/session_test.go:42-45` — `Delete` ignores `f.err` where `Put`/`Get` honour it.
- `backend/internal/auth/service.go:48-49` — the ordering guarantee this would cover.
- `grep -n 'repo.err\|sess.err' backend/internal/auth/*_test.go` returns nothing.
