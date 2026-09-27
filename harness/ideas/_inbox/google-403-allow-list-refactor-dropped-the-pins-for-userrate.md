---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Google 403 allow-list refactor dropped the pins for userRateLimitExceeded and quotaExceeded and logs a dangling space for a no-reason 403

## Why
`throttleReasons` listed four quota reasons explicitly (`rateLimitExceeded`, `userRateLimitExceeded`, `dailyLimitExceeded`, `quotaExceeded`); the allow-list makes them upstream by default, and the plan's Review Focus 1 names `userRateLimitExceeded` as "still → upstream". `TestCalendarMapsStatusesToSentinelErrors` only has rows for `rateLimitExceeded` and `dailyLimitExceeded` (and `TestTasksMapsAQuota403ToUpstreamNotReauth` one Tasks reason), so a future edit that adds either missing reason to `reauthReasons` — or a broad "rate" substring match — would pass the suite. Minor: for a 403 with no parseable reason the wrapped error is `…calendar returned 403 ` with a trailing space (`strings.Join(nil, ",")`), visible in `logSyncFailure` lines.

## Expected output
Rows `userquota` (`userRateLimitExceeded`) and `quota` (`quotaExceeded`) → `*UpstreamError{calendar, 403}`, not reauth; the reauth message omits the reasons segment when none parsed (e.g. `…returned 403` or `…returned 403 (no reason)`).

## Evidence
- Plan `harness/plans/2026-09-27-google-403-accessnotconfigured-api-disabled-still-maps-to-re.md` (Review Focus 1).
- `backend/internal/google/calendar_test.go` `TestCalendarMapsStatusesToSentinelErrors` upstream table; `backend/internal/google/client.go` `doJSON` 403 case `strings.Join(googleErrorReasons(raw), ",")`.
- Red check against main's `client.go` printed `svcdisabled: err = google: re-authentication required: calendar returned 403 , want …` (trailing space before the comma).
