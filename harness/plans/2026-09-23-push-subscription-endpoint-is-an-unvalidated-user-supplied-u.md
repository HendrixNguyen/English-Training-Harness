---
idea: harness/ideas/_inbox/push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md
status: approved
priority: high
merged: false
amends: harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md
---
# Notify amend: validate `push_subscription.endpoint` at subscribe and refuse private destinations at the dial (SSRF) — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use the executing-plans (or subagent-driven-development) skill to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea:** `harness/ideas/_inbox/push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md` (BLOCKER, high)
**Amends:** `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md` (MVP slice 8, `done`, awaiting merge). This plan lands on that plan's branch
`harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue` in its existing worktree
`.worktrees/notify-web-push-subscriptions-and-delayed-reminder-queue` (HEAD `547c0ab`). **No new branch, no new worktree.** Every task edits files that already exist there, plus one new file pair in `backend/internal/notify`.
**Review that found it:** `harness/reviews/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md` (verdict `pass-with-bugs`; bug 1 is the blocker).

**Goal:** Make it impossible for `POST /api/v1/settings/notifications` to store, and for the reminder worker to dial, a push endpoint that points at a private, loopback, link-local or otherwise internal address — rejected with the existing `400 {"error":"invalid_request"}` at subscribe, and refused at the TCP dial (on the *resolved* IP) at send — without changing the §6.4 wire shape or touching any other finding from that review.

**Run every command from `backend/` inside the worktree** (`cd .worktrees/notify-web-push-subscriptions-and-delayed-reminder-queue/backend`) unless the step says otherwise. `rg` and `timeout` are not installed: use `grep -n`, bound tests with `go test -timeout`. Do not rebase or merge anything into the branch. Commit per task; end every commit message, after a blank line, with the `Co-Authored-By` trailer your session's attribution instructions give you.

---

## Design decisions (read before Task 1)

### 1. Validation model: deny private destinations at the dial — not an allowlist of push services

Two options were on the table:

- **Allowlist** the four real push-service hosts (`fcm.googleapis.com`, `*.push.services.mozilla.com`, `web.push.apple.com`, `*.notify.windows.com`). Strictest, and it would make DNS rebinding moot (the name is fixed and owned by the vendor).
- **Deny-range** check: scheme must be `https`, and the destination must not be loopback / link-local / RFC1918 / CGNAT / ULA / unspecified / multicast / reserved — enforced on the IP actually being connected to.

**Chosen: deny-range, enforced at dial time.** Reasons:

1. The hole is *dialling something internal*, not *dialling a non-vendor*. The deny-range rule closes exactly that for every present and future host; the allowlist closes it as a side effect of a much broader restriction.
2. The allowlist bakes four vendors' hostnames into a backend that spec §2 never scoped to those vendors. It breaks UnifiedPush / self-hosted push distributors (a real use case for a privacy-minded PWA) and any provider added later, and a browser vendor moving its endpoint host (Mozilla has, twice) becomes a production outage fixed by a code deploy.
3. The deny-range check is one pure function on `netip.Addr` — the same size as an allowlist, and it needs no maintenance list.

What this costs: a permitted public host that *later* resolves to a public IP the attacker controls is still reachable — which is exactly the same as any browser user subscribing from any push service, i.e. the Web Push threat model, not SSRF. Accepted.

### 2. DNS rebinding — the check runs inside `net.Dialer.Control`

Validating the hostname at subscribe time protects nothing hours later: the worker re-resolves at send, and an attacker's name can answer `1.2.3.4` at subscribe and `169.254.169.254` at send. So the authoritative check is **`net.Dialer.Control`**, which Go invokes with the literal `ip:port` it is about to `connect()` to, *per attempt*, after resolution. There is no window between check and connect. Dual-stack names (e.g. `localhost` → `::1` and `127.0.0.1`) are checked on every attempted address.

The transport also sets **`Proxy: nil`**. With the default `http.ProxyFromEnvironment`, an `HTTPS_PROXY` in the environment would make the dialer connect to the proxy and the guard would see the proxy's IP, not the destination. Railway does not need an egress proxy for push services; if one is ever required this is the line to revisit, and the guard must move behind it.

### 3. Both doors, and pre-existing rows

- **Subscribe** (`Service.UpdateSettings`): `ValidateEndpoint` — parses, `https` only, has a host, no userinfo, ≤ `MaxEndpointLength` (2048) bytes, and an IP-literal host is run through the same deny-range predicate. Fails fast with the existing `ErrInvalidRequest` → `400 invalid_request`; **nothing is written** (asserted on the call log, as the quests amend did). **No DNS lookup and no hostname denylist here** — a name-based check at subscribe is either network I/O inside a request handler or a false sense of security; `https://localhost/x` therefore passes subscribe and is refused at the dial (deviation from the idea's *Expected output*, recorded in the idea's `## Evaluation`).
- **Send** (`WebPushSender.Send`): runs `ValidateEndpoint` again (cheap; covers stored rows syntactically — `http://` rows never leak a VAPID JWT over plaintext), then dials through the guarded client (`newPushHTTPClient`).
- **Pre-existing rows:** in every deployed environment there are **zero** — this slice has never been merged. For developer databases and defence in depth, a stored endpoint the guard refuses fails `Send` with `ErrForbiddenEndpoint`; `Tick` treats that like 404/410 — **prune the row** (`repo.DeleteSubscription`) and count it in `Pruned` — and additionally **reports it** in the joined error so the worker's `log.Printf` records it once. A hostile row is gone after the first tick instead of being retried daily forever. No migration, no backfill.

### 4. Redirects

`http.Client` follows up to 10 redirects by default, so a permitted host answering `302 Location: https://169.254.169.254/…` would defeat a subscribe-time check. Two layers, both in `newPushHTTPClient`:

