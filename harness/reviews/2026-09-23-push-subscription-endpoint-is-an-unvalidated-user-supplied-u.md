---
plan: harness/plans/2026-09-23-push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md
verdict: pass
bugs: []
---
# Review — Notify amend: validate `push_subscription.endpoint` at subscribe and refuse private destinations at the dial (SSRF)

**Plan:** `harness/plans/2026-09-23-push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md`
**Branch/worktree:** `harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue` / `.worktrees/notify-web-push-subscriptions-and-delayed-reminder-queue`
**Diff:** `git diff main...harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue --stat`

## Plan vs idea

The idea is the notify review's blocker 1: `push_subscription.endpoint` was an authenticated-user-supplied string handed to `http.Client.Do` once per user per day at a user-chosen time, reachable at `https://169.254.169.254/latest/meta-data/`. Expected output was a 400 at subscribe with nothing written, and a refusal at send.

Both are delivered, and the reviewer's exact reproduction body is carried verbatim through Gin by `TestSettingsHandlerRejectsHostileEndpointsWith400AndWritesNothing` (`backend/internal/notify/handler_test.go:102`). The §6.4 wire shape is unchanged apart from the `spec64Body` fixture, whose endpoint had to stop being the spec's non-URL placeholder `push_subscription_endpoint_string` — a necessary and correctly-documented consequence.

## Code vs plan

All five tasks followed, no deviations. Fix commits touch only `backend/internal/notify/*` plus one `harness/CODEMAP.md` sentence; `git diff 547c0ab..b85189a --name-only` lists exactly nine files. All five commits carry the `Co-Authored-By` trailer.

### Verification re-run (worktree `backend/`, my own shell)

```
go build ./... && go vet ./...                       # silent
go test -timeout 120s ./...                          # ok: airouter auth config health notify pet quests store
go test -race -timeout 180s ./internal/notify/...    # ok  (1.784s, no races — CI does not run this)
go test ./internal/notify/... -run 'ValidateEndpoint|ForbiddenAddr|GuardDial|PushHTTPClient|WebPushSender|HostileEndpoints|TickPrunes' -v
                                                     # 17 --- PASS, 0 FAIL  (matches the summary's "17/17")
gh run view 35819788843 --json headSha,conclusion     # headSha b85189a3953dba…, conclusion success
                                                     # jobs: harness-tooling / backend-integration / backend-unit all success
```

CI was read for head SHA `b85189a`, not the branch's earlier `547c0ab` run (`35815345858`). Everything in the execution summary reproduced; no executor gate failure.

### Attempts to defeat the guard

I did not take the suite as evidence. Every line below is from a program I wrote and ran (a `net.Dialer` with the same `Control` predicate, dialling a live loopback listener that records accepts) or from the code path itself.

1. **DNS rebinding — CLOSED.** `Control` is called per connect attempt with the resolved `ip:port`, so there is no check-then-connect window. My dialer probe on `localhost` shows the hook fired twice (`[::1]:p` then `127.0.0.1:p` — happy-eyeballs) and both were refused; the listener accepted nothing. There is no unguarded second door: `newPushHTTPClient` is the only `http.Client` in the package, `NewWebPushSender` (`push.go:63`) is its only non-test construction site, `cmd/api/main.go:100` is the only caller, and webpush-go v1.4.0 `webpush.go:236-243` uses `options.HTTPClient` when non-nil — its `&http.Client{}` fallback is unreachable here. `grep` for `http.DefaultClient` in non-test code finds only `internal/auth/google.go:68`, a server-controlled Google OAuth URL, out of scope.

2. **Address encodings — CLOSED, but at the dial, not the URL layer.** `netip.ParseAddr` rejects `2130706433`, `0177.0.0.1`, `0x7f000001`, `127.1` and `169.254.169.254.`, so `ValidateEndpoint` lets all of them through. That is harmless because the resolver normalises them *before* `Control` runs — my probe shows the hook receiving `127.0.0.1:port` for the decimal, hex and short forms (all refused, zero accepts), and the trailing-dot form failing DNS outright with no dial at all. IPv4-mapped IPv6 is handled correctly in both spellings: `[::ffff:127.0.0.1]` and `[::ffff:7f00:1]` both `Unmap()` to `127.0.0.1` and are refused at the URL layer. `0.0.0.0` and `[::]` are caught by `IsUnspecified`.

3. **Redirects — CLOSED, and a 3xx is not a success.** `CheckRedirect` → `ErrUseLastResponse` means no second request is issued, so a redirect to a private host never opens a connection; and even if it did, that connect would pass through `guardDial` too. `push.go:96` classifies `< 200 || > 299` as an error, so the 302 is a failure, not a delivery. Mutation 6 below proves both halves.

