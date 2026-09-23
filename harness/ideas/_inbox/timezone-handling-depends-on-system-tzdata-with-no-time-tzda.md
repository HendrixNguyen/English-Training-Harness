---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# timezone handling depends on system tzdata with no time/tzdata import

## Why
Every timezone decision in `notify` (and in `quests` and `pet`) goes through
`time.LoadLocation`, which reads the *operating system's* zoneinfo database, or `$ZONEINFO`, or
the embedded database only if some package imports `time/tzdata`. Nothing in the backend
imports `time/tzdata`.

On a host without zoneinfo — a `scratch`/`distroless` container image, or a minimal Alpine
without the `tzdata` package — two things happen at once and they point in opposite
directions:

1. `UpdateSettings` rejects **every** valid IANA timezone with `400 invalid_request`
   ("unknown timezone"), so a user cannot set a reminder at all; and
2. `Location()` silently falls back to `time.UTC` for whatever is already in `users.timezone`,
   so existing users' reminders fire at the UTC hour rather than their local hour — a learner
   in Asia/Ho_Chi_Minh gets their 20:00 reminder at 03:00.

The second is the dangerous one: it is silent, and the feature looks like it is working.

Honest caveat: this repository contains no `Dockerfile`, `railway.json` or `nixpacks.toml`, so
I cannot confirm what the production image will be, and the platform's default Go image very
likely does ship zoneinfo. This is a one-line hardening against a failure mode that is
expensive to diagnose, not a demonstrated defect.

## Expected output
`backend/cmd/api/main.go` imports `_ "time/tzdata"`, so the binary carries its own copy of the
zone database and `time.LoadLocation` never depends on the image (the cost is roughly 450 KB of
binary size). A test asserts `Location("Asia/Ho_Chi_Minh")` does not return `time.UTC`, and
fails rather than skips when it does — `backend/internal/quests/day_test.go:116` already
diagnoses this condition in its failure message but there is no test that would catch it in a
built image. Optionally, boot fails fast with a clear error if a known zone cannot be loaded,
so the operator learns at deploy time instead of from wrongly-timed pushes.

## Evidence
- Plan under review: `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md`.
- `backend/internal/notify/service.go:64-68` — `time.LoadLocation(req.Timezone)` failure becomes `ErrInvalidRequest` → the handler's 400.
- `backend/internal/notify/schedule.go:21-30` — `Location` swallows the `LoadLocation` error and returns `time.UTC`, by design ("a bad timezone must never stop a reminder") — which is what makes the failure silent.
- `backend/internal/notify/service.go:119` and `service.go:132` — that `loc` decides both the re-slot time and the `daily:accumulated` local date read.
- `grep -rn 'time/tzdata' backend --include='*.go'` → no import anywhere; the only hits are diagnostic strings (`internal/quests/day_test.go:116`, `internal/store/keys_test.go:35`, `internal/notify/schedule_test.go:78`).
- No deployment image is committed: `find . -maxdepth 3 -iname 'Dockerfile*' -o -iname 'railway*' -o -iname 'nixpacks*' -o -iname 'Procfile'` returns nothing.
- Same class, different symptom, already filed: `harness/ideas/_inbox/zones-that-skip-local-midnight-on-spring-forward-are-never-s.md`.
