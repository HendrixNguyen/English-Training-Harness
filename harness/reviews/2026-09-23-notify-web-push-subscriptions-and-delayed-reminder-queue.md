---
plan: harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md, harness/ideas/_inbox/no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md, harness/ideas/_inbox/tick-sends-serially-with-a-10s-client-timeout-so-one-slow-pu.md, harness/ideas/_inbox/due-plus-re-slot-is-not-atomic-so-two-api-instances-double-s.md, harness/ideas/_inbox/ci-never-runs-go-test-race-so-background-goroutine-races-go-.md, harness/ideas/_inbox/nothing-tells-the-pwa-to-flatten-pushsubscription-tojson-so-.md, harness/ideas/_inbox/timezone-handling-depends-on-system-tzdata-with-no-time-tzda.md]
---
# Review — Notify: Web Push subscriptions and delayed reminder queue

**Plan:** `harness/plans/2026-09-23-notify-web-push-subscriptions-and-delayed-reminder-queue.md`
**Branch/worktree:** `harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue` / `.worktrees/notify-web-push-subscriptions-and-delayed-reminder-queue`
**Diff:** `git diff main...harness/2026-09-23-high-notify-web-push-subscriptions-and-delayed-reminder-queue --stat`

## Plan vs idea

Delivered. The idea's *Expected output* has five bullets; four land verbatim and the fifth was
deliberately re-contracted in the plan, not dropped.

- **`POST /api/v1/settings/notifications` behind `auth.Require()`** — present
  (`cmd/api/main.go:127`, `internal/notify/handler.go:30`), inserting into `push_subscriptions`
  deduplicated on `endpoint`, updating `users.notification_time`/`timezone`, and `ZADD`ing the
  next local occurrence. Subscription is optional so the same route moves only the time
  (`handler_test.go:60-69`).
- **The wire shape differs from the idea and matches the spec.** The idea asked for
  `{subscription:{endpoint, keys:{p256dh, auth}}}`; backend spec §6.4 (line 329) shows flat
  `push_subscription:{endpoint, p256dh, auth}`. Per AGENTS.md → *Reading the spec*, the backend
  spec wins for its own layer, the evaluator recorded the correction in the idea's
  `## Evaluation`, and the plan restates it under *Spec precedence*. Implementation follows the
  spec. **This is the correct call, not a deviation** — see *The flat-vs-nested contract* below
  for the one thing it does leave exposed.
- **Reminder worker on the in-process cron, 30 s ticker** — `RunWorker` (`worker.go:15`),
  `Service.Tick` (`service.go:100`): `ZRANGEBYSCORE -inf now LIMIT 100`, skip target-met, send to
  every subscription, delete on 404/410, re-slot for tomorrow. Payload `{title, body, url:"/"}`
  (`push.go:20-31`).