4. **Ranges — every boundary correct on both sides.** I swept 56 addresses one below / at / at-top / one above each range: 10/8, 172.16/12, 192.168/16, 127/8, 169.254/16, 100.64/10, 0/8, 192.0.0/24, 198.18/15, 240/4, 224/4 multicast, `::`, `::1`, fc00::/7 (`fbff:…` out, `fc00::` in, `fdff:…` in, `fe00::` out), fe80::/10 (`fe7f:…` out, `fe80::` in, `febf:…` in, `fec0::` out), ff00::/8. Zero mismatches. Legitimate endpoints still pass: FCM, Mozilla autopush, Apple, WNS, a self-hosted UnifiedPush host and an explicit `:443` are all accepted, and I confirmed `8.8.8.8`, `1.1.1.1`, `142.250.31.188`, `2001:4860:4860::8888` and `::ffff:8.8.8.8` are permitted. Railway's own private networking (ULA `fd00::/8`) is covered by `IsPrivate`; `metadata.google.internal` resolves into link-local and dies at the guard.

5. **The two doors — both confirmed.** `UpdateSettings` (`service.go:69-73`) calls `ValidateEndpoint` after the presence check and before `repo.UpdatePreferences` — the first repo call in the function. The handler test asserts `len(h.log.calls) == 0`, i.e. literally zero rows written, not merely no subscription row. `Send` (`push.go:71`) validates before `webpush.SendNotificationWithContext`. A row that bypassed the handler is refused at send and pruned: `Tick`'s switch gains `case errors.Is(err, ErrForbiddenEndpoint)` before the generic `err != nil`, incrementing `Pruned` and calling `DeleteSubscription`. I verified the `errors.Is` chain survives the real wrapping (`notify: sending push:` → `url.Error` → `net.OpError` → `guardDial`) — mutation 5's output shows it matching through all four layers.

6. **Scheme / userinfo / length — verified.** https-only, `u.User != nil` rejected, `len(raw) > 2048` checked before parsing (exact: 2048 passes, 2049 rejected). CRLF and NUL header-injection attempts (`https://host/x\r\nHost: evil`) are rejected by `url.Parse` as invalid control characters, before the scheme check.

### Mutation results (all reverted; `git status --porcelain` clean before and after)

| Mutation | Result |
|---|---|
| **Remove `Control: guardDial`** | `TestPushHTTPClientRefusesANameResolvingToLoopback` and the `dns name, dial layer` row of `TestWebPushSenderRefusesAForbiddenEndpointBeforeDialling` FAIL with `tls: failed to verify certificate: x509: certificate signed by unknown authority` — i.e. the TCP connection to loopback actually opened and only the test CA stopped it. This is the load-bearing assertion, and it holds. |
| Comment out `100.64.0.0/10` (plan Task 1) | 4 tests FAIL across all three layers: `TestForbiddenAddrBoundaries`, 2 CGNAT rows of `TestValidateEndpointRefusesHostileURLs`, `TestGuardDialRefusesPrivateAddresses`, and both door tests (`cgnat: status = 200` with `repo.SaveSubscription(u1,https://100.64.0.1/x)` in the call log). |
| Move `ValidateEndpoint` after `repo.UpdatePreferences` (plan Task 2) | `TestSettingsHandlerRejectsHostileEndpointsWith400AndWritesNothing` FAILs on all 7 rows with `400 but the service still called [repo.UpdatePreferences(u1,00:01:00,)]`. The zero-rows property is genuinely asserted, not implied. |
| Delete `Tick`'s `ErrForbiddenEndpoint` case (plan Task 4) | `TestTickPrunesAForbiddenEndpointAndReportsItOnce` FAILs: `{Pruned:0 Failed:1}` and `s1 was not deleted`. |
| Delete `Send`'s `ValidateEndpoint` call (mine) | The two URL-layer rows FAIL — *because the dial guard still caught them* and the error text changed from `not a public address` / `want https` to `refusing to dial`. The per-row `want` substring is what makes this visible. Good test design, and it proves the two doors are independent rather than one door tested twice. |
| Drop `CheckRedirect` (mine) | Both redirect tests FAIL: `/redirected was hit 1 time(s)`, and `TestWebPushSenderReturnsARedirectAsAFailureNotAFollow` gets `err = <nil>` — i.e. without it a redirect would be reported as a *successful send*. |

Six mutations, six kills, no survivors. Nothing here passes with its subject removed.

## Quality

