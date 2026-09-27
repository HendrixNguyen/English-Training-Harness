---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# SW header-strip comment and CODEMAP say a cached token is newer when it is older

## Why
`frontend/service-worker/apiStateCache.ts` and the CODEMAP `shell` sentence both justify stripping `X-Session-Token` from the `api-state` cache by saying a cached response would carry "a token *newer* than the one in the auth store". It is the other way round. A cache entry is always from the past, so its token is *older* than, or equal to, the store's. The danger the design names (§8, "the stored token never moves backwards") is that the client adopts that stale token, because it differs from `getToken()`, and moves backwards onto a token Redis no longer holds. A maintainer who trusts the comment may drop the strip once "newer" looks impossible.

## Expected output
- The comment in `apiStateCache.ts` and the CODEMAP `shell` sentence say "older / stale token, which the client would adopt and move backwards onto".

## Evidence
- Plan: `harness/plans/2026-09-26-stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md`; design `harness/designs/stay-signed-in.md` §4 "Offline" and §8.
- `frontend/service-worker/apiStateCache.ts` header comment: "A cache-served response must never carry a token that is newer than the one currently in the auth store".
- `harness/CODEMAP.md` `shell`: "(a token newer than the store would otherwise look like a renewal that never happened)".