- **Skip check moved from `daily_progress.is_target_met` to the §4 `daily:accumulated` counter**
  through a `StudyCounter` interface. Recorded in the idea's `## Evaluation` and the plan's *Spec
  precedence*. Right call: it is the same source of truth quests and pet use, and it keeps notify
  out of quests' table. `grep -rn 'daily:accumulated|DailyAccumulatedKey|daily_progress'
  internal/notify/` finds two hits, both doc comments (`service.go:13`, `schedule.go:62`) — no
  code reaches across the boundary.
- **`VAPID_PUBLIC_KEY` exposed via Nuxt runtime config, no new endpoint** — correctly not built
  here; §7 has no such route and the plan says so.
- **Tests the idea asked for** all exist: DST + non-UTC next-fire (`schedule_test.go:62-89`),
  worker pops only due members and reschedules (`service_test.go:99-145`), target-met skip
  (`:147`), 410 prune (`:178`), duplicate endpoint not inserted twice
  (`integration_test.go`, live).

## Code vs plan

All 8 tasks followed. Three deviations were declared in the *Execution summary*; **I verified all
three independently and all three are genuine.**

| Task | Verdict |
| --- | --- |
| 1 VAPID config (optional at boot) | followed |
| 2 Pure scheduling | followed, with one corrected plan bug (below) |
| 3 ZSET queue + Pg repo + integration test | followed |
| 4 Web Push sender over `webpush-go` | followed, with one corrected plan bug (below) |
| 5 Fakes + `Service.UpdateSettings`/`Tick` | followed |
| 6 Worker loop + §6.4 route | followed |
| 7 `main.go` wiring | followed |
| 8 CODEMAP | followed; the paragraph is accurate and I made no correction |

**Deviation 1 — the plan's DST test asserted 23h; the executor changed it to 22h.** Confirmed
correct, and the fix was to the *test*, not the code. 2026-03-07 21:00 EST is UTC-5 → 2026-03-08
02:00Z; 2026-03-08 20:00 EDT is UTC-4 → 2026-03-09 00:00Z; the interval is 22h. The plan's 23h
was arithmetically wrong. `schedule_test.go:77-89` carries a `PLAN DEVIATION` comment explaining
it. The implementation (`time.Date` in `loc`, letting Go resolve the offset) was already right.

**Deviation 2 — `.env.example`'s VAPID generator command.** Confirmed: the v1.4.0 module
contains only `example/` and `.github/` (`find $(go env GOMODCACHE)/github.com/!sher!clock!holmes/webpush-go@v1.4.0 -maxdepth 1 -type d`),
so `go run .../cmd/webpush-go@v1.4.0` does not exist. The plan's own Task 1 contingency note
authorised exactly this substitution.

**Deviation 3 — mutex on `fakeSender.sent`.** Also authorised by the plan's contingency note.
See *Is the race a fake-only artefact?* below.

**Deviation 4 — the `daily:accumulated` grep finds 2 hits, not 0.** Confirmed: both are doc
comments that the plan itself supplied verbatim. The check's intent (no cross-package access) is
satisfied.

**Deviation 5 — the `.env.example` fix bundled into the Task 4 commit.** Confirmed, harmless.

### Verification re-run (in the worktree, from `backend/`)

```
$ go build ./... && go vet ./...
BUILD+VET OK                                   (no output from either)

$ env -u DATABASE_URL -u REDIS_URL -u TEST_DATABASE_URL -u TEST_REDIS_URL \
      go test ./... -count=1 -timeout 180s
ok  .../internal/airouter  0.751s      ok  .../internal/pet     2.737s
ok  .../internal/auth      1.132s      ok  .../internal/quests  4.539s
ok  .../internal/config    1.570s      ok  .../internal/store   3.906s
ok  .../internal/health    2.155s      ok  .../internal/notify  3.410s

$ env -u DATABASE_URL -u REDIS_URL go test ./internal/notify/... -race -count=1 -timeout 180s
ok  .../internal/notify  1.957s

