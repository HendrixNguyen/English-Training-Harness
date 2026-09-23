---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md
plan: harness/plans/2026-09-23-push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md
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

## Evaluation

**Verdict: selected, `high` (blocker).** The *Why* is real and confirmed read-only in the
worktree at `547c0ab`: `handler.go:22-26` has `binding:"required"` only, `service.go:69-71`
checks `sub.Endpoint == ""` only, `push.go:62` builds `&http.Client{Timeout: 10s}` on the
default transport (redirects followed, no dial control, `Proxy` from environment), and
`push.go:72-81` hands `sub.Endpoint` to `webpush.SendNotificationWithContext`, which does
`http.NewRequest("POST", s.Endpoint, …)` and `client.Do(req)` with no checks of its own
(`webpush-go@v1.4.0/webpush.go:194,243`). `Tick` (`service.go:152-164`) then sends to every
stored row on every fire. Root cause: the endpoint is treated as opaque data when it is a
destination the server dials. Authenticated blind SSRF with an attacker-chosen daily trigger;
the branch must not merge with it.

**Fix model chosen — deny private destinations at the dial, not an allowlist of push services.**
Reasoning recorded in the plan (*Design decisions*): the allowlist is tighter today but bakes
four vendors' hostnames into the backend, breaks self-hosted push services (UnifiedPush and
any future provider), and would need a code change the moment a browser changes its endpoint
host — a dial-time guard on the resolved IP closes the actual hole (dialling something private)
for every future host without maintenance. It is enforced in `net.Dialer.Control`, which Go
calls with the address it is *about to connect to*, so DNS rebinding between subscribe and
send cannot pass. Redirects are not followed (`CheckRedirect` → `http.ErrUseLastResponse`),
and the transport has `Proxy: nil` so the guard always sees the true destination.

**Both doors:** subscribe-time `ValidateEndpoint` (https, parses, host, no userinfo, ≤ 2048
bytes, IP literals checked against the same deny ranges) fails fast with the existing 400
`invalid_request` and writes nothing; send-time runs the same syntactic check plus the dial
guard. Rows already stored (none in any deployed environment — this slice is unmerged) fail at
`Send` with `ErrForbiddenEndpoint`, which `Tick` treats like 404/410: prune the row and
report it once, so a hostile row is gone after the first tick rather than retried daily.

**Deliberate narrowing of the idea's *Expected output*:** `https://localhost/x` is *not*
rejected at subscribe time — no hostname denylist and no DNS lookup in the handler. Name-based
checks at subscribe are either network I/O in a request handler or cosmetic; the honest place
for a name is the dial, where `localhost` (and any rebinding name) is refused on its resolved
loopback address. The plan's test for a DNS name resolving to a private address uses exactly
that route. The optional allowlist in *Expected output* is not adopted (see above).

**Dependencies:** none new — Go stdlib `net`, `net/netip`, `net/url`. **Size:** one new file
plus edits to five existing ones in `backend/internal/notify`; well under a day. Not folded in:
the review's other six findings (separate inbox items; MVP-first standing rule).
