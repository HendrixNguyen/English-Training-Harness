---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# POST roadmaps regenerate answers 400 to a chunked request with an empty body

## Why
`RegenerateHandler` treats the body as optional only when `c.Request.ContentLength == 0`. A request sent with `Transfer-Encoding: chunked` and no body has `ContentLength == -1`, so `ShouldBindJSON` runs, hits EOF and returns `400 invalid_request`. Some clients do this for a bodyless POST, for example fetch with an empty stream body or some proxies. Plan Review Focus 4 says "the body may be absent or {}". This case is absent-but-chunked.

## Expected output
An empty body is treated as "keep the current level" however it is framed. For example, bind only when the body is non-empty (peek, or treat `io.EOF` from `ShouldBindJSON` as an empty request), and add a handler test row that sends a chunked empty body and expects 201.

## Evidence
- `harness/plans/2026-09-26-59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md`; `backend/internal/onboarding/handler.go` `RegenerateHandler` (`if c.Request.ContentLength != 0 { ShouldBindJSON }`).
- Live, against the branch binary on the review stack: `curl -X POST …/api/v1/roadmaps/regenerate -H 'Authorization: Bearer …' -H 'Transfer-Encoding: chunked' --data-binary @/dev/null` gave `{"error":"invalid_request"} 400`, while the same request with no body and no chunking gave 201.