- `CheckRedirect` returns `http.ErrUseLastResponse`: the 3xx is returned to `Send` as-is, `Send` treats it as `push service returned 302` (a non-gone failure, retried tomorrow). Push services never redirect; RFC 8030 §5 gives them no reason to.
- Even if a redirect *were* followed, the follow-up request goes through the **same guarded transport**, so a private redirect target is refused at its dial. The dial guard is the load-bearing control; the redirect policy is belt-and-braces and gives a clearer error.

### 5. Proportionate

This is a security fix to a `done` slice. Not in scope: per-user subscription caps, `Tick` concurrency, atomic `Due`+re-slot, `-race` in CI, the PWA flattening note, tzdata embedding — all separate inbox items. `main.go`, `config`, `repo.go`, `queue.go`, `schedule.go`, `worker.go`, migrations and the §6.4 DTO are untouched.

### Effects on existing tests (deliberate, listed so nothing is "discovered" mid-task)

- `push_test.go` uses `httptest.NewServer` (`http://127.0.0.1:…`). After Task 3, `Send`'s URL layer rejects both the `http://` scheme and the `127.0.0.1` literal, and the guarded client refuses the loopback dial. The three request-shape tests move to `httptest.NewTLSServer`, address it as `https://localhost:<port>`, and use an unguarded client trusting the test CA under `ServerName: "example.com"` (httptest's cert does not list `localhost` — checked in `net/http/internal/testcert`). Those tests prove RFC 8291/8292 encoding, not the guard, and the guard has its own tests. Say so in a comment (Task 3 gives the helpers).
- `handler_test.go`'s `spec64Body` uses the spec's placeholder literal `"push_subscription_endpoint_string"`, which is not a URL and now 400s. Replace it with a real-shaped FCM endpoint and note that the spec's example value is a placeholder, not a contract.
- `service_test.go` fixtures use `https://push.example/ep1` — a public-looking https name; unaffected.
- `integration_test.go` goes through `Repo` directly with `https://push.example.test/…`; unaffected.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/notify/endpoint.go` | **new** — `ErrForbiddenEndpoint`, `MaxEndpointLength`, `ValidateEndpoint`, `forbiddenAddr`, `guardDial`, `newPushHTTPClient` |
| `backend/internal/notify/endpoint_test.go` | **new** — deny tables (URL layer, dial layer), permitted real endpoints, localhost-at-dial, redirect |
| `backend/internal/notify/service.go` | `UpdateSettings` calls `ValidateEndpoint`; `Tick` prunes + reports `ErrForbiddenEndpoint` |
| `backend/internal/notify/service_test.go` | hostile-endpoint table asserting zero calls; forbidden-row prune test |
| `backend/internal/notify/handler.go` | **no change** (error mapping already exists) — listed so the executor does not "improve" it |
| `backend/internal/notify/handler_test.go` | `spec64Body` gets a URL-shaped endpoint; hostile-endpoint HTTP table → 400 |
| `backend/internal/notify/push.go` | `NewWebPushSender` uses `newPushHTTPClient()`; `Send` validates before encrypting |
| `backend/internal/notify/push_test.go` | TLS test servers + `srv.Client()`; Send-level refusal tests |
| `backend/internal/notify/fakes_test.go` | `fakeSender.forbidden` map |
| `harness/CODEMAP.md` | `notify` line: endpoint validation, dial guard, no redirects, forbidden → prune |

---

## Tasks

### Task 1: `ValidateEndpoint` and the deny-range predicate (pure)

**Files:** create `backend/internal/notify/endpoint.go`, `backend/internal/notify/endpoint_test.go`.

- [ ] **Step 1: Write the failing tests.**

```go
package notify

import (
	"errors"
	"net/netip"
	"strings"
	"testing"
)

// hostileEndpoints is the reviewer's table plus every range forbiddenAddr
// names. Each must be refused by ValidateEndpoint (URL layer) — the IP-literal
// ones are also refused by guardDial in TestGuardDialRefusesPrivateAddresses.
var hostileEndpoints = map[string]string{
	"http scheme":            "http://fcm.googleapis.com/fcm/send/abc",
	"no scheme":              "fcm.googleapis.com/fcm/send/abc",
	"file scheme":            "file:///etc/passwd",
	"userinfo":               "https://user:pw@fcm.googleapis.com/fcm/send/abc",
	"no host":                "https:///fcm/send/abc",
	"not a url":              "not a url at all",
	"spec placeholder":       "push_subscription_endpoint_string",
	"ipv4 loopback":          "https://127.0.0.1/x",
	"ipv4 loopback high":     "https://127.255.255.254/x",
	"ipv6 loopback":          "https://[::1]/x",
	"link-local metadata":    "https://169.254.169.254/latest/meta-data/",
	"ipv6 link-local":        "https://[fe80::1]/x",
	"rfc1918 10/8":           "https://10.0.0.5/x",
	"rfc1918 172.16/12":      "https://172.16.0.1/x",
	"rfc1918 172.31":         "https://172.31.255.254/x",
	"rfc1918 192.168/16":     "https://192.168.1.1/x",
	"cgnat 100.64/10":        "https://100.64.0.1/x",
	"cgnat 100.127":          "https://100.127.255.254/x",
	"ipv6 ula fc00::/7":      "https://[fd00::1]/x",
	"ipv6 ula fc":            "https://[fc00::1]/x",
	"unspecified v4":         "https://0.0.0.0/x",
	"unspecified v6":         "https://[::]/x",
	"this-network 0/8":       "https://0.1.2.3/x",
	"ietf protocol 192.0.0":  "https://192.0.0.1/x",
	"benchmark 198.18/15":    "https://198.19.0.1/x",
	"reserved 240/4":         "https://240.0.0.1/x",
	"broadcast":              "https://255.255.255.255/x",
	"multicast v4":           "https://224.0.0.1/x",
	"multicast v6":           "https://[ff02::1]/x",
	"ipv4-mapped private":    "https://[::ffff:10.0.0.5]/x",
	"ipv4-mapped loopback":   "https://[::ffff:127.0.0.1]/x",
	"ipv4-mapped link-local": "https://[::ffff:169.254.169.254]/x",
	"too long":               "https://fcm.googleapis.com/fcm/send/" + strings.Repeat("a", MaxEndpointLength),
}

