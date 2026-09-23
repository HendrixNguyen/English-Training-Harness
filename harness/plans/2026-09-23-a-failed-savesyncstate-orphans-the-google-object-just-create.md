---
idea: harness/ideas/_inbox/a-failed-savesyncstate-orphans-the-google-object-just-create.md
status: approved
priority: high
merged: false
---
# google sync: idempotent Calendar insert via a client-supplied event id, and honest docs about what a failed save can orphan — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/a-failed-savesyncstate-orphans-the-google-object-just-create.md`
**Goal:** A `SaveSyncState` failure right after `InsertEvent` (a pool hiccup, or — most plausibly — the client disconnecting during the 60 s sync, which cancels the request context) can no longer leave a second 28-day "English practice" event on the user's calendar: the insert carries a deterministic client-supplied id, so a repeat is a `409` that `Sync` treats as "already ours" and patches. The Tasks half (no client id in the Tasks API) is documented truthfully instead of claimed safe, the fakes gain error hooks so this window is finally testable, and the plan/CODEMAP sentence "a failure never orphans a Google object" is corrected.

**Why now (`priority: high`):** data-integrity bug on the user's own Google account with no in-app remedy (nothing can find or delete the stray event), plus a false invariant written into `harness/CODEMAP.md` and the merged plan's *Architecture* that the next maintainer will trust. Reproduction path on `main`: `service.go:78-87` inserts then saves; `handler.go:28` wraps `c.Request.Context()`, so a disconnect fails the save with `context.Canceled`; the next sync sees `CalendarEventID == ""` and inserts again.

