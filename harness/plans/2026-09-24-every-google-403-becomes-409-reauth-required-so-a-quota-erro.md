---
idea: harness/ideas/_inbox/every-google-403-becomes-409-reauth-required-so-a-quota-erro.md
status: done
priority: medium
merged: false
branch: harness/2026-09-24-medium-every-google-403-becomes-409-reauth-required-so-a-quota-erro
worktree: .worktrees/every-google-403-becomes-409-reauth-required-so-a-quota-erro
---
# Google sync: quota 403s stop forcing re-consent, unconsumed 409s stop surfacing as 500, and the route logs what Google said — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Bug team — ticket **B1** of 2026-09-24. **Estimate:** 4 h. **Branch:** `harness/2026-09-24-medium-every-google-403-becomes-409-reauth-required-so-a-quota-erro`.

**Idea (head):** `harness/ideas/_inbox/every-google-403-becomes-409-reauth-required-so-a-quota-erro.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/every-google-409-now-becomes-500-internal-error-instead-of-5.md` → Task 3
- `harness/ideas/_inbox/the-google-sync-route-logs-nothing-so-a-502-or-500-discards-.md` → Task 4

**Goal:** `POST /api/v1/integrations/google/sync` answers `409 reauth_required` only when Google actually rejected the grant or the scopes, answers `502 google_unavailable` for quota/throttling 403s, 429s and any 409 the insert path did not consume, and writes one server-side log line per failure carrying Google's reason — while the client keeps receiving only the opaque codes the CODEMAP documents.

**Architecture:** All three fixes stay inside `backend/internal/google`. `doJSON` (`client.go`) gains a tiny decoder for Google's standard error envelope `{"error":{"errors":[{"reason":…}]}}` and uses it only on 403; `SyncHandler` (`handler.go`) gains one switch case and two `log.Printf` calls. No new types cross the package boundary; `ErrReauthRequired`, `ErrAlreadyExists` and `*UpstreamError` keep their meaning, so `service.go` is untouched.

**Tech stack:** Go (see `backend/go.mod`), Gin, stdlib `log` (the package convention — `quests`/`pet`/`airouter` all use `log.Printf`). No new dependencies.

**Spec:** backend spec §6.4 (no error catalogue for this route — the mapping is ours); CODEMAP `google` bullet is the contract clients read.