// realEndpoints are the shapes the four browser push services actually issue
// (tokens shortened). All must pass the URL layer.
var realEndpoints = []string{
	"https://fcm.googleapis.com/fcm/send/dA1b2C3d4E5:APA91bHqZ-example",
	"https://updates.push.services.mozilla.com/wpush/v2/gAAAAABk-example",
	"https://web.push.apple.com/QGtuZXhhbXBsZQ-example",
	"https://wns2-par02p.notify.windows.com/w/?token=AwYAAAB-example",
	"https://push.example/ep1",                 // the service_test fixture
	"https://ntfy.example.org/up/abc123",       // self-hosted UnifiedPush distributor
	"https://fcm.googleapis.com:443/fcm/send/x", // explicit port
}

func TestValidateEndpointRefusesHostileURLs(t *testing.T) {
	for name, raw := range hostileEndpoints {
		err := ValidateEndpoint(raw)
		if !errors.Is(err, ErrForbiddenEndpoint) {
			t.Errorf("%s (%q): err = %v, want ErrForbiddenEndpoint", name, raw, err)
		}
	}
}

func TestValidateEndpointAcceptsRealPushServiceURLs(t *testing.T) {
	for _, raw := range realEndpoints {
		if err := ValidateEndpoint(raw); err != nil {
			t.Errorf("%q: unexpected %v", raw, err)
		}
	}
	// A DNS name is a dial-time decision, not a subscribe-time one (Design §3).
	if err := ValidateEndpoint("https://localhost/x"); err != nil {
		t.Errorf("localhost is decided at the dial, not here: %v", err)
	}
}

func TestForbiddenAddrBoundaries(t *testing.T) {
	forbidden := []string{
		"127.0.0.1", "::1", "169.254.169.254", "169.254.0.1", "fe80::1",
		"10.0.0.0", "10.255.255.255", "172.16.0.0", "172.31.255.255", "192.168.0.0", "192.168.255.255",
		"100.64.0.0", "100.127.255.255", "fc00::", "fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"0.0.0.0", "::", "0.255.255.255", "192.0.0.8", "198.18.0.1", "198.19.255.255", "240.0.0.1", "255.255.255.255",
		"224.0.0.1", "ff02::1", "::ffff:10.0.0.5", "::ffff:127.0.0.1",
	}
	permitted := []string{
		"8.8.8.8", "1.1.1.1", "142.250.31.188", "172.15.255.255", "172.32.0.0", "100.63.255.255", "100.128.0.0",
		"198.17.255.255", "198.20.0.0", "9.255.255.255", "11.0.0.0", "2001:4860:4860::8888", "2a00:1450:4001:80b::200a",
		"::ffff:8.8.8.8",
	}
	for _, s := range forbidden {
		if !forbiddenAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must be forbidden", s)
		}
	}
	for _, s := range permitted {
		if forbiddenAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must be permitted", s)
		}
	}
}
```

- [ ] **Step 2: Run them — they must fail to compile** (`ValidateEndpoint`, `forbiddenAddr`, `ErrForbiddenEndpoint`, `MaxEndpointLength` undefined).

```bash
go test ./internal/notify/... -run 'ValidateEndpoint|ForbiddenAddr' -timeout 60s
```

- [ ] **Step 3: Implement `endpoint.go` (URL layer + predicate only; the dial client is Task 3).**

```go
package notify

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
)

// ErrForbiddenEndpoint means a push endpoint is not something this server
// will dial: wrong scheme, malformed, or pointing at a private, loopback,
// link-local or otherwise internal address. UpdateSettings wraps it in
// ErrInvalidRequest (→ 400); Tick treats it like ErrSubscriptionGone (prune).
//
// This is the SSRF boundary: endpoint is client-supplied and the worker POSTs
// to it daily, so it is validated when stored AND enforced on the resolved
// address at dial time (guardDial) — DNS can change between the two.
var ErrForbiddenEndpoint = errors.New("notify: forbidden push endpoint")

// MaxEndpointLength bounds the stored URL. Real push endpoints are 100–300
// bytes; 2048 is the conventional URL ceiling.
const MaxEndpointLength = 2048

// ValidateEndpoint is the syntactic, no-network check: https, parses, has a
// host, no userinfo, bounded, and an IP-literal host is not a forbidden
// address. Hostnames are deliberately NOT resolved here — that decision
// belongs to guardDial, on the address actually being connected to.
func ValidateEndpoint(raw string) error {
	if len(raw) > MaxEndpointLength {
		return fmt.Errorf("%w: longer than %d bytes", ErrForbiddenEndpoint, MaxEndpointLength)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrForbiddenEndpoint, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q, want https", ErrForbiddenEndpoint, u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("%w: userinfo not allowed", ErrForbiddenEndpoint)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%w: no host", ErrForbiddenEndpoint)
	}
	if ip, err := netip.ParseAddr(host); err == nil && forbiddenAddr(ip) {
		return fmt.Errorf("%w: %s is not a public address", ErrForbiddenEndpoint, ip)
	}
	return nil
}

// Explicit IPv4 ranges the netip predicates do not cover.
var forbiddenPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),      // "this" network
	netip.MustParsePrefix("100.64.0.0/10"),  // CGNAT (RFC 6598)
	netip.MustParsePrefix("192.0.0.0/24"),   // IETF protocol assignments
	netip.MustParsePrefix("198.18.0.0/15"),  // benchmarking (RFC 2544)
	netip.MustParsePrefix("240.0.0.0/4"),    // reserved + broadcast
}

// forbiddenAddr reports whether ip is an address this server must never
// dial for a push endpoint. IPv4-mapped IPv6 is unmapped first so
// ::ffff:10.0.0.5 is judged as 10.0.0.5.
func forbiddenAddr(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return true
	}
	for _, p := range forbiddenPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
