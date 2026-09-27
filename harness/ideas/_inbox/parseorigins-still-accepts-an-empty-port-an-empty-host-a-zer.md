---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# ParseOrigins still accepts an empty port, an empty host, a zero-padded or out-of-range port and a non-ASCII host

## Why
The plan's goal is that "a `FRONTEND_ORIGIN` entry that can never equal a browser's `Origin` header … fails boot". It closes three such shapes (default port, wildcard, trailing dot), but `url.Parse` accepts more that no browser ever sends, and each still boots with `cors: allowing [...]` and then 403s every PWA preflight — the same silent total outage the idea describes. The most plausible is a typo'd trailing colon (`https://app.example.com:` — browsers never serialise an empty port); the others are `https://:8443` (empty host), `https://app.example.com:0443` (browsers normalise the port), `:99999` (not a port), and a Unicode host such as `https://bücher.example` (browsers send the punycode `xn--…` form).

## Expected output
`ParseOrigins` refuses, with a reason like the existing three: an empty port (`u.Port() == ""` while `u.Host` ends in `:`), an empty `u.Hostname()`, a port that is not the canonical decimal form of 1..65535 (`strconv.Itoa(n) != port`), and a host with non-ASCII bytes ("use the punycode form the browser sends"). Rows for each in `TestParseOrigins` and `TestParseOriginsSaysWhy`.

## Evidence
- Plan: `harness/plans/2026-09-25-parseorigins-accepts-frontend-origin-entries-no-browser-send.md` (Goal; Task 1).
- Branch `harness/2026-09-25-medium-parseorigins-accepts-frontend-origin-entries-no-browser-send` @ `4884a04`: `backend/internal/middleware/cors.go` `ParseOrigins` / `neverSentByABrowser`.
- Reviewer probe 2026-09-27 (temporary `_test.go` in a scratch worktree, deleted after): `ParseOrigins("https://app.example.com:")` → `["https://app.example.com:"] err=<nil>`; `"https://:8443"` → `["https://:8443"]`; `"https://app.example.com:0443"` → accepted; `"https://app.example.com:99999"` → accepted; `"https://bücher.example"` → accepted.
