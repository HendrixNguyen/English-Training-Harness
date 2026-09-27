---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Backend spec is silent on push fail key, subscription key validation and the 10-subscription cap

## Why
The plan kept `project-base/` off-limits and left the spec update as a follow-up for the reviewer to file. The backend spec is the contract (AGENTS.md: §6 shapes are the contract, keep §4 in sync), and it now disagrees with the code in three places: §4's Redis key table has no `push:fail:{subscription_id}` (String, TTL 7 d); §6.4 `POST /settings/notifications` does not say `p256dh`/`auth` are validated (base64url → 65-byte P-256 point / 16 bytes, else 400) or that a user keeps at most 10 subscriptions with the oldest evicted. A frontend or client author reading the spec cannot know a subscription can be silently evicted or refused.

## Expected output
Backend spec §4 gains the `push:fail:{subscription_id}` row (String, INCR+EXPIRE 7 d on a failed send, DEL on success/prune; not a §4 original — say so, like `pet:revive`). §6.4 gains two bullets in the file's escaped style: key validation → `400 invalid_request`; at most 10 subscriptions per user, oldest by `created_at` evicted; and that `Tick` prunes a subscription after 3 consecutive failed days.

## Evidence
- Plan: `harness/plans/2026-09-27-no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md` (Notes: "Spec follow-up"; Execution summary: "Follow-ups for the reviewer to file").
- Branch @ `273be93`: `backend/internal/store/keys.go` `PushFailKey`/`PushFailTTL`, `backend/internal/notify/subscription_keys.go`, `backend/internal/notify/repo.go` `MaxSubscriptionsPerUser`.
- `grep -n 'push:fail' "project-base/Adaptive English Learning Platform - Backend Technical Specification.md"` → no output.