```

Note: `netip.Addr.IsPrivate` covers 10/8, 172.16/12, 192.168/16 **and** fc00::/7 (ULA); `IsLoopback` covers 127/8 and ::1; `IsLinkLocalUnicast` covers 169.254/16 and fe80::/10. Do not duplicate those in `forbiddenPrefixes`.

- [ ] **Step 4: Run — all three pass.**

```bash
go vet ./internal/notify/... && go test ./internal/notify/... -run 'ValidateEndpoint|ForbiddenAddr' -v -count=1 -timeout 60s
```

- [ ] **Step 5: Mutation check (do not commit the mutation).** Comment out the `100.64.0.0/10` prefix → `TestForbiddenAddrBoundaries` and the two CGNAT rows of `TestValidateEndpointRefusesHostileURLs` must fail. Change `u.Scheme != "https"` to `u.Scheme == ""` → the `http scheme` row must fail. Revert both, re-run Step 4.

- [ ] **Step 6: Commit.**

```bash
git add internal/notify/endpoint.go internal/notify/endpoint_test.go
git commit -m "notify: ValidateEndpoint — https only, no userinfo, bounded, IP literals checked against private/loopback/link-local/CGNAT/ULA ranges"
```

---

### Task 2: Subscribe door — `UpdateSettings` refuses hostile endpoints before writing

**Files:** `backend/internal/notify/service.go`, `service_test.go`, `handler_test.go`.

- [ ] **Step 1: Failing service test** — append to `service_test.go`:

```go
func TestUpdateSettingsRefusesHostileEndpointsBeforeWriting(t *testing.T) {
	for name, raw := range hostileEndpoints {
		h := newHarness()
		_, err := h.svc.UpdateSettings(context.Background(), "u1", SettingsRequest{
			NotificationTime: "20:00",
			Subscription:     &Subscription{Endpoint: raw, P256dh: "BNc5T", Auth: "aX8v"},
		})
		if !errors.Is(err, ErrInvalidRequest) || !errors.Is(err, ErrForbiddenEndpoint) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest wrapping ErrForbiddenEndpoint", name, err)
		}
		if len(h.log.calls) != 0 {
			t.Errorf("%s: rejected endpoint still made calls %v", name, h.log.calls)
		}
		if len(h.queue.scores) != 0 || len(h.repo.subs["u1"]) != 0 {
			t.Errorf("%s: rejected endpoint was stored or scheduled", name)
		}
	}
}
```

- [ ] **Step 2: Failing handler test** — in `handler_test.go`, change the `spec64Body` constant's endpoint to a URL and add a hostile table:

```go
// spec64Body is the §6.4 request. The spec's example endpoint is the
// placeholder "push_subscription_endpoint_string", which is not a URL; a real
// FCM-shaped endpoint stands in for it because endpoints are validated.
const spec64Body = `{"notification_time": "20:00:00", "push_subscription": {"endpoint": "https://fcm.googleapis.com/fcm/send/dA1b2C3:APA91b-example", "p256dh": "BNc5T...", "auth": "aX8v..."}}`
```

Update the stored-endpoint assertion in `TestSettingsHandlerAcceptsTheSpec64BodyAndAnswersTheSpec64Response` to the same string. Then:

```go
func TestSettingsHandlerRejectsHostileEndpointsWith400AndWritesNothing(t *testing.T) {
	// The reviewer's reproduction, verbatim, plus one per layer of the deny list.
	for name, endpoint := range map[string]string{
		"metadata service (review repro)": "https://169.254.169.254/latest/meta-data/",
		"loopback":                        "https://127.0.0.1:8080/api/v1/healthz",
		"ipv6 loopback":                   "https://[::1]/x",
		"rfc1918":                         "https://10.0.0.5/x",
		"cgnat":                           "https://100.64.0.1/x",
		"plain http":                      "http://fcm.googleapis.com/fcm/send/abc",
		"userinfo":                        "https://u:p@fcm.googleapis.com/fcm/send/abc",
	} {
		h := newHarness()
		body := `{"notification_time":"00:01","push_subscription":{"endpoint":"` + endpoint + `","p256dh":"BNc5T","auth":"aX8v"}}`
		w := post(t, router(h.svc, "u1"), body)
		if w.Code != http.StatusBadRequest || w.Body.String() != `{"error":"invalid_request"}` {
			t.Errorf("%s: status = %d, body = %s", name, w.Code, w.Body)
		}
		if len(h.log.calls) != 0 {
			t.Errorf("%s: 400 but the service still called %v", name, h.log.calls)
		}
	}
}
```

- [ ] **Step 3: Run — the new tests fail (200 / no ErrForbiddenEndpoint in chain).**

```bash
go test ./internal/notify/... -run 'HostileEndpoints|Spec64Body' -timeout 60s
```

- [ ] **Step 4: Implement** — in `service.go` `UpdateSettings`, directly after the existing "push_subscription needs endpoint, p256dh and auth" check and before any repo call:

```go
	if req.Subscription != nil {
		if err := ValidateEndpoint(req.Subscription.Endpoint); err != nil {
			return SettingsResult{}, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
		}
	}
