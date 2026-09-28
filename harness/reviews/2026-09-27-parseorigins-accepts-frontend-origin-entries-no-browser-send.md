---
plan: harness/plans/2026-09-25-parseorigins-accepts-frontend-origin-entries-no-browser-send.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/parseorigins-still-accepts-an-empty-port-an-empty-host-a-zer.md, harness/ideas/_inbox/a-malformed-frontend-origin-is-refused-only-after-migrations.md]
---
# Review — middleware: `ParseOrigins` refuses origins no browser sends, and lookalikes are pinned to 403

**Plan:** `harness/plans/2026-09-25-parseorigins-accepts-frontend-origin-entries-no-browser-send.md`
**Branch/worktree:** `harness/2026-09-25-medium-parseorigins-accepts-frontend-origin-entries-no-browser-send` / `.worktrees/parseorigins-accepts-frontend-origin-entries-no-browser-send`
**Diff:** `git diff main...harness/2026-09-25-medium-parseorigins-accepts-frontend-origin-entries-no-browser-send --stat`

## Plan vs idea
Delivered for the shapes the idea names: an explicit default port, a wildcard host and a trailing-dot host now fail boot with the entry and the reason quoted, and the exact-match rule is pinned by a seven-row lookalike table. The plan's broader goal ("an entry that can never equal a browser's Origin header") is not fully closed: `url.Parse` still lets through an empty port, an empty host, a zero-padded or out-of-range port and a non-ASCII host, each of which boots and then 403s every preflight (bug 1, low).

## Code vs plan
Reviewed at origin head `4884a04` in a detached scratch worktree (4 files, +85/-6). Merges cleanly into current `main`.
- Task 1: followed verbatim (`neverSentByABrowser`, simplified scheme condition, `ErrBadOrigin` text, six new `TestParseOrigins` rows, `TestParseOriginsSaysWhy` with `errors.Is`).
- Task 2: followed (`TestLookalikeOriginsGetNoCORSAndA403Preflight`, seven subtests, preflight 403 + no Allow-Origin on the actual request).
- Task 3: `.env.example` and CODEMAP updated as specified; CODEMAP text accurate.
- Extra commit: a `gofmt` fix to `cors_test.go`. Fine.

Re-run evidence:
```
$ gh run list --branch <branch> --limit 1
completed success middleware: gofmt cors_test.go  CI ... 36094226541
$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL make check
go vet ./... ; go test ./... -count=1 -race   -> 13 "ok" lines, nothing else
$ go test ./internal/middleware/ -count=1 -race -v | grep -c '^--- PASS'
13
$ grep -c 'ToLower(u.Scheme)' internal/middleware/cors.go
0
Boot, built ./cmd/api against scratch pg 5443 / redis 6393:
FRONTEND_ORIGIN='https://*.up.railway.app'   -> exit=1  "... (wildcard hosts are not supported: list each preview origin explicitly)"
FRONTEND_ORIGIN='https://app.example.com:443' -> exit=1 "... (default port: browsers omit :443 from Origin, so this entry would never match)"
FRONTEND_ORIGIN='https://app.example.com.'   -> exit=1  "... (trailing dot: browsers send the host without it)"
FRONTEND_ORIGIN='https://app.example.com:8443' -> "cors: allowing [https://app.example.com:8443]", then OPTIONS /api/v1/quests/daily:
  Origin https://app.example.com:8443      -> 204 Access-Control-Allow-Origin: https://app.example.com:8443
  Origin https://app.example.com           -> 403 (no allow-origin)
  Origin https://app.example.com.evil.com  -> 403 (no allow-origin)
```

## Quality
- Tests honest: the lookalike table exercises both the preflight and the actual-request branches; the executor mutation-checked it with a suffix match.
- Gaps probed (temporary test file, removed): `https://app.example.com:`, `https://:8443`, `https://app.example.com:0443`, `https://app.example.com:99999`, `https://bücher.example` are all accepted (bug 1).
- Ordering (bug 2, pre-existing): `ParseOrigins` runs in `main` after the DB connection and migrations, so a bad origin with an unreachable DB reports a dial error instead of the origin error, and a good DB gets migrated before boot is refused. Surfaced while re-running this plan's boot proof.
- Boundaries: `middleware` still knows nothing about routes/users/tables.

## Bugs filed
- `harness/ideas/_inbox/parseorigins-still-accepts-an-empty-port-an-empty-host-a-zer.md` — low.
- `harness/ideas/_inbox/a-malformed-frontend-origin-is-refused-only-after-migrations.md` — low (pre-existing).

## Verdict
pass-with-bugs — the three named shapes are refused with reasons and the lookalike pin is in place; residual never-matching shapes and the validation order are low follow-ups.