$ python3 tools/harness/cli.py validate     -> exit 0
$ git status --short (worktree)             -> clean
$ gh run list --branch harness/2026-09-23-high-notify-...
completed  success  CI  ...  35815345858  1m44s     (backend-unit, backend-integration, harness-tooling)
```

No executor gate failure: everything the summary claims reproduces. I did not re-run the live
timing proof as my main activity, per the review brief.

## Quality

### Test honesty — mutation-proved

I broke each central behaviour transiently and confirmed the tests object, then reverted
(`git status --short` clean afterwards).

| Mutation | Result |
| --- | --- |
| `NextSendTime` builds the candidate in `time.UTC` instead of `loc` | **6 tests fail**, incl. `service_test.go:26` `next_reminder_at`, `:29` `ZSET score = 1790107200, want 1790082000 (tonight 20:00 HCM)`, `:109` re-slot score, and both DST assertions |
| `!next.After(now)` → `next.Before(now)` (equality no longer rolls to tomorrow) | fails `schedule_test.go:59` "at the exact minute: got 2026-09-22 20:00…, want tomorrow" |
| `total >= TargetSeconds` → `total > TargetSeconds` | fails `service_test.go:156` — the 1800-exactly boundary is pinned |
| Move the `queue.Schedule` re-slot to *after* the send loop | fails `service_test.go:120` "must re-slot before sending (crash safety)", printing the reordered call log |

The two claims the slice rests on — **correct timezone-aware score** and **re-slot-before-send,
so no double fire** — are both load-bearing. This is not a Google-review-style hollow assertion.
The call-log fake (`fakes_test.go`) is what makes the ordering assertion possible and is good
design.

### Is the race the executor found a fake-only artefact?

Yes, and I checked rather than assuming. `Service`'s six fields are written once in `NewService`
and only read in `Tick`; `RunWorker` (`worker.go:15-33`) is the single goroutine calling `Tick`;
`RedisQueue`/`PgRepo`/`WebPushSender` hold only goroutine-safe clients. The race was between the
*test* goroutine polling `h.sender.sent` and the worker goroutine appending to it — a test
artefact, correctly fixed with a mutex and a `Sent()` accessor. `go test ./internal/notify/... -race`
is green. **But CI never looks** (`.github/workflows/ci.yml:44` is `go test ./... -count=1`), which
is now a gap worth closing given two background goroutines ship in the binary — filed separately.

### Crash safety — which trade-off was chosen, and was it deliberate?

Deliberate and stated in three places (plan *Architecture*, `service.go:95-99`, `worker.go:13-14`):
**re-slot first, then send**. A crash between the `ZADD` and the push costs the user *one*
reminder; it can never produce a re-fire loop. The alternative (send, then re-slot) would risk
re-firing the same reminder every 30 s until the process recovered. The safer choice was made,
and it is the right one for a notification: a missed nudge is invisible, a repeating one is a
reason to uninstall. This is a well-made decision, not an accident.

### Unbounded growth of the persistent ZSET — checked, and it is bounded

`queue:webpush:delay` has no TTL, so every removal path matters. I traced all of them:

- user row deleted → `Preferences` returns `ErrUserNotFound` → `queue.Remove` (`service.go:110-114`);
- last subscription gone → `len(subs) == 0` → `queue.Remove` (`service.go:148-151`);
- endpoint 410s → the *subscription row* is deleted (`service.go:155-157`) but the user stays
  queued with tomorrow's score. **This is correct and self-healing, not a leak**: tomorrow's tick
  finds zero subscriptions and `ZREM`s the member. The member survives at most one extra cycle,
  costing one Postgres lookup. I confirmed the same holds for a time-only `UpdateSettings` that
  queues a user with no subscription — the plan documents this as "one wasted lookup per day".

There is no account-deletion or unsubscribe endpoint in §7, so `ErrUserNotFound` is the only
account-removal path and it is handled. **I found no unbounded-growth defect in the queue.** The
unbounded growth that *does* exist is in `push_subscriptions`, not Redis — filed separately.

### VAPID key handling

Clean. The private key is never logged, never formatted into an error, and never committed:

- `NewWebPushSender`'s only error string is the generic "both are required" (`push.go:56`);
- `main.go:101` logs only `notify: %v` of that error, and `main.go:116` logs the *absence* of
  keys by env-var name, never a value;
- `Tick`'s error joins carry user ids and HTTP status codes, never key material;
- `.env.example:31-33` ships commented, empty placeholders; the repo `.gitignore:5-6` excludes
  `*.env`/`.env*`; tests generate throwaway pairs with `webpush.GenerateVAPIDKeys()`
  (`push_test.go`), so no fixture key exists.

**Missing-key degradation is clean, and I verified the path rather than trusting the claim.**
`config.Load` treats both keys as optional (`config.go:64-69`); `main.go:99-119` constructs the
sender only when both are present, and starts the worker only when the sender is non-nil, logging
which. `Tick` is therefore unreachable with a nil sender, so there is no nil-panic in practice.
*Observation, not filed:* `NewService` still accepts a nil `Sender` guarded only by a doc comment
(`service.go:50-52`) and `service.go:153` would panic if a future caller (say, an admin
"send test push" route) ever called `Tick` without one. A `Tick` early-return on `s.sender == nil`
would make the invariant self-enforcing. Too small to spend an evaluator cycle on today.

### The flat-vs-nested contract — my judgement

**Deliberate, correct, and documented — but under-communicated to the frontend.**

The review brief framed this as "flat implementation vs. nested spec §6.4". That framing is
backwards, and it matters, so to be precise: I read the spec line. Backend spec §6.4, line 329,
is **flat**:

```json
{"notification_time": "20:00:00", "push_subscription": {"endpoint": "push_subscription_endpoint_string", "p256dh": "BNc5T...", "auth": "aX8v..."}}
```

`handler_test.go:37` is that line copied character for character. It is the **idea file** that
proposed the nested `subscription.keys.*` shape, and the evaluator overruled it in favour of the
spec under AGENTS.md's precedence rule, recording the correction in the idea's `## Evaluation` and
again in the plan's *Spec precedence*. So this is not a spec discrepancy at all — it is an
idea-vs-spec discrepancy resolved the right way, and the executor implemented the spec. Both
items in the plan's *Notes and open questions* are recorded honestly; I checked the second one
too — §9 (line 372) lists `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` and genuinely has no
`VAPID_SUBJECT`, exactly as claimed, and the default is flagged in `config.go:29-31` and
`.env.example`.