```

(`fmt.Errorf` with two `%w` is fine on Go 1.20+; both sentinels are reachable with `errors.Is`.) `handler.go` needs no change — `ErrInvalidRequest` already maps to `400 invalid_request`.

- [ ] **Step 5: Run the whole notify package** — everything green, including the existing `TestUpdateSettingsRejectsBadInputBeforeWriting` (its `"https://e"` case still fails for the missing keys, before the endpoint check — keep the order: keys presence, then `ValidateEndpoint`).

```bash
go vet ./internal/notify/... && go test ./internal/notify/... -count=1 -timeout 60s
```

- [ ] **Step 6: Mutation check.** Move the `ValidateEndpoint` call to *after* `s.repo.UpdatePreferences` → `TestUpdateSettingsRefusesHostileEndpointsBeforeWriting` must fail on `rejected endpoint still made calls [repo.UpdatePreferences(...)]`. Revert.

- [ ] **Step 7: Commit.**

```bash
git add internal/notify/service.go internal/notify/service_test.go internal/notify/handler_test.go
git commit -m "notify: UpdateSettings refuses hostile push endpoints with 400 before any write"
```

---

### Task 3: Send door — guarded HTTP client (dial-time deny on the resolved IP, no redirects) and `Send` validation

**Files:** `backend/internal/notify/endpoint.go`, `endpoint_test.go`, `push.go`, `push_test.go`.

- [ ] **Step 1: Failing tests.** Append to `endpoint_test.go`:

```go
func TestGuardDialRefusesPrivateAddresses(t *testing.T) {
	// guardDial receives what net.Dialer is about to connect() to: the
	// RESOLVED ip:port, after DNS. This is the layer that defeats rebinding.
	for _, addr := range []string{
		"127.0.0.1:443", "[::1]:443", "169.254.169.254:80", "[fe80::1]:443",
		"10.0.0.5:443", "172.16.0.1:443", "192.168.1.1:443", "100.64.0.1:443",
		"[fd00::1]:443", "0.0.0.0:443", "[::ffff:10.0.0.5]:443",
	} {
		if err := guardDial("tcp", addr, nil); !errors.Is(err, ErrForbiddenEndpoint) {
			t.Errorf("%s: err = %v, want ErrForbiddenEndpoint", addr, err)
		}
	}
	if err := guardDial("tcp", "not-an-ip:443", nil); !errors.Is(err, ErrForbiddenEndpoint) {
		t.Errorf("unparseable dial address must be refused, got %v", err)
	}
}

func TestGuardDialAllowsPublicAddresses(t *testing.T) {
	for _, addr := range []string{"142.250.31.188:443", "[2a00:1450:4001:80b::200a]:443", "1.1.1.1:443"} {
		if err := guardDial("tcp", addr, nil); err != nil {
			t.Errorf("%s: unexpected %v", addr, err)
		}
	}
}

// A DNS name that resolves to a private address must be refused at the dial
// even though it passes the URL layer. "localhost" is the one such name every
// machine resolves without a network, and httptest gives us a live listener
// on it; the handler counter proves no connection was completed.
func TestPushHTTPClientRefusesANameResolvingToLoopback(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))

	client := newPushHTTPClient()
	if err := ValidateEndpoint("https://localhost:" + port + "/x"); err != nil {
		t.Fatalf("precondition: the URL layer must let a hostname through: %v", err)
	}
	_, err := client.Post("https://localhost:"+port+"/x", "application/octet-stream", nil)
	if !errors.Is(err, ErrForbiddenEndpoint) {
		t.Fatalf("err = %v, want ErrForbiddenEndpoint from the dial guard (a TLS/x509 error here means the dial went through)", err)
	}
	if hits.Load() != 0 {
		t.Errorf("server handled %d request(s); the dial must be refused before any connection", hits.Load())
	}
}

func TestPushHTTPClientDoesNotFollowRedirects(t *testing.T) {
	var redirected atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) {
		// A "permitted" origin bouncing to somewhere else. The target is
		// same-origin only so a regression fails fast instead of dialling
		// a real link-local address; the dial guard covers the target too.
		http.Redirect(w, r, "/redirected", http.StatusFound)
	})
	mux.HandleFunc("/redirected", func(w http.ResponseWriter, _ *http.Request) {
		redirected.Add(1)
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	// Keep the client's redirect policy, swap only the transport for one that
	// trusts the test CA and may dial loopback (the guard is tested above).
	client := newPushHTTPClient()
	client.Transport = srv.Client().Transport

	resp, err := client.Post(srv.URL+"/push", "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status = %d, want the 302 handed back unfollowed", resp.StatusCode)
	}
	if redirected.Load() != 0 {
		t.Errorf("/redirected was hit %d time(s); redirects must not be followed", redirected.Load())
	}
}
```

Add the imports `net`, `net/http`, `net/http/httptest`, `sync/atomic` to `endpoint_test.go`.

Append to `push_test.go`:

```go
// Send-level: the sender built by NewWebPushSender must refuse before dialling.
func TestWebPushSenderRefusesAForbiddenEndpointBeforeDialling(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))

	s := newTestSender(t) // default HTTPClient: the guarded one
	// `want` pins WHICH layer answered: the URL-layer rows would also be
	// refused by the dial guard (loopback), so the message text is what
	// proves Send validated before dialling.
	for name, tc := range map[string]struct{ endpoint, want string }{
		"ip literal, URL layer": {srv.URL + "/x", "not a public address"}, // https://127.0.0.1:port
		"plain http, URL layer": {"http://127.0.0.1:" + port + "/x", "want https"},
		"dns name, dial layer":  {"https://localhost:" + port + "/x", "refusing to dial"},
	} {
		err := s.Send(context.Background(), browserSubscription(t, tc.endpoint), Payload{Title: "t"})
		if !errors.Is(err, ErrForbiddenEndpoint) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want ErrForbiddenEndpoint containing %q", name, err, tc.want)
		}
		if errors.Is(err, ErrSubscriptionGone) {
			t.Errorf("%s: forbidden must not masquerade as gone", name)
		}
	}
	if hits.Load() != 0 {
		t.Errorf("push server handled %d request(s); nothing may be sent to a forbidden endpoint", hits.Load())
	}
}

