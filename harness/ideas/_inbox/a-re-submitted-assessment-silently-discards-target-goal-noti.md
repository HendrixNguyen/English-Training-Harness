---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# A re-submitted assessment silently discards target_goal notification_time and timezone and still answers status success

## Why
`POST /api/v1/onboarding/assessment` writes four `users` columns —
`cefr_current`, `target_goal`, `timezone`, `notification_time` (backend spec
§6.1 request body). When a roadmap is already active, `Assess` returns before
any of them is written (`service.go:48-60`) and still answers
`{"status": "success", ...}` with HTTP 200. Nothing tells the caller that three
of the four fields it just sent were thrown away.

`timezone` is not cosmetic: `quests` derives `day_number` from it
(`internal/quests/repo.go:66`, `SELECT COALESCE(timezone, 'UTC') FROM users`)
and `pet`'s hourly sweep decides whose local midnight it is from the same
column. A user who completes onboarding in the wrong timezone, notices, and
re-submits gets "success" and stays in the wrong timezone — their day rolls over
at the wrong hour and their plant is penalised on the wrong schedule, with no
error anywhere to explain it. `notification_time` has the same shape: the
`POST /api/v1/settings/notifications` endpoint (§6.4) that would fix it is not
built yet, so today there is no other writer at all.

The plan acknowledged that retaking the placement is out of scope
("Retaking the placement", *Notes and open questions*), but the note is about
regenerating the *roadmap*. Silently dropping the three profile fields — and
reporting success for it — is a separate, undesigned behaviour.

## Expected output
- A re-submitted assessment either updates `target_goal`, `timezone` and
  `notification_time` (keeping the existing roadmap and CEFR level untouched),
  or it refuses with an explicit status and error code, e.g. `409
  roadmap_already_exists`. It does not answer `status: "success"` for a request
  it partly ignored.
- If the update path is chosen, the write happens in a transaction like the
  create path, and `timezone`/`notification_time` are validated the same way.
- A test pins the chosen behaviour: submitting a second assessment with a
  different `timezone` either changes `users.timezone` or is rejected — it
  cannot pass silently.

## Evidence
- Plan: `harness/plans/2026-09-22-onboarding-placement-test-cefr-grading-and-roadmap-generatio.md`.
- Early return that skips all four column writes:
  `backend/internal/onboarding/service.go:48-60` — only `Profile` and
  `Pet.Ensure` run; `repo.SaveAssessment` (the sole writer of those columns,
  `repo.go:106`) is never reached.
- Response is nevertheless `status: "success"`, HTTP 200:
  `backend/internal/onboarding/service.go:59` and
  `backend/internal/onboarding/handler.go:59`.
- Downstream consumers of the discarded `timezone`:
  `backend/internal/quests/repo.go:66`, and `pet`'s sweep (see CODEMAP's
  **pet** bullet, "for users whose `users.timezone` is in local hour 0").
- Spec: backend §6.1 lists `target_goal`, `notification_time` and `timezone` as
  request fields of this endpoint; nothing in the spec says they are ignored on
  a repeat call.
- Concrete input: submit a valid assessment, then submit again with
  `"timezone": "UTC"` instead of `"Asia/Ho_Chi_Minh"` → 200 `success`,
  `users.timezone` unchanged.
- Test gap: `TestAssessIsIdempotentWhileARoadmapIsActive`
  (`service_test.go:86-104`) asserts `len(h.repo.saved) != 0` is a failure — it
  pins the discard as intended rather than questioning it.