**Test honesty.** `hostileEndpoints` is a shared table driven from three layers (`ValidateEndpoint`, `UpdateSettings`, the Gin handler), so a range added to the predicate is automatically exercised end to end. The guard tests assert a handler hit-counter is `0`, not merely that an error came back — an assertion that distinguishes "refused" from "connected then failed". The two tests that swap the transport (`TestPushHTTPClientDoesNotFollowRedirects`, `TestWebPushSenderReturnsARedirectAsAFailureNotAFollow`) say in-comment that they are testing `CheckRedirect` only and that the guard is covered elsewhere — accurate, and mutation 6 shows they still bite.

**The `localhost` TLS override did not weaken anything.** `InsecureSkipVerify` appears nowhere in the repo, test or otherwise. `push_test.go:62` clones the httptest transport and sets `TLSClientConfig.ServerName = "example.com"` — still full chain and hostname verification, just against a name the httptest cert actually carries. It is confined to `testTLSClient` and one inline block in `push_test.go`; the production client built by `newPushHTTPClient` sets no `TLSClientConfig` at all.

**The documented narrowing is what the code does.** No DNS lookup at subscribe, so `https://localhost/x` is stored and dies at the first tick rather than being 400'd. `TestValidateEndpointAcceptsRealPushServiceURLs:81` asserts exactly that with the reason in-comment, `endpoint.go:30` states it, and `CODEMAP.md` ends the notify bullet with it verbatim. Honest, and the right trade — resolving at subscribe would be a TOCTOU check that the dial guard has to redo anyway.

**Scope held.** None of the notify review's other six findings were folded in: no per-user subscription cap (`service.go` has no cap or `LIMIT`), sends are still serial in a plain `for` loop, due+re-slot is still non-atomic, `.github/workflows/ci.yml` still has no `-race` (the only `race` hit is a comment about parallel test binaries), no `time/tzdata` import, no `toJSON` anywhere. `MaxEndpointLength` does incidentally satisfy the length half of one inbox bug, which the plan flagged in advance.

**CODEMAP** is accurate — I checked each clause against the code, including `Proxy: nil`, the prune-like-410 behaviour and the localhost narrowing. No correction needed.

### Residual risk — reported, deliberately not filed as bugs

I could not turn any of these into a reachable internal address, so filing them would inflate the count. Recording them so a human can overrule.

- **IPv4-compatible IPv6 (`::127.0.0.1`, `::a9fe:a9fe`), NAT64 `64:ff9b::/96`, 6to4 `2002::/16`, site-local `fec0::/10`, IPv4-translated `::ffff:0:0/96`** all pass `forbiddenAddr` — `Unmap()` only handles `::ffff:0:0/96`. I dialled every one of them at a live loopback listener: all six returned `connect: no route to host`, listener hits 0. IPv4-compatible and site-local are deprecated and unrouted; NAT64 and 6to4 need a translator. The plan's *Notes* explicitly considered NAT64 and 6to4 and deferred them on "Railway is IPv4-egress … revisit only if egress changes" — a recorded decision, not an oversight. Worth one line in `forbiddenPrefixes` if egress ever becomes IPv6/NAT64 (AWS IPv6-only subnets do run a `64:ff9b::/96` translator to `169.254.169.254`). Tested on Darwin; I did not verify Linux routing.
- **Port is unbounded.** `https://8.8.8.8:22/` passes both layers, so a user can make the server open one TCP connection per day to any port of any *public* host. Not internal SSRF and not what the blocker was about; a weak outbound-connect primitive at most.
- **`WebPushSender.HTTPClient` is an exported field.** Today only `NewWebPushSender` builds the struct, so it is always the guarded client. A future `&WebPushSender{…}` literal that forgets it would silently get webpush-go's unguarded `&http.Client{}` fallback, leaving only `Send`'s `ValidateEndpoint` (which does not stop DNS rebinding). Latent maintainability risk, not a current hole.

## Bugs filed

None.

## Verdict

**pass.** The hole is closed at both doors and, decisively, at the dial on the resolved address — the one layer that survives DNS rebinding. I tried the encoding, rebinding and redirect bypasses an attacker would actually reach for and could not get a connection to a private destination; the only predicate gaps I found are unroutable, and one of them was already a recorded decision. The suite is not merely green: six mutations including removal of the guard itself each killed the tests that claim to cover them.

**The notify branch is safe to merge** (`/harness merge harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md`) once its own six non-blocking inbox findings are accepted as deferred. No new blocker filed, so `cli.py blockers --plan <notify plan>` correctly continues to exit 0.

No PR exists for this branch — `gh pr create` 403s because the `gh` CLI is authenticated as the owner's work account — so there is nothing to comment on or mark ready. Noted, not a finding.