func TestWebPushSenderReturnsARedirectAsAFailureNotAFollow(t *testing.T) {
	var redirected atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/redirected", http.StatusFound) })
	mux.HandleFunc("/redirected", func(w http.ResponseWriter, _ *http.Request) { redirected.Add(1); w.WriteHeader(http.StatusCreated) })
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	s := newTestSender(t)
	// The URL layer must pass the endpoint, so dial by hostname, not srv.URL's
	// IP literal. httptest's certificate covers 127.0.0.1, ::1, example.com
	// and *.example.com — NOT localhost (checked: net/http/internal/testcert)
	// — so verify it under the example.com name. Keep CheckRedirect; swap
	// only the transport (unguarded, trusts the test CA).
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	tr := srv.Client().Transport.(*http.Transport).Clone()
	tr.TLSClientConfig.ServerName = "example.com"
	client := newPushHTTPClient()
	client.Transport = tr
	s.HTTPClient = client
	sub := browserSubscription(t, "https://localhost:"+port+"/push")

	err := s.Send(context.Background(), sub, Payload{Title: "t"})
	if err == nil || errors.Is(err, ErrSubscriptionGone) || !strings.Contains(err.Error(), "302") {
		t.Errorf("err = %v, want a non-gone failure mentioning 302", err)
	}
	if redirected.Load() != 0 {
		t.Errorf("/redirected was hit %d time(s)", redirected.Load())
	}
}
```

Also convert the three existing request-shape tests in `push_test.go` (`…PostsAnEncryptedVAPIDSignedRequest`, `…ReportsGoneOn404And410`, `…ReportsOtherFailuresAsErrors`). After Task 3 they fail twice over: `httptest.NewServer` is `http://` (refused by `Send`'s URL layer) and `srv.URL` is the IP literal `127.0.0.1` (refused by the URL layer too, before any client is used). So each becomes `httptest.NewTLSServer`, addresses the server as `https://localhost:<port>/…`, and uses an unguarded client that trusts the test CA under the `example.com` name. Add two helpers once, above the first of them:

```go
// These tests prove the RFC 8291/8292 request shape, not the dial guard, so
// they run over an unguarded client that trusts the httptest CA. The guard
// has its own tests in endpoint_test.go and below.
//
// Send's URL layer refuses IP literals, so the server is addressed as
// "localhost:<port>"; httptest's certificate covers 127.0.0.1, ::1,
// example.com and *.example.com — not localhost (net/http/internal/testcert)
// — hence the ServerName override.
func testTLSClient(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()
	tr := srv.Client().Transport.(*http.Transport).Clone()
	tr.TLSClientConfig.ServerName = "example.com"
	return &http.Client{Transport: tr, Timeout: 5 * time.Second}
}

func localhostURL(t *testing.T, srv *httptest.Server, path string) string {
	t.Helper()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	if err != nil {
		t.Fatal(err)
	}
	return "https://localhost:" + port + path
}
```

Then in each of the three: `srv := httptest.NewTLSServer(…)`, `s := newTestSender(t); s.HTTPClient = testTLSClient(t, srv)`, and `browserSubscription(t, localhostURL(t, srv, "/push/abc"))` (or `"/"`). Use the same two helpers in the two new Send-level tests above instead of their inline `SplitHostPort`/`Clone()` code. Add imports `net`, `sync/atomic`, `time` to `push_test.go`.

- [ ] **Step 2: Run — fails to compile (`guardDial`, `newPushHTTPClient` undefined).**

```bash
go test ./internal/notify/... -run 'GuardDial|PushHTTPClient|WebPushSender' -timeout 60s
```

- [ ] **Step 3: Implement the guarded client** — append to `endpoint.go` (add imports `net`, `net/http`, `syscall`, `time`):

```go
// guardDial is the net.Dialer.Control hook: Go calls it with the address it
// is about to connect() to — the RESOLVED ip:port, per attempt — so a name
// that resolved somewhere public at subscribe time and somewhere private at
// send time (DNS rebinding) is still refused here.
func guardDial(_ string, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("%w: dial address %q: %v", ErrForbiddenEndpoint, address, err)
	}
	if forbiddenAddr(ap.Addr()) {
		return fmt.Errorf("%w: refusing to dial %s", ErrForbiddenEndpoint, ap.Addr())
	}
	return nil
}

// pushClientTimeout is the whole-request bound the original slice chose.
const pushClientTimeout = 10 * time.Second

// newPushHTTPClient is the only HTTP client the sender may use for push
// endpoints: every TCP connect passes guardDial, redirects are handed back
// unfollowed (a permitted host must not be able to bounce us to a private
// one), and Proxy is nil so the guard always sees the true destination —
// with ProxyFromEnvironment it would see the proxy's address instead.
func newPushHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: pushClientTimeout, Control: guardDial}
	return &http.Client{
		Timeout: pushClientTimeout,
		Transport: &http.Transport{
			Proxy:               nil,
			DialContext:         dialer.DialContext,
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: pushClientTimeout,
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}
```

- [ ] **Step 4: Wire it into `push.go`.**

In `NewWebPushSender`, replace `HTTPClient: &http.Client{Timeout: 10 * time.Second},` with `HTTPClient: newPushHTTPClient(),`. Update the `WebPushSender` doc comment: "HTTPClient defaults to newPushHTTPClient (dial guard, no redirects); tests may replace it."

In `Send`, before `json.Marshal`:

```go
	// Stored rows predate no validation in any deployed environment, but the
	// URL layer is one call: never dial a non-https or IP-literal-private row.
	if err := ValidateEndpoint(sub.Endpoint); err != nil {
		return err
	}
```

Drop the now-unused `time` import from `push.go` only if nothing else uses it (`PushTTL` uses `time.Hour` — it stays).

- [ ] **Step 5: Run — everything green.**

```bash
go vet ./internal/notify/... && go test ./internal/notify/... -count=1 -timeout 60s
```

If `TestPushHTTPClientRefusesANameResolvingToLoopback` reports an x509 error instead of `ErrForbiddenEndpoint`, the guard is not on the transport that was used — fix the wiring, never the assertion. If `errors.Is` does not see `ErrForbiddenEndpoint` through `*url.Error` → `*net.OpError`, both types implement `Unwrap` in Go 1.25; check that `guardDial` returns the wrapped sentinel and that `Send` wraps with `%w` (it does).

