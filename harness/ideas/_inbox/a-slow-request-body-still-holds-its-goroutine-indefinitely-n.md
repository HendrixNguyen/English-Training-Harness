---
type: bug
status: planned
source: reviewer
run: _inbox
priority: medium
plan: harness/plans/2026-09-25-cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md
---
# A slow request body still holds its goroutine indefinitely: no server ReadTimeout, and MaxBytesReader's connection-close hook never fires through gin's writer

## Why
The CORS/edge plan bounds request bodies by **size** (`middleware.BodyLimit`, 64 KiB, verified on every POST route), but not by **time**. The folded idea `no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md` asked for both: "With no read timeout, a slow sender holds the goroutine and the allocation for as long as it likes", and its *Expected output* lists `ReadTimeout`. The evaluator split the timeout half off to the cmd/api shutdown plan (`2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md`), but that plan sets only `ReadHeaderTimeout` and `IdleTimeout`. Nothing sets `ReadTimeout`. So after this branch, a client that sends its headers promptly and then trickles a body one byte every few seconds still holds a goroutine and a connection for as long as it likes. `POST /api/v1/auth/google` is unauthenticated, so no token is needed. The allocation is now capped at 64 KiB per connection, but the connection count is not.

The plan's *Notes* also say that "`http.MaxBytesReader` also tells the server to close the connection after the response, so a slow sender cannot keep the goroutine". That is not what happens here. `net/http`'s `maxBytesReader.Read` calls the close hook only when its writer satisfies the unexported `requestTooLarger` interface (`$GOROOT/src/net/http/request.go` ~L1266). Only `*http.response` satisfies it. `BodyLimit` passes `c.Writer`, which is gin's `responseWriter` wrapper, and that wrapper does not promote the method. So the hook never fires. On the live binary, the over-limit `400` comes back with no `Connection: close`. The limit still works; the claim about the connection does not hold.

## Expected output
- `cmd/api`'s `http.Server` sets a `ReadTimeout`: the whole request, body included, must arrive within a bounded time (for example 15–30 s; a 64 KiB JSON body needs milliseconds). This does not conflict with the shutdown plan's reason for leaving `WriteTimeout` unset, because `ReadTimeout` covers only reading the request, not the long AI/Google handler responses.
- A server test (in the shutdown plan's `server_test.go` style, with a real listener) opens a connection, sends headers and a partial body, stalls, and asserts that the server closes the connection within the budget.
- Either `BodyLimit` passes the underlying `http.ResponseWriter`, so the connection-close hook fires and the `400` carries `Connection: close`, or the plan note or comment that claims the hook fires is corrected. Pick one and make the docs match the behaviour.

## Evidence
- Reviewing `harness/plans/2026-09-24-the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md` (the *Notes* bullet "Status for an oversized body"). Folded idea: `harness/ideas/_inbox/no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md` (*Expected output*, third bullet: `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`).
- `harness/plans/2026-09-23-main-go-installs-a-signal-handler-with-no-server-shutdown-so.md` L31/L45: `ReadHeaderTimeout` and `IdleTimeout` only; `grep -n 'ReadTimeout' ` on that plan finds no `ReadTimeout` setting.
- `backend/internal/middleware/bodylimit.go:24`: `http.MaxBytesReader(c.Writer, c.Request.Body, max)` passes gin's wrapper. The type assertion is at `go1.27.1 $GOROOT/src/net/http/request.go:1266-1270`.
- Live, on the branch binary (review run, API :18092): a 70 000-byte `POST /api/v1/auth/google` returns `HTTP/1.1 400 Bad Request` / `{"error":"invalid_request"}` with **no** `Connection: close` header.

## Evaluation
_Evaluator, 2026-09-25 — daily decide (AGENTS.md standing priority: rank on user impact; ≤ 5 plans today)._

**Select — medium, planned today, folded into `cmd-api-exits-1-through-log-fatalf-when-the-shutdown-grace-r.md`.** Confirmed on `main`: `newServer` sets `ReadHeaderTimeout` and `IdleTimeout` only, and `middleware.BodyLimit` hands `http.MaxBytesReader` gin's `c.Writer`, which does not satisfy net/http's unexported `requestTooLarger` (checked at `$GOROOT/src/net/http/request.go:1266` on go1.27.1), so the "close the connection" claim in the CORS plan's notes is false. `POST /api/v1/auth/google` is unauthenticated, so the goroutine-per-trickling-body is reachable by anyone. Decision: `ReadTimeout = 30 s` on the server (covers headers + body for the whole request; it does not bound handler writes, so the no-`WriteTimeout` rationale is untouched), and the doc/comment claim about the close hook is **corrected rather than worked around** — gin's writer cannot be unwrapped to `*http.response` from a middleware, and with a read deadline the connection is bounded anyway. Same file (`server.go`) as the head, hence the fold.
