---
idea: harness/ideas/_inbox/parseorigins-accepts-frontend-origin-entries-no-browser-send.md
status: done
priority: medium
merged: false
branch: harness/2026-09-25-medium-parseorigins-accepts-frontend-origin-entries-no-browser-send
worktree: .worktrees/parseorigins-accepts-frontend-origin-entries-no-browser-send
---
# middleware: `ParseOrigins` refuses origins no browser sends, and lookalikes are pinned to 403 — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/parseorigins-accepts-frontend-origin-entries-no-browser-send.md`

**Goal:** A `FRONTEND_ORIGIN` entry that can never equal a browser's `Origin` header — an explicit default port, a wildcard host, a trailing-dot host — fails boot with the entry and the reason quoted, instead of booting "healthy" and answering `403` to every PWA request. The exact-match rule is pinned by a table of lookalike origins so a later "preview support" change cannot loosen it unnoticed.

**Why now (`priority: medium`, raised from the reviewer's low):** Confirmed on `origin/main` today: `backend/internal/middleware/cors.go` `ParseOrigins` validates "is this a bare `scheme://host[:port]`" and stores `strings.ToLower(u.Host)` verbatim; `CORS` matches by exact map lookup. `https://app.example.com:443`, `https://*.up.railway.app` and `https://app.example.com.` all pass, boot logs `cors: allowing [...]`, and every browser preflight then gets `403` — a silent total outage, at the first Railway deploy, which is exactly the day an operator types a wildcard or copies a `:443` from a dashboard. `cors_test.go` today has one disallowed origin (`https://evil.example`). Same-day overlap: none (today's cmd/api plan edits `bodylimit.go`'s comment, not `cors.go`).

**Root cause:** the CORS plan specified the shape of a valid origin from RFC 6454 alone and never asked "would a browser send this string".

**Design decisions:**
1. **Reject, do not normalise.** Dropping `:443` silently would rewrite what the operator typed; a loud boot failure is what this package promises ("boot refuses a malformed list", CODEMAP). Each refusal says why: `default port` / `wildcard` / `trailing dot`.
2. **Only the *default* port for the scheme is refused.** `https://app.example.com:8443` and `http://localhost:443` are legitimate (browsers include a non-default port in `Origin`).
3. **The lookalike table is a test, not code.** `CORS` is already exact; the table (`null`, suffix, prefix, scheme downgrade, explicit default port, trailing slash, upper-case) makes the reviewer's live probe permanent.
4. **The scheme condition is simplified** to `u.Scheme != "http" && u.Scheme != "https"` — `url.Parse` already lower-cases the scheme; the `HTTPS://App.Example.com` row proves it.

**Tech stack:** Go 1.25, Gin, stdlib `net/url`. No new dependencies.

**Run every command from the worktree root** unless a step says otherwise. `rg` is not installed — use `grep -n`. Tests run with `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL` in front.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/middleware/cors.go` | `ParseOrigins`: three new refusals with reasons; scheme condition simplified; `ErrBadOrigin` text extended |
| `backend/internal/middleware/cors_test.go` | `TestParseOrigins` rows; `TestParseOriginsSaysWhy`; `TestLookalikeOriginsGetNoCORSAndA403Preflight` |
| `backend/.env.example` | `FRONTEND_ORIGIN` comment: no default port, no wildcard, no trailing dot |
| `harness/CODEMAP.md` | `middleware` paragraph |

---

## Tasks

### Task 1: `ParseOrigins` refuses what a browser never sends

**Files:**
- Modify: `backend/internal/middleware/cors.go`, `backend/internal/middleware/cors_test.go`

- [ ] **Step 1: Write the failing rows** in `TestParseOrigins`:

```go
		{"https://app.example.com:443", nil, true},          // browsers omit the default port from Origin
		{"http://localhost:80", nil, true},
		{"https://*.up.railway.app", nil, true},             // no wildcards: list each preview origin
		{"https://app.example.com.", nil, true},             // trailing-dot FQDN
		{"https://app.example.com:8443", []string{"https://app.example.com:8443"}, false},
		{"http://localhost:443", []string{"http://localhost:443"}, false}, // 443 is not http's default
```

  and a new `TestParseOriginsSaysWhy`: for `https://app.example.com:443` the error string contains `default port`; for `https://*.up.railway.app` it contains `wildcard`; for `https://app.example.com.` it contains `trailing dot`; all three satisfy `errors.Is(err, ErrBadOrigin)`.
- [ ] **Step 2: Run red:** from `backend/`: `env -u … go test ./internal/middleware/ -run TestParseOrigins -count=1` → the four refusal rows fail (`want error true`).
- [ ] **Step 3: Make them pass.** In `cors.go`, replace the validation condition and the append with:

```go
		u, err := url.Parse(p)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
			u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || u.Opaque != "" {
			return nil, fmt.Errorf("%w: %q", ErrBadOrigin, strings.TrimSpace(part))
		}
		// The three shapes url.Parse accepts that no browser ever puts in an
		// Origin header. Refuse them rather than normalise: boot must say what
		// the operator typed and why it can never match.
		if reason := neverSentByABrowser(u); reason != "" {
			return nil, fmt.Errorf("%w: %q (%s)", ErrBadOrigin, strings.TrimSpace(part), reason)
		}
		out = append(out, u.Scheme+"://"+strings.ToLower(u.Host))
```

  with

```go
// neverSentByABrowser names why a parsed origin can never equal a browser's
// Origin header, or "" when it can. Browsers omit the scheme's default port
// (RFC 6454 §6.1 serialises host:port only when the port is not the default),
// never send a wildcard, and strip a trailing dot.
func neverSentByABrowser(u *url.URL) string {
	host, port := u.Hostname(), u.Port()
	switch {
	case strings.Contains(host, "*"):
		return "wildcard hosts are not supported: list each preview origin explicitly"
	case strings.HasSuffix(host, "."):
		return "trailing dot: browsers send the host without it"
	case (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80"):
		return "default port: browsers omit :" + port + " from Origin, so this entry would never match"
	}
	return ""
}
```

  Update `ErrBadOrigin`'s message to end "…with no path, query, fragment, userinfo, default port, wildcard or trailing dot". (Empty `host` after `Hostname()` cannot happen: `u.Host == ""` was refused above.)
- [ ] **Step 4: Run green:** `env -u … go test ./internal/middleware/ -count=1` → `ok` (the `HTTPS://App.Example.com` row still yields `https://app.example.com`).
- [ ] **Step 5: Commit:** `git commit -am "middleware: ParseOrigins refuses default ports, wildcards and trailing dots with a reason"`.

### Task 2: Lookalike origins are pinned

**Files:**
- Modify: `backend/internal/middleware/cors_test.go`

- [ ] **Step 1: Write the table test** using the file's `newCORSRouter(t, app)` / `do(...)` helpers with the allow-list `[]string{"https://app.example.com"}`:

```go
func TestLookalikeOriginsGetNoCORSAndA403Preflight(t *testing.T) {
	for _, origin := range []string{
		"null",                             // opaque origin (sandboxed iframe, file://)
		"https://app.example.com.evil.com", // suffix
		"https://evilapp.example.com",      // prefix
		"http://app.example.com",           // scheme downgrade
		"https://app.example.com:443",      // explicit default port
		"https://app.example.com/",         // trailing slash
		"HTTPS://APP.EXAMPLE.COM",          // not what a browser sends; must not match either
	} {
		t.Run(origin, func(t *testing.T) {
			r := newCORSRouter(t, app)
			pre := do(t, r, http.MethodOptions, "/api/v1/quests/daily", map[string]string{"Origin": origin, "Access-Control-Request-Method": "GET"})
			if pre.Code != http.StatusForbidden || pre.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("preflight from %q: status %d, allow-origin %q; want 403 and none", origin, pre.Code, pre.Header().Get("Access-Control-Allow-Origin"))
			}
			act := do(t, r, http.MethodGet, "/api/v1/onboarding/quiz", map[string]string{"Origin": origin})
			if act.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("actual request from %q carried Allow-Origin %q", origin, act.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
```

  (Adapt the paths and the `app` handler to what `cors_test.go` already uses — read its first 35 lines.)
- [ ] **Step 2: Run:** `env -u … go test ./internal/middleware/ -run TestLookalike -count=1 -v` → all seven subtests pass on the first run (the code is already exact; this task adds the pin). Then prove the pin bites: temporarily change `if !allow[origin]` in `CORS` to a `strings.HasSuffix(origin, ".example.com")`-style match → the suffix and prefix rows go red; revert.
- [ ] **Step 3: Commit:** `git commit -am "middleware: pin that lookalike origins get no CORS headers and a 403 preflight"`.

### Task 3: Operator docs and CODEMAP

**Files:**
- Modify: `backend/.env.example`, `harness/CODEMAP.md`

- [ ] **Step 1:** `.env.example`, the `FRONTEND_ORIGIN` comment: after "exact scheme://host[:port]" add "— exactly what the browser sends: no default port (`:443` / `:80`), no wildcard (list every preview origin), no trailing dot; boot refuses those and says why".
- [ ] **Step 2:** CODEMAP `middleware` paragraph: after "parsed by `ParseOrigins` — boot refuses a malformed list" add "or an entry no browser ever sends (an explicit default port, a `*` in the host, a trailing dot — each refused with its reason, 2026-09-25)"; after "a preflight from any other origin is `403`" add "(`TestLookalikeOriginsGetNoCORSAndA403Preflight` pins `null`, suffix/prefix hosts, a scheme downgrade, `:443`, a trailing slash and upper-case)".
- [ ] **Step 3:** `python3 tools/harness/cli.py validate` → 0. Commit: `git commit -am "docs: FRONTEND_ORIGIN must be exactly what the browser sends"`.

---

## Verification

```bash
cd backend
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./internal/middleware/ -count=1 -race -v 2>&1 | grep -c '^--- PASS'
# expect: 13 (11 today + TestParseOriginsSaysWhy + TestLookalikeOriginsGetNoCORSAndA403Preflight)
grep -c 'ToLower(u.Scheme)' internal/middleware/cors.go
# expect: 0 (the redundant condition is gone)
grep -n 'neverSentByABrowser' internal/middleware/cors.go
# expect: the func and its one call
make check
# expect: fmt-check silent, vet silent, ok for every package under -race
# Live boot proof, from backend/ with the dev stack vars exported in a subshell (never in your shell):
go build -o /tmp/api-cors ./cmd/api
(set -a; . ./.env; set +a; FRONTEND_ORIGIN='https://*.up.railway.app' /tmp/api-cors); echo "exit=$?"
# expect: one line `config: middleware: FRONTEND_ORIGIN entries must be … : "https://*.up.railway.app" (wildcard hosts are not supported: list each preview origin explicitly)`, exit=1
(set -a; . ./.env; set +a; FRONTEND_ORIGIN='https://app.example.com:443' /tmp/api-cors); echo "exit=$?"
# expect: the default-port reason, exit=1
cd ..
grep -n 'default port' backend/.env.example harness/CODEMAP.md
# expect: one hit in each
git diff --stat origin/main...HEAD -- harness/ | grep -v CODEMAP
# expect: no output
python3 tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0
gh run list --branch "$(git branch --show-current)" --limit 1
# expect: all four jobs green
```

## Notes and open questions

- **Railway previews:** with wildcards refused, a preview environment needs its exact origin in `FRONTEND_ORIGIN`. That is the honest state of the code today; suffix matching would be a feature (and would need the lookalike table extended, not loosened) — for the ideator, not this bug.
- **IPv6 literals** (`http://[::1]:3000`) parse with `Hostname()` = `::1` and are unaffected by the three checks.
- **Not touched:** `CORS` itself, `BodyLimit` (today's cmd/api plan edits its comment).

## Execution summary

Built exactly as planned, no deviations except one mechanical `gofmt` fixup (a fourth commit) that the plan's inline code snippet's comment alignment didn't match gofmt's own column rule for the new `TestParseOrigins` rows.

**Plan verification (from `backend/`, `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL`):**
- `go test ./internal/middleware/ -count=1 -race -v 2>&1 | grep -c '^--- PASS'` → `13` (as expected: 11 pre-existing + `TestParseOriginsSaysWhy` + `TestLookalikeOriginsGetNoCORSAndA403Preflight`).
- `grep -c 'ToLower(u.Scheme)' internal/middleware/cors.go` → `0` (redundant condition removed).
- `grep -n 'neverSentByABrowser' internal/middleware/cors.go` → the func def and its one call site.
- `make check` → `fmt-check` silent, `vet` silent, `go test ./... -count=1 -race` → `ok` for all 13 packages (`cmd/api`, `airouter`, `auth`, `config`, `google`, `health`, `middleware`, `notify`, `onboarding`, `pet`, `quests`, `secrets`, `store`).
- Live boot proof (scratch Postgres/Redis, `docker compose -p bf-origins`, ports 55444/56444): `FRONTEND_ORIGIN='https://*.up.railway.app'` → `config: middleware: FRONTEND_ORIGIN entries must be … : "https://*.up.railway.app" (wildcard hosts are not supported: list each preview origin explicitly)`, `exit=1`. `FRONTEND_ORIGIN='https://app.example.com:443'` → the default-port reason, `exit=1`. Both match the plan's expected text verbatim.
- `grep -n 'default port' backend/.env.example harness/CODEMAP.md` → one hit in each.
- `git diff --stat origin/main...HEAD -- harness/` → only `harness/CODEMAP.md` changed.
- `python3 tools/harness/cli.py validate` → `exit=0`.

**Runtime proof (step 8, beyond the plan's own Verification section):**
- `go build -o /tmp/api-cors ./cmd/api` → built clean.
- Full suite from a clean shell (`env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL`): `make check` green across all packages (above).
- Booted the real binary against the scratch Postgres/Redis stack on a spare port (`PORT=58765`): log showed `cors: allowing [http://localhost:3000]`, then `listening on [::]:58765`. `curl` `GET /healthz` → `200`. `curl` `OPTIONS /api/v1/onboarding/quiz` with `Origin: http://localhost:3000` → `204` with `Access-Control-Allow-Origin: http://localhost:3000` and the full preflight header set. Same request with `Origin: http://localhost:3000:80` (an explicit default port — a lookalike, not the allow-listed origin) → `403`, no `Access-Control-Allow-Origin`.
- Two boot-failure invocations above (wildcard, default port) — the destructive-sounding "refuse to boot" path was checked and does refuse, with the quoted entry and reason.
- Cleanup verified: `kill` the app PID, `docker compose -p bf-origins down`, deleted the scratch `backend/.env`; `pgrep -fl api-cors` and `docker ps --filter name=bf-origins` both empty afterward.

CI on `harness/2026-09-25-medium-parseorigins-accepts-frontend-origin-entries-no-browser-send`: all four jobs green (`harness-tooling`, `backend-unit`, `frontend`, `backend-integration`) — https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36094226541