- [ ] **Step 6: Mutation checks (revert each).**
  1. Remove `Control: guardDial` from the dialer → `TestPushHTTPClientRefusesANameResolvingToLoopback` and the `dns name, dial layer` row of `TestWebPushSenderRefusesAForbiddenEndpointBeforeDialling` must fail (x509 error, and/or `hits != 0`).
  2. Remove `CheckRedirect` → both redirect tests must fail with `/redirected was hit 1 time(s)`.
  3. Remove the `ValidateEndpoint` call from `Send` → the two `URL layer` rows of `TestWebPushSenderRefusesAForbiddenEndpointBeforeDialling` must fail: the dial guard still refuses the loopback connect, but the error text is now `refusing to dial`, not `not a public address` / `want https` — the per-row `want` substring is what makes this mutation visible.

- [ ] **Step 7: Commit.**

```bash
git add internal/notify/endpoint.go internal/notify/endpoint_test.go internal/notify/push.go internal/notify/push_test.go
git commit -m "notify: push client refuses private destinations at the dial (resolved IP), never follows redirects; Send validates the endpoint first"
```

---

### Task 4: `Tick` prunes rows the guard refuses (pre-existing rows self-heal)

**Files:** `backend/internal/notify/service.go`, `service_test.go`, `fakes_test.go`.

- [ ] **Step 1: Failing test** — append to `service_test.go`:

```go
func TestTickPrunesAForbiddenEndpointAndReportsItOnce(t *testing.T) {
	// A row stored before validation existed (or via a DNS name that now
	// resolves privately) must be deleted on the first tick, not retried daily.
	h := dueHarness()
	h.sender.forbidden["https://push.example/ep1"] = true

	stats, err := h.svc.Tick(context.Background(), h.now)
	if !errors.Is(err, ErrForbiddenEndpoint) {
		t.Errorf("err = %v, want the forbidden endpoint reported so the worker logs it", err)
	}
	if stats.Pruned != 1 || stats.Sent != 1 || stats.Failed != 0 {
		t.Errorf("stats = %+v, want Pruned 1 Sent 1 Failed 0", stats)
	}
	if subs := h.repo.subs["u1"]; len(subs) != 1 || subs[0].ID != "s2" {
		t.Errorf("subscriptions after prune = %+v, want only s2", subs)
	}
	var deleted bool
	for _, c := range h.log.calls {
		if c == "repo.DeleteSubscription(s1)" {
			deleted = true
		}
	}
	if !deleted {
		t.Errorf("s1 was not deleted; calls = %v", h.log.calls)
	}
}
```

In `fakes_test.go`, add `forbidden map[string]bool // endpoint → answer ErrForbiddenEndpoint` to `fakeSender`, initialise it in `newFakeSender`, and in `Send` add before the `gone` check:

```go
	if f.forbidden[sub.Endpoint] {
		return fmt.Errorf("%w: refusing to dial 10.0.0.5", ErrForbiddenEndpoint)
	}
```

- [ ] **Step 2: Run — fails** (`Failed 1`, row still present).

```bash
go test ./internal/notify/... -run 'TickPrunesAForbidden' -timeout 60s
```

- [ ] **Step 3: Implement** — in `service.go` `Tick`, the send-loop switch becomes:

```go
			err := s.sender.Send(ctx, sub, s.payload)
			switch {
			case errors.Is(err, ErrSubscriptionGone):
				stats.Pruned++
				errs = appendIf(errs, s.repo.DeleteSubscription(ctx, sub.ID))
			case errors.Is(err, ErrForbiddenEndpoint):
				// Never deliverable and never should have been stored: drop it
				// like a 410, but say so once in the log.
				stats.Pruned++
				errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
				errs = appendIf(errs, s.repo.DeleteSubscription(ctx, sub.ID))
			case err != nil:
				stats.Failed++
				errs = append(errs, fmt.Errorf("user %s: %w", userID, err))
			default:
				stats.Sent++
			}
```

Update `Tick`'s doc comment: "…prune 404/410 **and forbidden-endpoint** ones…".

- [ ] **Step 4: Run — green.**

```bash
go vet ./internal/notify/... && go test ./internal/notify/... -count=1 -timeout 60s
```

- [ ] **Step 5: Mutation check.** Delete the new `case` → the test fails with `stats = {… Failed:1 …}` and `s1 was not deleted`. Revert.

- [ ] **Step 6: Commit.**

```bash
git add internal/notify/service.go internal/notify/service_test.go internal/notify/fakes_test.go
git commit -m "notify: Tick prunes subscriptions the endpoint guard refuses, reporting each once"
```

---

### Task 5: CODEMAP, full re-verification, push, CI

**Files:** `harness/CODEMAP.md` (repo root of the worktree), this plan's `## Execution summary`.

- [ ] **Step 1: CODEMAP** — extend the `notify` bullet in `harness/CODEMAP.md` (worktree copy) with one sentence, matching the neighbours' density:

> `push_subscription.endpoint` is a **server-dialled, client-supplied URL** (SSRF boundary): `ValidateEndpoint` (https, no userinfo, ≤ 2048 bytes, IP literals not private/loopback/link-local/CGNAT/ULA) runs at subscribe (→ 400 `invalid_request`, nothing written) and again in `Send`; the sender's only HTTP client (`newPushHTTPClient`) enforces the same deny list in `net.Dialer.Control` on the *resolved* address (DNS rebinding-safe), never follows redirects, and has `Proxy: nil` so the guard sees the true destination; a stored row the guard refuses fails with `ErrForbiddenEndpoint`, which `Tick` prunes like 404/410 and reports once. Hostnames are not resolved at subscribe time by design — `https://localhost/x` is stored and pruned at the first tick.

- [ ] **Step 2: Re-run the review's evidence** (the exact commands from `harness/reviews/2026-09-23-notify-…md` → *Verification re-run*), all from the worktree's `backend/`:

```bash
go build ./... && go vet ./...
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 180s
env -u DATABASE_URL -u REDIS_URL go test ./internal/notify/... -race -count=1 -timeout 180s
go test ./internal/notify/... -run 'ValidateEndpoint|ForbiddenAddr|GuardDial|PushHTTPClient|WebPushSender|HostileEndpoints|TickPrunes' -v -count=1 -timeout 60s
```

