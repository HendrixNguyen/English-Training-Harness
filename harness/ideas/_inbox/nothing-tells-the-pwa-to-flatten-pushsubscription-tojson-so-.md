---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Nothing tells the PWA to flatten PushSubscription toJSON so slice 9 will send the nested keys shape and get 400

## Why
The backend takes a **flat** `push_subscription: {endpoint, p256dh, auth}` — which is correct:
backend spec §6.4 line 329 shows exactly that shape, and the handler test copies the spec line
verbatim. The *idea* file proposed the nested `subscription: {endpoint, keys: {p256dh, auth}}`
shape and was overruled in favour of the spec, per AGENTS.md precedence. That call is right and
is documented.

The problem is on the other side of the wire. The browser's `PushSubscription.toJSON()`
returns the **nested** shape — `{endpoint, expirationTime, keys: {p256dh, auth}}` — so the
natural frontend implementation is `body: JSON.stringify({notification_time, push_subscription:
subscription.toJSON()})`, and that is a `400 invalid_request` every time. The frontend
technical specification says nothing about this endpoint's body at all; its only mentions of
`p256dh` are the DDL and the ER diagram. MVP slice 9 (the Nuxt PWA) is being planned and built
against these contracts right now, so the trap is live, not hypothetical.

The backend does fail loudly rather than silently — the nested body is rejected with 400 and
there is a test pinning it — so this is a documentation/contract-visibility bug, not a
data-corruption one.

## Expected output
The flattening requirement is stated where a frontend author will actually read it, before
they write the call:

- the frontend technical specification's API→UI mapping (§5) gains a
  `POST /api/v1/settings/notifications` row showing the exact flat body and a one-line note
  that `subscription.toJSON()` must be destructured into `{endpoint, p256dh: keys.p256dh, auth:
  keys.auth}`;
- the same note lands in `harness/CODEMAP.md`'s frontend section (or the notify paragraph gains
  "the PWA must flatten `toJSON()`" rather than only "flat keys per §6.4");
- optionally, the handler accepts the nested shape as an alias and normalises it — cheaper for
  every future client than making each one remember, and it costs one extra struct field. If
  the owner prefers strictness, the 400 body should say *which* fields were missing instead of
  a bare `invalid_request`, so the developer sees the cause immediately.

## Evidence
- Plan under review: `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md`; recorded in its *Notes and open questions* → "**Flat `push_subscription` keys.**"
- `project-base/Adaptive English Learning Platform - Backend Technical Specification.md:329` — the §6.4 request body, flat, verbatim: `{"notification_time": "20:00:00", "push_subscription": {"endpoint": "...", "p256dh": "BNc5T...", "auth": "aX8v..."}}`. The implementation matches the spec; the idea file (`harness/ideas/2026-09-22-run-02/notify-web-push-subscriptions-and-delayed-reminder-queue.md`, *Expected output*) is the one that used the nested shape.
- `backend/internal/notify/handler.go:22-26` — flat `pushSub` struct, all three fields `binding:"required"`.
- `backend/internal/notify/handler_test.go:77` — the `"nested keys (not §6.4)"` case asserts `400 {"error":"invalid_request"}` for the browser-native body.
- `project-base/Adaptive English Learning Platform - Frontend Technical Specification.md` — `grep -in 'pushManager|subscribe|push_subscription|notification_time'` matches nothing; only `p256dh` in the DDL (line 208) and the ER diagram (line 126). The frontend contract for this endpoint does not exist.
- MDN: `PushSubscription.toJSON()` serialises to `{endpoint, expirationTime, keys: {p256dh, auth}}`.
