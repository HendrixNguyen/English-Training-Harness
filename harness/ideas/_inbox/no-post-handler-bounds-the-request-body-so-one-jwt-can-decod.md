---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# No POST handler bounds the request body so one JWT can decode an unbounded answers array into memory

## Why
Every JSON endpoint in the backend calls `c.ShouldBindJSON` straight against
`c.Request.Body` with no `http.MaxBytesReader` and no server-level limit —
`auth/handler.go:20`, `quests/handler.go:58`, `pet/handler.go:78` and now
`onboarding/handler.go:37`. `gin.Default()` (`cmd/api/main.go:76`) adds only
Logger and Recovery, and `r.Run` builds an `http.Server` with no `ReadTimeout`,
`WriteTimeout` or `MaxHeaderBytes`. The codebase is careful about this in the
other direction — outbound responses are read through `io.LimitReader`
(`auth/google.go:121`, 1 MiB; `airouter/gemini.go:102`, 4 MiB) — but inbound
bodies are unbounded.

The onboarding assessment is the sharpest case, because it is the first request
DTO with an *array* field. `AssessmentRequest.Answers` is a `[]Answer`
(`types.go:14`); `encoding/json` grows that slice until the body ends, before
`validate` (`service.go:134`) ever runs. One authenticated client streaming a
few hundred megabytes of `{"question_id":"q1","selected_option":"A"}` objects
allocates the whole array first and is only then told the request is invalid.
With no read timeout, a slow sender holds the goroutine and the allocation for
as long as it likes.

This is the same class as the already-filed
`duration-seconds-is-unbounded-so-one-request-bricks-a-user-s.md`: trusting the
shape of a client-supplied value before bounding it. The difference is that this
one predates the onboarding slice and applies to all four handlers, so it wants
one fix, not four.

## Expected output
- Inbound request bodies are bounded once, centrally — a small middleware that
  wraps `c.Request.Body` in `http.MaxBytesReader` (a JSON API of this shape needs
  tens of kilobytes, not megabytes) mounted on the `/api/v1` group, so every
  current and future handler inherits it.
- A body over the limit is answered `400 invalid_request` (or `413`), not an
  allocation.
- `cmd/api` serves through an explicit `http.Server` with `ReadTimeout`,
  `ReadHeaderTimeout` and `WriteTimeout` set, instead of `r.Run`'s defaults of
  none. (This overlaps the already-filed graceful-shutdown bugs
  `main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` and
  `cmd-api-has-no-graceful-shutdown-so-its-deferred-close-calls.md`, which also
  need an explicit `http.Server` — worth doing in one change.)
- A test posts a body past the limit to one guarded route and asserts the 4xx.

## Evidence
- Plan: `harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md` (the onboarding slice is where this became load-bearing, but
  the gap is repo-wide and pre-existing).
- Unbounded binds: `backend/internal/auth/handler.go:20`,
  `backend/internal/quests/handler.go:58`,
  `backend/internal/pet/handler.go:78`,
  `backend/internal/onboarding/handler.go:37`.
- No limiter anywhere inbound: `grep -rn 'MaxBytesReader|MaxMultipartMemory'
  --include='*.go' backend/` returns nothing; the only `io.LimitReader` uses are
  outbound (`backend/internal/auth/google.go:121`,
  `backend/internal/airouter/gemini.go:102`).
- No server timeouts: `backend/cmd/api/main.go:76` (`gin.Default()`) and
  `backend/cmd/api/main.go:129` (`r.Run`).
- Unbounded array field: `backend/internal/onboarding/types.go:14`
  (`Answers []Answer`), decoded before
  `backend/internal/onboarding/service.go:134` (`validate`) runs.
- Concrete input: `POST /api/v1/onboarding/assessment` with a valid bearer token
  and an `answers` array of a few million entries.