The real risk the brief was pointing at survives all of that, and I agree it is live: the
browser's `PushSubscription.toJSON()` emits the **nested** shape, so the obvious frontend line
— `push_subscription: subscription.toJSON()` — is a 400 every time. The frontend technical
specification has no entry for this endpoint at all (grepping it for `pushManager|subscribe|
push_subscription|notification_time` matches nothing; `p256dh` appears only in its DDL and ER
diagram). MVP slice 9 is being built against these contracts right now, so I filed it. Mitigating
it from a blocker: the backend rejects the nested body **loudly** with 400 and has a test pinning
that (`handler_test.go:77`), so this fails in the first five minutes of frontend development
rather than silently writing NULLs — materially better than the `token`/`access_token` class of
bug. Hence medium, not high.

### Boundaries, conventions, CODEMAP

- **Boundaries hold.** notify reaches quests only through `StudyCounter` (`service.go:16-18`),
  registered in `main.go:114`. No cross-package table access; no import of `quests` in the
  package. `Location`/`LocalDate`/`TargetSeconds` are deliberately duplicated rather than
  imported, with the "third copy → hoist" rule written down.
- **Conventions match.** `RunWorker` mirrors `pet.RunHourly`; the four-interface + fakes design
  matches pet and quests; error sentinels, `fmt.Errorf("notify: …: %w", err)` wrapping, `var _ I =
  (*T)(nil)` assertions, and the `TEST_DATABASE_URL`/`TEST_REDIS_URL` integration gate all follow
  the existing house style.
- **`Tick`'s error handling is idiomatic and careful**: per-user failures accumulate into
  `errors.Join` and the pass never stops early (`service.go:108`, `:166`); a counter read error
  counts as "not met" so a Redis outage can never silence reminders (`service.go:133-137`) — and
  that is itself tested (`service_test.go:166`).
- **CODEMAP** (`harness/CODEMAP.md`) is accurate, specific and matches the code I read. I made no
  correction.

### Shutdown — does this slice make the known `main.go` gap worse?

No, and this is the one place I want to be explicit because the executor's cleanup note could be
misread. `RunWorker` **does** respect the context: `select { case <-ctx.Done(): return … }`
(`worker.go:18-23`), `defer ticker.Stop()` (`worker.go:17`), and
`TestRunWorkerTicksAndStopsWhenTheContextIsCancelled` asserts the goroutine returns within a
second of `cancel()`. It is wired to `main.go`'s `signal.NotifyContext` at `main.go:118`. So on
SIGTERM the worker's context *is* cancelled, the ticker stops, and an in-flight push is aborted
by the same context — no goroutine leak, no orphaned ticker. The reason the executor's test
server survived SIGTERM is entirely the pre-existing gap that the blocking `r.Run()` never
consults that context and the process never reaches the point of exiting, already filed as
`main-go-installs-a-signal-handler-with-no-server-shutdown-so.md`. **Not re-filed.** If anything
this slice mildly *improves* the case for fixing that bug: there is now a second background
goroutine whose clean stop is wasted because the process cannot exit gracefully.

### Untested boundaries — observed, not filed

`schedule_test.go` pins the equality boundary well (mutation-proved above), but leaves month/year
rollover (`l.Day()+1` on 31 December) and a `notification_time` inside a spring-forward gap hour
unpinned. Both are correct today because `time.Date` normalises, and the identical gap in the
sibling `internal/google/schedule.go` is already filed
(`schedule-go-boundaries-are-untested-at-notification-time-equ.md`), so filing a near-duplicate
would only cost the evaluator a cycle. Noting it here so it is on the record.