Expected: build/vet silent; every package `ok`; `-race` `ok`; the named tests all `PASS` (count them in the summary).

- [ ] **Step 3: Reproduce the reviewer's request against the real handler.** The review's reproduction was `POST /api/v1/settings/notifications` with `{"notification_time":"00:01","push_subscription":{"endpoint":"https://169.254.169.254/latest/meta-data/","p256dh":"BNc5T","auth":"aX8v"}}` → `200`. `TestSettingsHandlerRejectsHostileEndpointsWith400AndWritesNothing` carries that body verbatim through Gin; confirm it is in the `-v` output above as `PASS`. **Optional but preferred** if a dev stack is convenient: repeat the original slice's runtime proof (`COMPOSE_PROJECT_NAME=<slug>`, non-default ports, `make down` after) and `curl --max-time 5` the same body with a real JWT — expect `400 {"error":"invalid_request"}` and `SELECT count(*) FROM push_subscriptions` unchanged. Record whichever you did.

- [ ] **Step 4: Harness bookkeeping, then commit and push.** From the **repo root of the worktree**:

```bash
python3 tools/harness/cli.py validate
git add harness/CODEMAP.md
git commit -m "codemap: notify — endpoint validation at subscribe, dial-time private-address guard, no redirects"
git push origin harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue
gh run list --branch harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue --limit 3
```

Wait for `backend-unit`, `backend-integration`, `harness-tooling` to be green (`gh run watch <id>` is fine; bound with `--exit-status`). A red check is not done. The existing Draft PR for the branch updates itself; add a comment naming this plan and the blocker idea.

- [ ] **Step 5: Execution summary** — append `## Execution summary` to **this** plan file (in the main checkout, via the normal harness flow), listing: commits (expect 5), the verification output, the mutation checks performed and what each broke, any deviation, and the CI run id. Then the orchestrator/human runs `python3 tools/harness/cli.py set <this plan> status=done` and `blockers` must report nothing for slice 8.

---

## Verification

Run from the worktree's `backend/` unless noted. All must hold before this plan is `done`:

```bash
# 1. Build, vet, full unit suite without services, notify under -race (the review's exact commands)
go build ./... && go vet ./...
env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL go test ./... -count=1 -timeout 180s
env -u DATABASE_URL -u REDIS_URL go test ./internal/notify/... -race -count=1 -timeout 180s

# 2. The SSRF tests by name — every one PASS
go test ./internal/notify/... -run 'ValidateEndpoint|ForbiddenAddr|GuardDial|PushHTTPClient|WebPushSender|HostileEndpoints|TickPrunes' -v -count=1 -timeout 60s

# 3. Nothing outside notify (and CODEMAP) changed on the branch since 547c0ab
git diff --stat 547c0ab..HEAD -- . ':!internal/notify' ':!../harness/CODEMAP.md'    # expect empty

# 4. Five commits on top of 547c0ab, each with the trailer
git log --oneline 547c0ab..HEAD | wc -l                                              # 5
git log 547c0ab..HEAD --format=%B | grep -c 'Co-Authored-By'                        # 5

# 5. From the repo root: harness artefacts valid, branch CI green
python3 tools/harness/cli.py validate
gh run list --branch harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue --limit 1   # completed success
```

Load-bearing evidence the reviewer should re-check (each was mutation-tested during execution; the summary must list them):

| Behaviour | Test | Mutation that must break it |
| --- | --- | --- |
| Hostile URL refused at subscribe, nothing written | `TestUpdateSettingsRefusesHostileEndpointsBeforeWriting`, `TestSettingsHandlerRejectsHostileEndpointsWith400AndWritesNothing` | move `ValidateEndpoint` after `UpdatePreferences` |
| Every deny range, incl. CGNAT and IPv4-mapped | `TestForbiddenAddrBoundaries`, `TestValidateEndpointRefusesHostileURLs` | drop `100.64.0.0/10`; drop `Unmap()` |
| DNS name → private IP refused **at the dial** | `TestPushHTTPClientRefusesANameResolvingToLoopback`, `dns name, dial layer` row | remove `Control: guardDial` |
| Redirects not followed | `TestPushHTTPClientDoesNotFollowRedirects`, `TestWebPushSenderReturnsARedirectAsAFailureNotAFollow` | remove `CheckRedirect` |
| Real push endpoints still accepted | `TestValidateEndpointAcceptsRealPushServiceURLs`, `TestGuardDialAllowsPublicAddresses`, existing `TestWebPushSenderPostsAnEncryptedVAPIDSignedRequest` | make `forbiddenAddr` return `true` for `!ip.Is4()` |
| Stored forbidden row pruned on first tick | `TestTickPrunesAForbiddenEndpointAndReportsItOnce` | delete the `ErrForbiddenEndpoint` case |

---

## Notes and open questions

- **Why not resolve DNS at subscribe too?** It would give `https://evil.example` a 400 today, but it is network I/O in a request handler (latency, a second SSRF-ish primitive — DNS exfiltration — and flakiness), and it is exactly the check rebinding defeats. The dial guard is the honest layer; the 400 path covers the cases that are decidable without a network.
- **`Proxy: nil` is a real constraint.** If Railway ever requires an egress proxy for push, the guard must be re-thought (the proxy would do the connecting). Documented in CODEMAP so it is not undone casually.
- **Not folded in:** `no-per-user-subscription-cap-and-no-length-bound-on-endpoint` — `MaxEndpointLength` incidentally gives the length bound because a URL check without one is incomplete, but the per-user cap stays in the inbox; do not add it here.
- **Metadata services on other clouds** (`fd00:ec2::254`, `metadata.google.internal` → `169.254.169.254`) are covered by ULA and link-local respectively. NAT64 (`64:ff9b::/96`) and 6to4 are not special-cased; Railway is IPv4-egress and neither reaches a private v4 without a translator this backend does not run. Revisit only if egress changes.