**Root cause (from the idea's `## Evaluation`):** persisting state *between* Google calls protects against a failure of the *later* Google call, not against a failure of the persistence itself, which is the step that makes the id findable. Only an idempotency key on the remote call closes that window. Calendar v3 `events.insert` accepts one (`id`, 5–1024 chars of base32hex `[a-v0-9]`); Tasks v1 `tasklists.insert` does not.

**Folded findings (rejected in the inbox with a pointer here):**
- `the-google-fakes-have-no-error-field-for-eleven-of-sync-s-er.md` → Task 1 (per-method `errs` hooks on all three fakes; handler 500 + deadline tests).
- `no-test-asserts-the-calendar-patch-body-so-an-empty-patch-pa.md` → Task 2 (`fakeCalendar` records the patched `Event`; the PATCH client test asserts the body).

**Architecture:**
- `PracticeEventID(userID)` = `"aelp"` + the user id lower-cased with every character outside `[a-v0-9]` dropped (a UUID's 32 hex digits are a subset of base32hex). Deterministic per user; one practice event per user is the product's model already.
- `Event` gains `ID`; `payload()` sends `"id"` when set and always `"status": "confirmed"` — the latter is what lets a PATCH restore an event the user deleted (Google keeps a deleted event as `cancelled` and its id stays reserved, so a re-insert with that id is a 409, not a new event).
- `doJSON` maps `409` → `ErrAlreadyExists`. `Sync`'s insert path: insert with the deterministic id → on `ErrAlreadyExists`, `PatchEvent(ev.ID)` → if *that* is `ErrNotFound` (the id is reserved but the event is gone for good — a 410 slot), fall back to one insert **without** a client id, i.e. today's behaviour, reachable only after a user deleted the event by hand. State records whichever id was used.
- Tasks: unchanged flow, corrected comment. The list is created then saved; a save failure orphans an empty list and the retry deletes only the stored id. That is now *stated*, in `service.go`, CODEMAP and the merged plan (the evaluator has already appended a correction to the plan on `main`; this branch fixes CODEMAP).

**Tech stack:** Go 1.25, stdlib only (`net/http`, `encoding/json`, `strings`, `regexp` in tests). No new dependencies. `go test ./...` never calls Google (httptest for clients, fakes for the service) — unchanged.

**Run every command from `backend/` inside the worktree.** `rg`/`timeout` are not installed — `grep -n`, `go test -timeout`. `internal/google/fakes_test.go` is one of the three files `gofmt -l` flags on `main`; you edit it, so leave it formatted.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/google/fakes_test.go` | `errs map[string]error` + `fail(method)` on `fakeCalendar`/`fakeTasks`/`fakeRepo`; `fakeRepo.failSaveAt`; `fakeCalendar` tracks known ids (409 on repeat) and records patched `Event`s |
| `backend/internal/google/handler_test.go` | Add `TestSyncHandlerMapsPlainErrorsTo500`, `TestSyncHandlerMapsADeadlineTo502` |
| `backend/internal/google/schedule.go` | `Event.ID`; `payload()` adds `id`/`status`; `PracticeEventID` |
| `backend/internal/google/schedule_test.go` | `TestPracticeEventIDIsDeterministicBase32Hex`, `TestEventPayloadCarriesIDAndConfirmedStatus` |
| `backend/internal/google/client.go` | `ErrAlreadyExists`; `409` mapping in `doJSON` |
| `backend/internal/google/calendar_test.go` | Insert sends `id`+`status`, `409` → `ErrAlreadyExists`; PATCH body asserted |
| `backend/internal/google/service.go` | Insert path with deterministic id / 409 → patch / dead-slot fallback; doc comment corrected |
| `backend/internal/google/service_test.go` | New regression tests; `evt_new` expectations updated to `PracticeEventID("u1")` |
| `harness/CODEMAP.md` | `google` bullet: idempotency sentence corrected |

---

## Tasks

### Task 1: Make every `Sync` error branch reachable — fakes with error hooks, handler 500/502 tests

**Files:**
- Modify: `backend/internal/google/fakes_test.go`
- Modify: `backend/internal/google/handler_test.go`

- [ ] **Step 1: Add the hooks**

In `fakes_test.go`, add one field and one helper to each of `fakeCalendar`, `fakeTasks`, `fakeRepo`:

```go
	errs map[string]error // method name → error returned before doing anything else
```
```go
func (f *fakeRepo) fail(method string) error { return f.errs[method] } // same shape on fakeCalendar and fakeTasks
```

At the top of **every** method on the three fakes, after the `f.log.add(...)` line: `if err := f.fail("<MethodName>"); err != nil { return <zero>, err }` (or `return err`). Keep the existing `patchErr`, `insertErr`/`failAfter` fields and semantics — several tests use them.

Add to `fakeRepo`:
```go
	failSaveAt int   // 1-based index of the SaveSyncState call that fails (0 = never)
	saveErr    error // what it fails with (defaults to errSaveBoom)
	saveCalls  int
```
and in `SaveSyncState`, after the log line:
```go
	f.saveCalls++
	if f.failSaveAt != 0 && f.saveCalls == f.failSaveAt {
		if f.saveErr == nil {
			return errSaveBoom
		}
		return f.saveErr
	}
```
with `var errSaveBoom = errors.New("boom: pool hiccup / client disconnected")` at file scope (import `errors`).

- [ ] **Step 2: Handler tests for the two unexercised branches**

Append to `handler_test.go` (build the service with `newHarness()` and inject through `h.repo.errs`):

```go
func TestSyncHandlerMapsPlainErrorsTo500(t *testing.T) {
	h := newHarness()
	h.repo.errs = map[string]error{"Profile": errors.New("pg: connection reset")}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusInternalServerError || w.Body.String() != `{"error":"internal_error"}` {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestSyncHandlerMapsADeadlineTo502(t *testing.T) {
	h := newHarness()
	h.repo.errs = map[string]error{"Profile": context.DeadlineExceeded}
	w := post(t, router(h.svc, "u1"))
	if w.Code != http.StatusBadGateway || w.Body.String() != `{"error":"google_unavailable"}` {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}
```
(add `context` and `errors` to the imports).

- [ ] **Step 3: Run, format, commit**

Run: `gofmt -l ./internal/google` → empty. `go test ./internal/google/... -count=1 -timeout 120s` → `ok` (existing tests unchanged in behaviour; the hooks default to nil).

```bash
git add internal/google/fakes_test.go internal/google/handler_test.go
git commit -m "google: fakes gain per-method error hooks; handler 500 and deadline cases covered"
```

---

### Task 2: Deterministic event id, `status: confirmed`, `409 → ErrAlreadyExists`, PATCH body pinned

**Files:**
- Modify: `backend/internal/google/schedule_test.go`, `schedule.go`
- Modify: `backend/internal/google/calendar_test.go`, `client.go`

- [ ] **Step 1: Failing tests**

Append to `schedule_test.go`:

```go
func TestPracticeEventIDIsDeterministicBase32Hex(t *testing.T) {
	const uuid = "A0EEBC99-9C0B-4EF8-BB6D-6BB9BD380A11"
	id := PracticeEventID(uuid)
	if id != PracticeEventID(uuid) {
		t.Fatal("not deterministic")
	}
	// Calendar v3 events.insert: id is 5–1024 chars of base32hex, i.e. [a-v0-9].
	if !regexp.MustCompile(`^[a-v0-9]{5,1024}$`).MatchString(id) {
		t.Fatalf("id %q is not base32hex", id)
	}
	if id != "aelpa0eebc999c0b4ef8bb6d6bb9bd380a11" {
		t.Fatalf("id = %q", id)
	}
	if PracticeEventID("u1") == PracticeEventID("u2") {
		t.Fatal("two users share an id")
	}
}

func TestEventPayloadCarriesIDAndConfirmedStatus(t *testing.T) {
	ev := sampleEvent()
	if _, has := ev.payload()["id"]; has {
		t.Fatal("payload sends an id when none is set")
	}
	ev.ID = "aelpu1"
	p := ev.payload()
	if p["id"] != "aelpu1" || p["status"] != "confirmed" {
		t.Fatalf("payload = %v", p)
	}
}
```
(`sampleEvent` lives in `calendar_test.go`, same package; import `regexp`.)

Append to `calendar_test.go`:

```go
func TestCalendarInsertSendsTheClientIDAndMaps409ToAlreadyExists(t *testing.T) {
	ok := fakeAnswer(200, `{"id":"aelpu1"}`)
	dup := fakeAnswer(409, `{"error":{"code":409,"message":"The requested identifier already exists.","errors":[{"reason":"duplicate"}]}}`)
	for _, tc := range []struct {
		name   string
		answer struct{ Status int; Body string }
		wantErr error
	}{{"first insert", ok, nil}, {"repeat insert", dup, ErrAlreadyExists}} {
		t.Run(tc.name, func(t *testing.T) {
			srv, calls := fakeGoogleAPI(t, map[string]struct{ Status int; Body string }{"POST /calendars/primary/events": tc.answer})
			c := NewHTTPCalendarClient()
			c.BaseURL = srv.URL
			ev := sampleEvent()
			ev.ID = "aelpu1"
			id, err := c.InsertEvent(context.Background(), "ya29.tok", ev)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && id != "aelpu1" {
				t.Fatalf("id = %q", id)
			}
			body := (*calls)[0].Body
			if body["id"] != "aelpu1" || body["status"] != "confirmed" {
				t.Fatalf("insert body = %v", body)
			}
		})
	}
}
```
Write a tiny `fakeAnswer(status int, body string) struct{ Status int; Body string }` helper beside `fakeGoogleAPI` (or inline the struct literals — match the file's existing style).

Then extend `TestCalendarPatchEventUsesTheStoredID`: after the existing Method/Path assertion, add

```go
	body := got.Body
	start, _ := body["start"].(map[string]any)
	if body["summary"] != EventSummary || body["status"] != "confirmed" || start["timeZone"] != "Asia/Ho_Chi_Minh" || start["dateTime"] == nil || body["end"] == nil {
		t.Fatalf("patch body = %v — a PATCH that drops the payload must fail this test", body)
	}
	if rec, _ := body["recurrence"].([]any); len(rec) != 1 || rec[0] != Recurrence {
		t.Fatalf("recurrence = %v", body["recurrence"])
	}
```

- [ ] **Step 2: Run — expect compile failures on `PracticeEventID`, `Event.ID`, `ErrAlreadyExists`**

Run: `go test ./internal/google/... -count=1 -timeout 120s -run 'PracticeEventID|EventPayload|Calendar'`

- [ ] **Step 3: Implement**

`schedule.go` — add `ID string` to `Event` (doc: "client-supplied Calendar id; empty lets Google assign one"), extend `payload()`:

```go
	p := map[string]any{
		"summary":     e.Summary,
		"description": e.Description,
		"start":       map[string]string{"dateTime": e.Start.Format(time.RFC3339), "timeZone": e.TimeZone},
		"end":         map[string]string{"dateTime": e.End.Format(time.RFC3339), "timeZone": e.TimeZone},
		"recurrence":  []string{Recurrence},
		// A PATCH with status confirmed restores an event the user deleted
		// (Google keeps it as "cancelled" and reserves its id).
		"status": "confirmed",
	}
	if e.ID != "" {
		p["id"] = e.ID
	}
	return p
```
and add:

```go
// PracticeEventID is the client-supplied Calendar id of a user's recurring
// practice block: "aelp" + the user id lower-cased with every character
// outside base32hex ([a-v0-9]) dropped — a UUID's 32 hex digits are a subset.
// Calendar v3 events.insert accepts 5–1024 such characters. Deterministic per
// user, so a repeat insert (after a SaveSyncState failure or a client
// disconnect) is a 409 from Google rather than a second event: that is what
// makes the insert idempotent.
func PracticeEventID(userID string) string {
	var b strings.Builder
	b.WriteString("aelp")
	for _, r := range strings.ToLower(userID) {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'v') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
```

`client.go` — beside `ErrNotFound`:

```go
// ErrAlreadyExists means Google already holds a resource with the id we sent
// (Calendar events.insert with a client id → 409). Sync treats it as ours.
var ErrAlreadyExists = errors.New("google: resource already exists")
```
and in `doJSON`'s switch, before the generic non-2xx case:
```go
	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%w: %s returned 409", ErrAlreadyExists, service)
```

- [ ] **Step 4: Run the package; the existing service tests will now expect a Google id where a deterministic one is used — that is Task 3. Run only the client/schedule tests here:**

`go test ./internal/google/... -count=1 -timeout 120s -run 'PracticeEventID|EventPayload|Calendar' -v` → all `PASS`.

- [ ] **Step 5: Commit**

```bash
git add internal/google/schedule.go internal/google/schedule_test.go internal/google/client.go internal/google/calendar_test.go
git commit -m "google: deterministic client-supplied practice event id; 409 → ErrAlreadyExists; PATCH body pinned"
```

---

### Task 3: `Sync` inserts with the deterministic id, patches on 409, and says what it cannot protect

**Files:**
- Modify: `backend/internal/google/fakes_test.go` (`fakeCalendar` realism)
- Modify: `backend/internal/google/service_test.go`
- Modify: `backend/internal/google/service.go`

- [ ] **Step 1: Make `fakeCalendar` behave like Google about ids**

```go
type fakeCalendar struct {
	log      *callLog
	nextID   string
	patchErr error // returned by PatchEvent (e.g. ErrNotFound)
	errs     map[string]error
	known    map[string]bool // ids Google has seen: a repeat insert is a 409, like the real API
	inserted []Event
	patched  []patchedEvent // id + the body sent (was []string)
}

type patchedEvent struct {
	ID string
	Ev Event
}

func (f *fakeCalendar) InsertEvent(_ context.Context, tok string, ev Event) (string, error) {
	f.log.add("calendar.InsertEvent(%s,%s)", tok, ev.ID)
	if err := f.fail("InsertEvent"); err != nil {
		return "", err
	}
	id := ev.ID
	if id == "" {
		id = f.nextID
	}
	if f.known == nil {
		f.known = map[string]bool{}
	}
	if f.known[id] {
		return "", fmt.Errorf("%w: calendar returned 409", ErrAlreadyExists)
	}
	f.known[id] = true
	f.inserted = append(f.inserted, ev)
	return id, nil
}

func (f *fakeCalendar) PatchEvent(_ context.Context, tok, id string, ev Event) error {
	f.log.add("calendar.PatchEvent(%s,%s)", tok, id)
	if err := f.fail("PatchEvent"); err != nil {
		return err
	}
	if f.patchErr != nil {
		return f.patchErr
	}
	f.patched = append(f.patched, patchedEvent{ID: id, Ev: ev})
	return nil
}
```
Update the two existing uses of `h.cal.patched[0]` (`service_test.go:98-118`, `TestResyncSameRoadmapPatchesEventAndCreatesNothing`) to `h.cal.patched[0].ID`, and add there: `if h.cal.patched[0].Ev.Start != <the expected 20:00 Asia/Ho_Chi_Minh start> { t.Errorf(...) }` — compute it with `PracticeEvent(h.now, "20:00:00", "Asia/Ho_Chi_Minh")` (this is the service-level half of the folded PATCH-body finding).

- [ ] **Step 2: The regression tests**

Append to `service_test.go`:

```go
// The idea's scenario: the client disconnects after InsertEvent, so the save
// that would make the id findable fails. Before this fix the retry inserted a
// second 28-day event; now the deterministic id makes the retry a 409 → patch.
func TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent(t *testing.T) {
	h := newHarness()
	h.repo.failSaveAt = 1

	if _, err := h.svc.Sync(context.Background(), "u1"); !errors.Is(err, errSaveBoom) {
		t.Fatalf("first sync err = %v, want the save failure", err)
	}
	if len(h.cal.inserted) != 1 || h.repo.state.CalendarEventID != "" {
		t.Fatalf("after the failed save: inserted=%d state=%+v", len(h.cal.inserted), h.repo.state)
	}

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	want := PracticeEventID("u1")
	if len(h.cal.inserted) != 1 {
		t.Fatalf("retry inserted a second event: %v", h.log.calls)
	}
	if len(h.cal.patched) != 1 || h.cal.patched[0].ID != want {
		t.Fatalf("retry should patch %s once, got %+v", want, h.cal.patched)
	}
	if res.CalendarEventID != want || h.repo.state.CalendarEventID != want {
		t.Fatalf("id not recorded: res=%+v state=%+v", res, h.repo.state)
	}
}

// A user deleted the event by hand and Google has let the id go entirely
// (PATCH → 404/410): fall back to one Google-assigned id rather than failing forever.
func TestAReservedButGoneIDFallsBackToAGoogleAssignedInsert(t *testing.T) {
	h := newHarness()
	h.cal.known = map[string]bool{PracticeEventID("u1"): true}
	h.cal.patchErr = ErrNotFound
	h.cal.nextID = "evt_fresh"

	res, err := h.svc.Sync(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if res.CalendarEventID != "evt_fresh" || len(h.cal.inserted) != 1 || h.cal.inserted[0].ID != "" {
		t.Fatalf("res=%+v inserted=%+v", res, h.cal.inserted)
	}
}

// The Tasks half has no idempotency key: state this rather than pretend.
func TestAFailedSaveAfterTheListInsertOrphansTheListAndIsDocumented(t *testing.T) {
	h := newHarness()
	h.repo.failSaveAt = 2 // the save right after InsertTaskList

	if _, err := h.svc.Sync(context.Background(), "u1"); !errors.Is(err, errSaveBoom) {
		t.Fatalf("err = %v", err)
	}
	h.tasks.nextList = "list_second"
	if _, err := h.svc.Sync(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	// Two lists were created and none deleted: the first is orphaned. This is the
	// documented gap (service.go Sync doc, CODEMAP google) — if a future change
	// closes it (e.g. tasklists.list by title), update this test to assert one list.
	if len(h.tasks.tasks) != 2 || len(h.tasks.deleted) != 0 {
		t.Fatalf("lists=%d deleted=%v", len(h.tasks.tasks), h.tasks.deleted)
	}
}
```

Also update every `"evt_new"` expectation for the *first-time* insert path in `service_test.go` (8 occurrences across the file, including the call-log prefixes `repo.SaveSyncState(evt=evt_new,…`) to `PracticeEventID("u1")` (`"aelpu1"`). Keep `evt_old` / stored-id tests as they are — a stored Google id is patched unchanged. `TestResyncReinsertsTheEventWhenGoogleLostIt` now expects the re-insert to carry the deterministic id.

- [ ] **Step 3: Run — the three new tests and the updated expectations must fail against the old service**

`go test ./internal/google/... -count=1 -timeout 120s -run 'Sync|Resync' -v` → `TestAFailedSaveAfterTheEventInsertDoesNotCreateASecondEvent` fails with "retry inserted a second event" (the bug), the `evt_new` updates fail on the id. If the second-event assertion passes, stop.

- [ ] **Step 4: Implement in `service.go`**

Replace the insert block (`if state.CalendarEventID == "" { … }`) with:

```go
	if state.CalendarEventID == "" {
		// Client-supplied id: if a previous attempt inserted and then failed to
		// save (pool hiccup, client disconnect cancelling the request context),
		// this insert is a 409 and we patch our own event instead of adding a
		// second one. status: confirmed in the payload also restores an event
		// the user deleted (Google keeps it as cancelled with the id reserved).
		ev.ID = PracticeEventID(userID)
		id, err := s.cal.InsertEvent(ctx, access, ev)
		if errors.Is(err, ErrAlreadyExists) {
			id, err = ev.ID, s.cal.PatchEvent(ctx, access, ev.ID, ev)
			if errors.Is(err, ErrNotFound) {
				// The id is reserved but the event is gone for good (410): one
				// insert with a Google-assigned id — today's non-idempotent path,
				// reachable only after a user deleted the event by hand.
				ev.ID = ""
				id, err = s.cal.InsertEvent(ctx, access, ev)
			}
		}
		if err != nil {
			return Result{}, err
		}
		state.CalendarEventID = id
	}
```

Replace `Sync`'s doc comment with the truth:

```go
// Sync pushes the recurring practice event and the per-day task list.
// Idempotency: the Calendar event is inserted with a deterministic client id
// (PracticeEventID), so a repeat insert is a 409 that is patched, and a
// stored id is patched (re-inserted only when Google lost it). The Tasks list
// is rebuilt only when the active roadmap changed. State is persisted after
// the event, after the list is created and after the task inserts.
//
// What that does NOT protect: a failure of SaveSyncState itself right after
// tasklists.insert (Tasks has no client-supplied id) orphans an empty list
// the retry cannot find — it deletes only the stored id. The Calendar half
// is covered by the deterministic id.
```

- [ ] **Step 5: Whole package, formatted, then whole suite**

```bash
gofmt -l ./internal/google   # must print nothing
go test ./internal/google/... -count=1 -timeout 120s -v 2>&1 | grep -E '^(--- |ok|FAIL)'
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s
```
Expected: every test `PASS`; every package `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/google/service.go internal/google/service_test.go internal/google/fakes_test.go
git commit -m "google: insert the practice event with a deterministic id; 409 → patch; document the Tasks gap"
```

---

### Task 4: CODEMAP tells the truth

**Files:**
- Modify: `harness/CODEMAP.md` → `**google**` bullet

- [ ] **Step 1: Replace the idempotency sentence**

Replace, in the `google` bullet, the sentence

> Idempotent via `google_sync` (migration `0002`, one row per user: …): re-sync **patches** the event (re-inserts on 404/410), leaves Tasks alone when `roadmap_id` matches the active roadmap, and deletes + rebuilds the list for a new roadmap; state is saved after the event and after the list is created, before the task inserts, so a failure never orphans a Google object.

with

> Idempotent via `google_sync` (migration `0002`, one row per user: `calendar_event_id`, `tasklist_id`, `roadmap_id`, `tasks_created_count`) **and** a deterministic client-supplied Calendar id — `PracticeEventID(userID)` = `aelp` + the user id in base32hex — so a repeat `events.insert` (after `SaveSyncState` failed, e.g. the client disconnected mid-sync) is a 409 that `Sync` patches (`payload()` always sends `status: confirmed`, which also restores a user-deleted event; a 404 on that patch falls back to one Google-assigned insert). Re-sync patches the stored event (re-inserts on 404/410), leaves Tasks alone when `roadmap_id` matches the active roadmap, and deletes + rebuilds the list for a new roadmap. **Known gap:** Tasks has no client id, so a `SaveSyncState` failure right after `tasklists.insert` orphans an empty list the retry cannot find (`TestAFailedSaveAfterTheListInsertOrphansTheListAndIsDocumented` pins it). Fakes carry per-method `errs` hooks, so every `Sync` error branch is testable.

- [ ] **Step 2: Commit**

```bash
git add ../harness/CODEMAP.md
git commit -m "CODEMAP: google idempotency — what the deterministic id covers and what Tasks cannot"
```

---

## Verification

From `backend/` in the worktree:

1. `go build ./... && go vet ./... && test -z "$(gofmt -l ./internal/google)"` → exit 0.
2. `env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 300s` → every package `ok`.
3. **Mutation check (paste the output):** in `service.go`, temporarily change `ev.ID = PracticeEventID(userID)` to `ev.ID = ""` and run `go test ./internal/google/... -run 'FailedSaveAfterTheEventInsert' -count=1` → must FAIL with "retry inserted a second event". Restore; `git diff --quiet internal/google/service.go` clean.
4. `grep -n "never orphans" internal/google/service.go ../harness/CODEMAP.md` → no matches.
5. Live boot (dev stack on non-default ports, unique compose project, AGENTS.md): `curl --max-time 5 /healthz` → 200; `POST /api/v1/integrations/google/sync` with a valid session for a user with no refresh token → `409 {"error":"reauth_required"}` (the one path exercisable without Google). Clean up.
6. With `TEST_DATABASE_URL` exported: `go test ./internal/google/... -run Integration -count=1 -p 1 -timeout 120s` → `TestIntegrationSyncStateIsOneRowPerUser` PASS (unchanged, must still pass).
7. Push; `gh run watch` — all four CI jobs green.

## Notes and open questions

- **Google-side assumption the tests cannot prove:** that `events.patch` on a *cancelled* (user-deleted) event with `status: confirmed` restores it, and that a repeat `events.insert` with a used id returns `409`. Both are Google's documented behaviour (events resource `id` semantics: "the ID … cannot be reused"; `status` is writable), and the code degrades safely if either is wrong — a 404 on the patch falls back to a fresh insert, and an unexpected non-409 error surfaces as `502 google_unavailable`. The first real re-sync after a manual delete is worth watching in the logs (see `the-google-sync-route-logs-nothing…`, selected).
- **Existing users:** anyone already synced has a Google-assigned id stored; they keep patching it. Only users whose `calendar_event_id` is empty get the deterministic id. No migration.
- **Tasks orphan:** the honest fix would be `tasklists.list` + delete-by-title before creating, or a `pending_tasklist_id` write-ahead column. Both grow the API surface; neither is in this plan. `a-user-deleted-tasks-list-is-never-rebuilt…` (selected, low) is the natural home for the `tasklists.list` call if it is ever added.
- **Merged plan text:** `harness/plans/2026-09-23-google-one-way-calendar-and-tasks-sync.md` carries an evaluator correction (2026-09-23) under its *Architecture* paragraph; plan files are ROOT bookkeeping, so the executor does not touch it on the branch.
- **Conflicts:** none with `main.go`. `every-google-403-becomes-409…` and `the-google-sync-route-logs-nothing…` edit `client.go`/`handler.go` — merge this first; they are small rebases.