## Bugs filed

1. **BLOCKER, high** — `harness/ideas/_inbox/push-subscription-endpoint-is-an-unvalidated-user-supplied-u.md`
   SSRF: `push_subscription.endpoint` is an arbitrary client-supplied URL that the worker POSTs
   to daily, forever. No scheme, host, or address validation anywhere between
   `handler.go:22-26` and `push.go:72-81`. An authenticated user can aim the backend's HTTP
   client at `https://169.254.169.254/…`, a `*.railway.internal` service, or `https://127.0.0.1/…`
   on a schedule they choose. `blocks` this plan.
2. **medium** — `no-per-user-subscription-cap-and-no-length-bound-on-endpoint.md`
   No cap on rows per user, no length limit on three `TEXT` columns, no validation that `p256dh`
   decodes to a 65-byte P-256 point or `auth` to 16 bytes. Each stored row is one outbound
   request per tick, and a malformed key produces a non-gone error that is retried daily forever
   because only 404/410 prunes.
3. **medium** — `tick-sends-serially-with-a-10s-client-timeout-so-one-slow-pu.md`
   `Tick` is fully serial (`service.go:109-165`) with a 10 s per-request timeout (`push.go:62`).
   Worst case for one pass is 100 users x subscriptions x 10 s against a 30 s ticker; one slow
   push service delays every other learner's reminder, and no deadline bounds a pass.
4. **medium** — `due-plus-re-slot-is-not-atomic-so-two-api-instances-double-s.md`
   `Due` is a plain `ZRANGEBYSCORE` read (`queue.go:43`) and the re-slot is a separate `ZADD`
   (`service.go:127`) with a Postgres round trip in between, so two processes both claim the same
   member. Documented honestly in the plan's *Notes* and `worker.go:13-14`, but nothing enforces
   "one worker": a Railway rolling deploy overlaps old and new containers, so this fires in
   practice, not only under scale-out.
5. **medium** — `ci-never-runs-go-test-race-so-background-goroutine-races-go-.md`
   `.github/workflows/ci.yml:44` has no `-race`, while the binary now runs two background
   goroutines. The executor's locally-caught race would have shipped green.
6. **medium** — `nothing-tells-the-pwa-to-flatten-pushsubscription-tojson-so-.md`
   The frontend spec has no entry for this endpoint, so slice 9 will naturally send
   `subscription.toJSON()` (nested) and get 400. Contract-visibility, not correctness.
7. **low** — `timezone-handling-depends-on-system-tzdata-with-no-time-tzda.md`
   No `_ "time/tzdata"` import anywhere. On an image without zoneinfo, `UpdateSettings` rejects
   every valid IANA zone with 400 *and* `Location()` silently falls back to UTC, firing reminders
   at the wrong hour. Conditional on a deployment image that does not exist in the repo yet, hence
   low.

## Verdict

**`pass-with-bugs`, with one blocker holding the merge.**

This is high-quality work and the best-documented slice I have reviewed on this project. The idea
is delivered, the plan is followed task for task, the three declared deviations are all genuine
and all three corrected something real (including an arithmetic error in the plan's own test).
The tests are honest — I mutation-proved the two assertions the slice rests on and both failed
loudly. The crash-safety trade-off was chosen deliberately and correctly, the persistent ZSET has
no growth leak, the VAPID private key never leaks into a log, an error or a fixture, the missing-key
boot path degrades cleanly, the worker shuts down properly on context cancellation, the package
boundary to quests is respected, and the CODEMAP paragraph is accurate enough that I changed
nothing in it. CI is green on the pushed branch.

What stops the merge is input validation, not design. `push_subscription.endpoint` is an
attacker-controlled URL that this service will dial, on a schedule, with no check on scheme, host
or resolved address. That is a security hole in code about to land, the slice owns that input, and
the fix belongs on this branch — so it is filed as a blocker with `blocks` set, and
`cli.py blockers --plan <plan>` exits 1 until an amending plan lands. Everything else is ordinary
inbox work.

No PR exists (`gh pr create` 403s — the authenticated `gh` account is the owner's work account and
is not a collaborator on this repository), so there is no PR to comment on or mark ready. Expected
and not counted as a finding.
