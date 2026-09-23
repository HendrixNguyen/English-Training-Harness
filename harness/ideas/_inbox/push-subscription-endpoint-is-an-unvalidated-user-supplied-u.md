---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md
---
# push_subscription.endpoint is an unvalidated user-supplied URL the reminder worker POSTs to (SSRF)

## Why
`POST /api/v1/settings/notifications` accepts `push_subscription.endpoint` as an arbitrary
string and stores it. The in-process reminder worker then makes an outbound HTTP POST to that
exact URL, once per user per day, forever. Nothing between the JSON body and
`http.Client.Do` validates the scheme, the host, or the address it resolves to.

Any authenticated learner (accounts are free — Google OAuth) can therefore point the backend's
own HTTP client at an internal address and have it fire a POST on a schedule the attacker
chooses. On Railway that reaches `*.railway.internal` service names, the API's own
`127.0.0.1:$PORT` routes, Postgres/Redis host ports, and any cloud metadata address the
platform exposes. The request carries a `TTL`, `Urgency`, `Content-Encoding: aes128gcm` and an
`Authorization: vapid t=…,k=…` header signed with the server's VAPID private key, plus an
opaque encrypted body.

It is blind SSRF — the response body is never returned to the caller, only the status code is
branched on — but a 404/410 vs. "anything else" distinction is still a one-bit oracle the
attacker can read back through whether their subscription row survives the next tick, and
POST-with-a-body to an internal endpoint is a write primitive, not just a probe. This is the
class of hole that must be closed before the endpoint is reachable in any deployed
environment, which is why it blocks the branch rather than waiting for the ideation queue.

## Expected output
`endpoint` is validated before it is ever stored, and again before it is ever dialled:

- scheme must be `https` (the Push API only ever issues `https` endpoints; `http`, `file`,
  `gopher`, `redis` and friends are rejected);
- the URL must parse, have a host, and carry no userinfo;
- the resolved address must not be loopback, link-local (`169.254.0.0/16`, `fe80::/10`),
  unique-local, or RFC1918 private — ideally enforced at dial time via a
  `net.Dialer.Control` hook on the sender's `http.Client` so a DNS-rebinding endpoint cannot
  pass validation and then resolve somewhere else at send time;
- optionally, a host allowlist of the real push services
  (`*.push.services.mozilla.com`, `fcm.googleapis.com`, `*.notify.windows.com`,
  `web.push.apple.com`), which is the tightest and simplest form.

A rejected endpoint returns `400 invalid_request` from `POST /settings/notifications` and
nothing is written. Tests: a table of hostile endpoints (`http://`, `https://localhost/x`,
`https://127.0.0.1/x`, `https://169.254.169.254/latest/meta-data/`, `https://10.0.0.5/x`,
`https://[::1]/x`, `https://user:pw@push.example/x`) each asserting 400 and zero repo/queue
calls, plus a dial-time test that a public-looking hostname resolving to a private address is
refused by the sender.

## Evidence
- Plan under review: `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md` (MVP slice 8, reviewed 2026-09-23).
- `backend/internal/notify/handler.go:22-26` — `pushSub.Endpoint` carries only `binding:"required"`; no format, scheme or length constraint.
- `backend/internal/notify/service.go:69-71` — the only service-level check is `sub.Endpoint == ""`.
- `backend/internal/notify/repo.go:97-102` — the raw string is inserted into `push_subscriptions.endpoint` (`TEXT NOT NULL`, `internal/store/migrations/0001_init.up.sql:27`).
- `backend/internal/notify/push.go:72-81` — `webpush.SendNotificationWithContext(... &webpush.Subscription{Endpoint: sub.Endpoint ...})` dials it directly; `push.go:62` is a plain `&http.Client{Timeout: 10 * time.Second}` with the default transport, no dial control.
- `backend/internal/notify/service.go:152-164` — `Tick` sends to every stored subscription on every fire, so one hostile row is a recurring request, not a one-off.
- Concrete request that reproduces it (authenticated):
  `POST /api/v1/settings/notifications` with
  `{"notification_time":"00:01","push_subscription":{"endpoint":"https://169.254.169.254/latest/meta-data/","p256dh":"BNc5T","auth":"aX8v"}}`
  → `200 {"status":"updated",...}`, the row is stored, and the worker POSTs to that address at the next tick past the scheduled score.
- Related inbox precedent for unbounded client-controlled input in this codebase: `harness/ideas/_inbox/duration-seconds-is-unbounded-so-one-request-bricks-a-user-s.md`, `harness/ideas/_inbox/no-post-handler-bounds-the-request-body-so-one-jwt-can-decod.md`.