**Root cause (from the ideas' `## Evaluation`, re-read on `main` @ 9517f25):**
- `client.go:74-75` — `case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:` → `ErrReauthRequired`, body never decoded. Google's Calendar v3 / Tasks v1 return **403** for `rateLimitExceeded`, `userRateLimitExceeded`, `dailyLimitExceeded`, `quotaExceeded`.
- `client.go:82-83` — `case resp.StatusCode == http.StatusConflict:` → `ErrAlreadyExists` for all three services; `handler.go:32-43` has no case for it → catch-all `500 internal_error`.
- `handler.go:31-44` — `err` is switched on and discarded; `internal/google` has zero `log.` statements in production code.

## Global Constraints

- Never edit app code in the main checkout; work in `.worktrees/<slug>` on the branch above (AGENTS.md).
- `rg` and `timeout` are not installed: use `grep -n`, `go test -timeout 60s`.
- The client response bodies stay exactly `{"error":"reauth_required"}`, `{"error":"google_unavailable"}`, `{"error":"internal_error"}` — nothing from Google's body ever reaches the client.
- No refresh token, access token or `Authorization` header value may appear in a log line. None is placed in an error value today (`token.go`, `oauth.go`, bearer lives only in a header) — Task 4's test pins that.
- `go test ./...` must never call Google (all clients are overridden by `httptest` in tests).
- `gofmt -l internal/google` must print nothing (CI runs gofmt since the 2026-09-24 gofmt plan).

## Review Focus

1. A 403 whose body is **not** JSON, or has no `errors[]` (today's `insufficient scopes` fixture) → still `ErrReauthRequired`. Test in Task 1.
2. A 403 `rateLimitExceeded` from **Tasks** (not only Calendar) → `*UpstreamError{Service:"tasks"}`. Test in Task 1.
3. A 429 from any service → `*UpstreamError` (it already does; pin it so a future case never swallows it). Test in Task 1.
4. `ErrAlreadyExists` from `events.insert` is still consumed by `service.go` (patch path) — the handler case must not change that flow. Existing `TestSyncInsertsThenPatchesOn409`-style service tests keep passing; Task 3 runs the whole package.
5. The log line on a `*UpstreamError` includes `Body` — Google bodies can be long; truncate to 512 bytes like `airouter.truncate` does so a verbose upstream cannot flood logs. Test in Task 4.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/google/client.go` | `googleErrorReason(raw []byte) string`; `throttleReasons` set; 403 branch splits reauth vs throttle; 429 explicit; doc comments on `ErrReauthRequired` |
| `backend/internal/google/calendar_test.go` | Extend `TestCalendarMapsStatusesToSentinelErrors` with `rateLimitExceeded`, `insufficientPermissions`, non-JSON 403, 429 |
| `backend/internal/google/tasks_test.go` | Append `TestTasksMapsAQuota403ToUpstream` |
| `backend/internal/google/handler.go` | `case errors.Is(err, ErrAlreadyExists)` → 502; `logSyncFailure(userID, err)` + success info line; `truncate` helper |
| `backend/internal/google/handler_test.go` | Append `TestSyncHandlerMapsAnUnconsumed409To502`, `TestSyncHandlerLogsTheFailureServerSideOnly`, `TestSyncHandlerLogsSuccessWithoutTokens` |
| `harness/CODEMAP.md` | `google` bullet: error mapping sentence rewritten |

Run every command from `backend/` inside the worktree.

---

## Tasks

### Task 1: `doJSON` tells a throttling 403 from an authorization 403

**Files:**
- Modify: `backend/internal/google/client.go:14-18` (doc comment), `:72-85` (switch)
- Test: `backend/internal/google/calendar_test.go` (`TestCalendarMapsStatusesToSentinelErrors`), `backend/internal/google/tasks_test.go` (append)

**Interfaces:**
- Produces: `func googleErrorReason(raw []byte) string` (unexported; returns `error.errors[0].reason` or `""`), `var throttleReasons = map[string]bool{...}`. Error semantics unchanged for callers: `ErrReauthRequired`, `ErrNotFound`, `ErrAlreadyExists`, `*UpstreamError`.

- [ ] **Step 1: Extend the failing status-mapping test** — replace the `answers` map and assertions in `TestCalendarMapsStatusesToSentinelErrors` (`calendar_test.go:147-172`) with:

```go
func TestCalendarMapsStatusesToSentinelErrors(t *testing.T) {
	srv, _ := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{
		"PATCH /calendars/primary/events/gone":       {404, `{"error":{"code":404}}`},
		"PATCH /calendars/primary/events/forbidden":  {403, `{"error":{"code":403,"message":"insufficient scopes"}}`},
		"PATCH /calendars/primary/events/noscope":    {403, `{"error":{"code":403,"errors":[{"reason":"insufficientPermissions"}]}}`},
		"PATCH /calendars/primary/events/throttled":  {403, `{"error":{"code":403,"errors":[{"domain":"usageLimits","reason":"rateLimitExceeded","message":"Rate Limit Exceeded"}]}}`},
		"PATCH /calendars/primary/events/daily":      {403, `{"error":{"code":403,"errors":[{"reason":"dailyLimitExceeded"}]}}`},
		"PATCH /calendars/primary/events/notjson403": {403, `<html>forbidden</html>`},
		"PATCH /calendars/primary/events/toomany":    {429, `{"error":{"code":429,"errors":[{"reason":"rateLimitExceeded"}]}}`},
		"PATCH /calendars/primary/events/broken":     {500, `{"error":{"code":500}}`},
	})
	c := NewHTTPCalendarClient()
	c.BaseURL = srv.URL
	ctx := context.Background()

	if err := c.PatchEvent(ctx, "t", "gone", sampleEvent()); !errors.Is(err, ErrNotFound) {
		t.Errorf("404: err = %v, want ErrNotFound", err)
	}
	for _, id := range []string{"forbidden", "noscope", "notjson403"} {
		if err := c.PatchEvent(ctx, "t", id, sampleEvent()); !errors.Is(err, ErrReauthRequired) {
			t.Errorf("403 %s: err = %v, want ErrReauthRequired", id, err)
		}
	}
	var up *UpstreamError
	for _, tc := range []struct {
		id     string
		status int
	}{{"throttled", 403}, {"daily", 403}, {"toomany", 429}, {"broken", 500}} {
		err := c.PatchEvent(ctx, "t", tc.id, sampleEvent())
		if !errors.As(err, &up) || up.Service != "calendar" || up.Status != tc.status {
			t.Errorf("%s: err = %v, want *UpstreamError{calendar, %d}", tc.id, err, tc.status)
		}
		if errors.Is(err, ErrReauthRequired) {
			t.Errorf("%s: a quota/throttle answer must never be reauth", tc.id)
		}
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test -timeout 60s ./internal/google -run TestCalendarMapsStatusesToSentinelErrors -v`
Expected: FAIL — `throttled: err = google: re-authentication required: calendar returned 403, want *UpstreamError{calendar, 403}` (and `daily` likewise).

- [ ] **Step 3: Implement** — in `client.go`, replace the `ErrReauthRequired` doc comment and the first `switch` case, and add the helper:

```go
// ErrReauthRequired means Google no longer honours this user's refresh token
// (invalid_grant), or rejects the request as unauthorized (401) or as a
// permissions/scope problem (403 without a throttling reason). The handler
// answers 409 reauth_required so the client sends the user through /login
// again (auth.AuthCodeURL asks for calendar.events + tasks with prompt=consent).
//
// A 403 whose error reason is one of throttleReasons is *not* reauth: Calendar
// v3 and Tasks v1 answer quota exhaustion with 403, and re-consenting cannot
// fix a quota. Those, and 429, are UpstreamError (→ 502, "try again later").
var ErrReauthRequired = errors.New("google: re-authentication required")

// throttleReasons are the error.errors[].reason values Google uses for quota
// and rate limiting on Calendar v3 and Tasks v1 (they arrive as 403).
var throttleReasons = map[string]bool{
	"rateLimitExceeded":     true,
	"userRateLimitExceeded": true,
	"dailyLimitExceeded":    true,
	"quotaExceeded":         true,
}

// googleErrorReason returns error.errors[0].reason from Google's standard
// error envelope, or "" when the body is not that shape.
func googleErrorReason(raw []byte) string {
	var env struct {
		Error struct {
			Errors []struct {
				Reason string `json:"reason"`
			} `json:"errors"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil || len(env.Error.Errors) == 0 {
		return ""
	}
	return env.Error.Errors[0].Reason
}
```

and the switch becomes:

```go
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w: %s returned 401", ErrReauthRequired, service)
	case resp.StatusCode == http.StatusForbidden && !throttleReasons[googleErrorReason(raw)]:
		return fmt.Errorf("%w: %s returned 403 %s", ErrReauthRequired, service, googleErrorReason(raw))
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return fmt.Errorf("%w: %s returned %d", ErrNotFound, service, resp.StatusCode)
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: %s returned 409", ErrAlreadyExists, service)
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		// Includes throttling 403s and 429: the handler answers 502.
		return &UpstreamError{Service: service, Status: resp.StatusCode, Body: string(raw)}
	}
```

- [ ] **Step 4: Run the test — it passes; run the package**

Run: `go test -timeout 60s ./internal/google -run TestCalendarMapsStatusesToSentinelErrors -v` → PASS.
Run: `go test -timeout 60s ./internal/google` → PASS (the existing `TestTasksInsertTaskMapsReauth` still passes: its 403 body has no throttle reason).

- [ ] **Step 5: Pin the Tasks side** — append to `tasks_test.go` (helpers `fakeGoogleAPI`/`fakeAnswer` live in `calendar_test.go`, same package):

```go
func TestTasksMapsAQuota403ToUpstreamNotReauth(t *testing.T) {
	srv, _ := fakeGoogleAPI(t, map[string]struct {
		Status int
		Body   string
	}{"POST /lists/l1/tasks": fakeAnswer(403, `{"error":{"code":403,"errors":[{"domain":"usageLimits","reason":"userRateLimitExceeded"}]}}`)})
	c := NewHTTPTasksClient()
	c.BaseURL = srv.URL
	err := c.InsertTask(context.Background(), "tok", "l1", Task{Title: "Day 1"})
	var up *UpstreamError
	if !errors.As(err, &up) || up.Service != "tasks" || up.Status != 403 {
		t.Fatalf("err = %v, want *UpstreamError{tasks, 403}", err)
	}
	if errors.Is(err, ErrReauthRequired) {
		t.Fatal("a Tasks quota 403 must not force re-consent")
	}
}
```

(If `tasks_test.go` does not already import `errors`, add it.) Run: `go test -timeout 60s ./internal/google -run TestTasksMapsAQuota403 -v` → PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -l internal/google && go vet ./internal/google
git add internal/google/client.go internal/google/calendar_test.go internal/google/tasks_test.go
git commit -m "google: a throttling 403 (rateLimitExceeded & co) is UpstreamError, not reauth"
```

### Task 2: The `doJSON` doc comment and `ErrAlreadyExists` comment say what the codes mean now

**Files:**
- Modify: `backend/internal/google/client.go:20-27` (`ErrNotFound`/`ErrAlreadyExists` comments), `:42-45` (`doJSON` comment)

- [ ] **Step 1: Edit the comments** so `ErrAlreadyExists` reads:

```go
// ErrAlreadyExists means Google already holds a resource with the id we sent.
// Only Calendar events.insert with a client id relies on it (Service patches
// its own event). Any other 409 is not consumed and the handler answers 502.
var ErrAlreadyExists = errors.New("google: resource already exists")
```

and the `doJSON` comment ends with "maps the status code as documented on the errors above (401 and non-throttle 403 → ErrReauthRequired; 404/410 → ErrNotFound; 409 → ErrAlreadyExists; everything else non-2xx, including throttling 403 and 429 → *UpstreamError)".

- [ ] **Step 2: Build and commit**

```bash
go build ./... && gofmt -l internal/google
git add internal/google/client.go
git commit -m "google: document the status → error mapping on the sentinels"
```

### Task 3: `SyncHandler` answers 502 for an unconsumed 409

**Files:**
- Modify: `backend/internal/google/handler.go:31-43`
- Test: `backend/internal/google/handler_test.go` (append)

**Interfaces:**
- Consumes: `ErrAlreadyExists` from `client.go`; `newHarness()`, `router`, `post` from the existing tests; `fakeTasks.errs` keyed by method name (`fakes_test.go:97-107`).

- [ ] **Step 1: Write the failing test** (append to `handler_test.go`; `fmt` may need importing):

```go
func TestSyncHandlerMapsAnUnconsumed409To502(t *testing.T) {
	h := newHarness()
	// A 409 from tasklists.insert is not the Calendar insert's "ours already";
	// nothing consumes it, so it must read as "Google is being difficult, retry".
	h.tasks.errs = map[string]error{"InsertTaskList": fmt.Errorf("%w: tasks returned 409", ErrAlreadyExists)}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Fatalf("status %d body %s, want 502 google_unavailable (CODEMAP: other Google failures → 502)", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test -timeout 60s ./internal/google -run TestSyncHandlerMapsAnUnconsumed409To502 -v`
Expected: FAIL — `status 500 body {"error":"internal_error"}`.

- [ ] **Step 3: Implement** — in `handler.go` the switch becomes:

```go
		switch {
		case errors.Is(err, ErrReauthRequired):
			// The refresh token is gone or revoked, or Google rejected the
			// scopes: the client sends the user back through /login
			// (auth.AuthCodeURL re-requests consent).
			c.JSON(http.StatusConflict, gin.H{"error": "reauth_required"})
		case errors.As(err, &up), errors.Is(err, context.DeadlineExceeded), errors.Is(err, ErrAlreadyExists):
			// Quota/throttle, 5xx, our 60 s deadline, or a 409 the insert path
			// did not consume (a concurrent patch, any Tasks conflict): Google
			// was the problem and a retry is the answer — 502, never 500.
			c.JSON(http.StatusBadGateway, gin.H{"error": "google_unavailable"})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		default:
			c.JSON(http.StatusOK, res)
		}
```

- [ ] **Step 4: Run the package** — the Calendar insert→patch service tests must still pass (the handler never sees a consumed `ErrAlreadyExists`).

Run: `go test -timeout 60s ./internal/google` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/google/handler.go internal/google/handler_test.go
git commit -m "google: an unconsumed 409 answers 502 google_unavailable, not 500"
```

### Task 4: The route logs the failure (and the success) server-side, never the token

**Files:**
- Modify: `backend/internal/google/handler.go` (imports `log`, `strings`; new `logSyncFailure`, `truncate`)
- Test: `backend/internal/google/handler_test.go` (append)

**Interfaces:**
- Produces: `func logSyncFailure(userID string, err error)` and `func truncate(s string, n int) string` (unexported).

- [ ] **Step 1: Write the failing tests** (append; import `bytes`, `log`, `strings`):

```go
// captureLog routes the stdlib logger into a buffer for one test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func TestSyncHandlerLogsTheFailureServerSideOnly(t *testing.T) {
	buf := captureLog(t)
	h := newHarness()
	h.oauth.err = &UpstreamError{Service: "oauth", Status: 503, Body: `{"error":"backend_error"}`}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Fatalf("client body must stay opaque: %d %s", w.Code, w.Body)
	}
	got := buf.String()
	for _, want := range []string{"google: sync", "user=u1", "oauth", "503", "backend_error"} {
		if !strings.Contains(got, want) {
			t.Errorf("log %q missing %q", got, want)
		}
	}
	// The refresh token the harness hands out and the derived access token
	// must never be written — they are never in an error value; keep it so.
	for _, secret := range []string{"1//refresh", "access-for-"} {
		if strings.Contains(got, secret) {
			t.Errorf("log leaks a token: %q", got)
		}
	}
}

func TestSyncHandlerLogsA500WithTheCauseAndTruncatesLongBodies(t *testing.T) {
	buf := captureLog(t)
	h := newHarness()
	h.oauth.err = &UpstreamError{Service: "oauth", Status: 502, Body: strings.Repeat("x", 5000)}
	post(t, router(h.svc, "u1"))
	if n := strings.Count(buf.String(), "x"); n > 600 {
		t.Errorf("log carries %d bytes of upstream body, want it truncated to ~512", n)
	}

	buf.Reset()
	h = newHarness()
	h.repo.errs = map[string]error{"Profile": errors.New("pg: connection reset")}
	post(t, router(h.svc, "u1"))
	if !strings.Contains(buf.String(), "connection reset") || !strings.Contains(buf.String(), "user=u1") {
		t.Errorf("500 path must log the cause: %q", buf.String())
	}
}

func TestSyncHandlerLogsSuccessWithoutTokens(t *testing.T) {
	buf := captureLog(t)
	h := newHarness()
	post(t, router(h.svc, "u1"))
	got := buf.String()
	if !strings.Contains(got, "user=u1") || !strings.Contains(got, "tasks=28") {
		t.Errorf("success line missing user/tasks: %q", got)
	}
	if strings.Contains(got, "1//refresh") || strings.Contains(got, "access-for-") {
		t.Errorf("success line leaks a token: %q", got)
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test -timeout 60s ./internal/google -run 'TestSyncHandlerLogs' -v`
Expected: FAIL — `log "" missing "google: sync"`.

- [ ] **Step 3: Implement** — in `handler.go` add imports `log` and `strings`, and:

```go
// logSyncFailure records why a sync failed, server-side only. For an
// UpstreamError that is Google's own reason (service, status, body) — the
// single most useful line when a user reports "sync does nothing". Tokens
// are never part of any error value in this package (token.go, oauth.go);
// TestSyncHandlerLogsTheFailureServerSideOnly keeps it that way.
func logSyncFailure(userID string, err error) {
	var up *UpstreamError
	if errors.As(err, &up) {
		log.Printf("google: sync user=%s failed: %s returned %d: %s", userID, up.Service, up.Status, truncate(up.Body, 512))
		return
	}
	log.Printf("google: sync user=%s failed: %v", userID, err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
```

and in the handler, before the switch: `if err != nil { logSyncFailure(userID, err) }`; in the `default:` branch, before `c.JSON(...)`: `log.Printf("google: sync user=%s ok event=%s tasks=%d", userID, res.CalendarEventID, res.TasksCreatedCount)`.

- [ ] **Step 4: Run the package** → PASS; `gofmt -l internal/google` prints nothing; `go vet ./internal/google` clean.

- [ ] **Step 5: Commit**

```bash
git add internal/google/handler.go internal/google/handler_test.go
git commit -m "google: SyncHandler logs Google's reason server-side; client body stays opaque"
```

### Task 5: CODEMAP states the new contract

**Files:**
- Modify: `harness/CODEMAP.md` (`google` bullet)

- [ ] **Step 1: Rewrite the sentence** `Errors: \`invalid_grant\`/401/403 → 409 \`reauth_required\`; other Google failures or the 60 s \`SyncTimeout\` → 502 \`google_unavailable\`.` to:

`Errors: \`invalid_grant\`, 401, or a 403 whose \`error.errors[].reason\` is **not** a quota reason → 409 \`reauth_required\`; a throttling 403 (\`rateLimitExceeded\`, \`userRateLimitExceeded\`, \`dailyLimitExceeded\`, \`quotaExceeded\`), 429, any other non-2xx, a 409 the Calendar insert did not consume, or the 60 s \`SyncTimeout\` → 502 \`google_unavailable\`; anything else → 500 \`internal_error\`. Every failure is logged once server-side (\`google: sync user=… failed: <service> returned <status>: <body ≤ 512 B>\`) and every success once at info; the client body never carries Google's text.`

- [ ] **Step 2: Commit**

```bash
git add harness/CODEMAP.md
git commit -m "codemap: google sync error mapping and logging"
```

## Verification

From `backend/` in the worktree:

```bash
gofmt -l . ; go vet ./... && go test -timeout 120s ./internal/google -count=1 -v 2>&1 | grep -E '^(=== RUN|--- (PASS|FAIL)|PASS|FAIL|ok)' | grep -E 'Throttle|Quota|Unconsumed409|Logs|MapsStatuses|ok|FAIL'
go test -timeout 300s ./...        # whole backend, no Google calls
```

Expected: no gofmt output; the five new/extended tests PASS; `ok …/internal/google`; whole-backend PASS. Then `git push -u origin <branch>` and confirm the branch's CI run (`gh run list --branch <branch>`) is green on `backend-unit` and `backend-integration`.

## Notes and open questions

- Google's 403 envelope sometimes carries `error.status: "PERMISSION_DENIED"` / `"RESOURCE_EXHAUSTED"` (newer APIs) instead of `errors[].reason`. Calendar v3 and Tasks v1 use the `errors[]` form; if a log line later shows a 403 with an empty reason and a `RESOURCE_EXHAUSTED` status, extend `googleErrorReason` to read `error.status` — do not pre-build it (YAGNI).
- No rate limiting of the sync route itself is added here (the plan's `Notes → Rate/abuse` gap stays open as a separate idea).

## Execution summary

Built and pushed as planned; all 5 tasks landed exactly as written, no deviations from the plan's intent.

**Commits** (branch `harness/2026-09-24-medium-every-google-403-becomes-409-reauth-required-so-a-quota-erro`, based on freshly fetched `origin/main`):
1. `c06961d` — a throttling 403 (`rateLimitExceeded` & co) is `*UpstreamError`, not reauth
2. `f8ed2a9` — document the status → error mapping on the sentinels
3. `2cfb4f3` — an unconsumed 409 answers 502 `google_unavailable`, not 500
4. `4e50505` — `SyncHandler` logs Google's reason server-side; client body stays opaque
5. `a378230` — CODEMAP: google sync error mapping and logging

**Deviations:** none from the plan's tasks/steps. One incidental fix: the plan's Task 4 Step 3 text said to add imports `log` **and** `strings` to `handler.go`, but the `truncate` helper as specified uses only `len`/slicing — `strings` would have been an unused import and failed the build. Added only `log`; `strings` stays in `handler_test.go` where it's actually used (`strings.Contains`, `strings.Repeat`, `strings.Count`).

**Verification output** (from `backend/` in the worktree):

```
$ gofmt -l . ; go vet ./... && go test -timeout 120s ./internal/google -count=1 -v 2>&1 | grep -E '^(=== RUN|--- (PASS|FAIL)|PASS|FAIL|ok)' | grep -E 'Throttle|Quota|Unconsumed409|Logs|MapsStatuses|ok|FAIL'
internal/quests/handler_test.go   # pre-existing, unrelated to this plan (internal/google is clean; see below)
internal/quests/repo.go           # pre-existing, unrelated to this plan
=== RUN   TestCalendarMapsStatusesToSentinelErrors
--- PASS: TestCalendarMapsStatusesToSentinelErrors (0.00s)
=== RUN   TestSyncHandlerMapsAnUnconsumed409To502
--- PASS: TestSyncHandlerMapsAnUnconsumed409To502 (0.00s)
=== RUN   TestSyncHandlerLogsTheFailureServerSideOnly
--- PASS: TestSyncHandlerLogsTheFailureServerSideOnly (0.00s)
=== RUN   TestSyncHandlerLogsA500WithTheCauseAndTruncatesLongBodies
--- PASS: TestSyncHandlerLogsA500WithTheCauseAndTruncatesLongBodies (0.00s)
=== RUN   TestSyncHandlerLogsSuccessWithoutTokens
--- PASS: TestSyncHandlerLogsSuccessWithoutTokens (0.00s)
=== RUN   TestTasksMapsAQuota403ToUpstreamNotReauth
--- PASS: TestTasksMapsAQuota403ToUpstreamNotReauth (0.00s)
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/google	0.503s

$ gofmt -l internal/google   # scoped per the plan's Global Constraints — clean
(no output)

$ go test -timeout 300s ./...
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter	0.599s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth	1.014s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/config	1.589s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/google	2.747s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/health	2.237s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/notify	3.443s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding	4.121s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/pet	6.383s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests	4.891s
ok  	github.com/HendrixNguyen/English-Training-Harness/backend/internal/store	5.511s
```

The `gofmt -l .` hits on `internal/quests/handler_test.go` / `internal/quests/repo.go` are pre-existing on `origin/main` (last touched by commit `267ab95`, unrelated to this branch, no local changes) — outside this plan's scope per its Global Constraints (`gofmt -l internal/google` must print nothing, and it does).

### Runtime proof

Ports/project used per the day's concurrency assignment: API `18081`, Postgres host port `15433`, Redis host port `16380`, `COMPOSE_PROJECT_NAME=b1google` (scratch `backend/.env`, deleted afterward).

```
$ go build -o /tmp/b1google-api ./cmd/api      # clean, no errors

$ docker compose -p b1google up -d --wait --wait-timeout 120
 Container b1google-postgres-1  Healthy
 Container b1google-redis-1  Healthy

$ PORT=18081 DATABASE_URL=postgres://english:english@localhost:15433/english?sslmode=disable \
  REDIS_URL=redis://localhost:16380/0 JWT_SECRET=*** GOOGLE_CLIENT_ID=*** GOOGLE_CLIENT_SECRET=*** \
  /tmp/b1google-api
2026/09/24 23:21:01 migrations applied: [0001_init 0002_google_sync 0003_pet_verdict_dates]
[GIN-debug] POST   /api/v1/integrations/google/sync --> .../internal/google.SyncHandler.func1 (4 handlers)
2026/09/24 23:21:01 listening on :18081

$ curl --max-time 10 -s -o /dev/null -w "healthz: %{http_code}\n" http://localhost:18081/healthz
healthz: 200

$ curl --max-time 10 -s -w "\nsync (no auth): %{http_code}\n" -X POST http://localhost:18081/api/v1/integrations/google/sync
{"error":"unauthorized"}
sync (no auth): 401
```

The binary boots against real Postgres + Redis, applies migrations, and serves a real request through the modified route (auth guard on `SyncHandler` answers correctly). The new 403/409/logging branches themselves are exercised by the `httptest`-backed unit tests above (`go test ./...` never calls Google, per the plan's Global Constraints) — a full authenticated sync would need a live Google account, which is out of scope for a local runtime proof.

**Cleanup verified:** process killed (`pgrep -fl b1google-api` → none), `docker compose -p b1google down` + `docker volume rm b1google_postgres_data`, `docker ps --filter name=b1google` → empty, scratch `backend/.env` and `/tmp/b1google-api*` removed. Worktree `git status` clean before push.

**CI:** pushed `harness/2026-09-24-medium-every-google-403-becomes-409-reauth-required-so-a-quota-erro` to origin. Green: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36026951264 — `backend-unit`, `backend-integration`, `frontend`, `harness-tooling` all passed.
