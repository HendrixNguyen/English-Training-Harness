---
plan: harness/plans/2026-09-24-every-google-403-becomes-409-reauth-required-so-a-quota-erro.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/google-403-accessnotconfigured-api-disabled-still-maps-to-re.md, harness/ideas/_inbox/no-test-pins-a-409-from-calendar-events-patch-to-502-google-.md]
---
# Review — Google sync: quota 403s stop forcing re-consent, unconsumed 409s stop surfacing as 500, and the route logs what Google said

**Plan:** `harness/plans/2026-09-24-every-google-403-becomes-409-reauth-required-so-a-quota-erro.md`
**Branch/worktree:** `harness/2026-09-24-medium-every-google-403-becomes-409-reauth-required-so-a-quota-erro` / `.worktrees/every-google-403-becomes-409-reauth-required-so-a-quota-erro`
**Diff:** `git diff main...harness/2026-09-24-medium-every-google-403-becomes-409-reauth-required-so-a-quota-erro --stat`

## Plan vs idea
Three inbox ideas folded into one plan; all three *Expected output*s are substantially delivered.

- **403 quota vs reauth (head idea):** delivered for the four quota reasons and 429, with Calendar and Tasks tests and a CODEMAP rewrite. **Deviation not called out in the plan:** the idea asked for an allow-list ("only a genuine 401, or a 403 whose reason is `insufficientPermissions`/`forbidden`, yields `ErrReauthRequired`"); the plan built a deny-list (any 403 not in `throttleReasons` → reauth). That is the safer default for an empty or non-JSON 403, but it leaves config-type 403s (`accessNotConfigured`, API disabled) on the reauth path, where re-consent cannot help → low bug filed.
- **Unconsumed 409 → 502:** delivered via the idea's second option (the handler maps `ErrAlreadyExists` to 502). The idea's required test "pins a 409 from `events.patch` and a 409 from `tasks`" is only half there: tasks is covered through a fake, events.patch is not → low bug filed.
- **Server-side logging:** delivered. One line per failure (service/status/body truncated to 512 B for `*UpstreamError`, `%v` otherwise), one success line with user/event/task count, and a test that the fake refresh token (`1//refresh`) and access token (`access-for-…`) never appear. I checked the fakes: those are the values the harness actually hands out (`fakes_test.go:50,234`), so the leak test asserts something real.

## Code vs plan
Worktree `.worktrees/every-google-403-becomes-409-reauth-required-so-a-quota-erro` did not exist; I created it detached at `origin/<branch>` = `a378230` (the local branch ref is checked out elsewhere). Diff vs `origin/main`: 6 files, +198/−20 (`client.go`, `handler.go`, three test files, CODEMAP). No app code outside `internal/google`.

| Task | Result |
| --- | --- |
| 1 — throttle 403 → `*UpstreamError` | Followed verbatim (`throttleReasons`, `googleErrorReason`, split 401/403 cases, extended `TestCalendarMapsStatusesToSentinelErrors`, new `TestTasksMapsAQuota403ToUpstreamNotReauth`). |
| 2 — doc comments | Followed. |
| 3 — unconsumed 409 → 502 | Followed. |
| 4 — logging | Followed; the executor's one deviation (no `strings` import in `handler.go`, since `truncate` does not use it) is correct — the plan text would not have compiled. |
| 5 — CODEMAP | Followed; the new sentence matches the code. |

Re-run in the worktree (`backend/`):

```
$ gofmt -l internal/google
(no output)
$ gofmt -l .
internal/quests/handler_test.go
internal/quests/repo.go            # not touched by this branch; clean on current origin/main
$ go vet ./... && go test -timeout 120s ./internal/google -count=1 -v | grep ...
--- PASS: TestCalendarMapsStatusesToSentinelErrors
--- PASS: TestSyncHandlerMapsAnUnconsumed409To502
--- PASS: TestSyncHandlerLogsTheFailureServerSideOnly
--- PASS: TestSyncHandlerLogsA500WithTheCauseAndTruncatesLongBodies
--- PASS: TestSyncHandlerLogsSuccessWithoutTokens
--- PASS: TestTasksMapsAQuota403ToUpstreamNotReauth
ok   .../internal/google 1.049s
$ go test -timeout 300s -count=1 ./...
ok for airouter, auth, config, google, health, notify, onboarding, pet, quests, store
```

**CI:** `gh run list --branch <branch>` → run 36026951264 on head `a378230`: `backend-unit`, `backend-integration`, `frontend`, `harness-tooling` all success.

**Runtime proof:** not re-run with Docker. The executor's proof boots the binary and hits the route unauthenticated (401), which exercises none of the changed branches; the changed behaviour is reachable only with a live Google account, and it is fully covered by the `httptest`/fake tests I re-ran above. I judged a Docker boot to add no evidence for this diff.

**Merge against current `origin/main`:** code merges cleanly, but **`harness/CODEMAP.md` conflicts** — main's daily PR #17 (`da79c2d`) rewrote the same one-line `google` paragraph (refresh token now sealed via `internal/secrets`). The daily integration merge must resolve it by hand: keep main's paragraph and replace only its `Errors: …` sentence with this branch's. I test-merged in a scratch worktree (CODEMAP taken from the branch): `go vet` + `go test ./internal/google` pass on the merged tree (46 tests). Main's new `token.go`/`secrets` errors (`stored value unusable (%v)`, `secrets: ciphertext did not authenticate`) carry no plaintext, so the "tokens never in an error value" invariant this branch's logging relies on still holds after the merge.

## Quality
- **Boundaries:** everything stays inside `internal/google`; no new exported types; `service.go` untouched. Good.
- **Error handling:** handler ordering is right — `ErrReauthRequired` is checked before `ErrAlreadyExists`, and `service.go:92` still consumes the insert 409 before the handler sees it.
- **Security:** client bodies stay opaque; tokens not logged (verified above, including post-merge). `userID` comes from the JWT, not user input.
- **Nits (not filed):**
  - `truncate` slices bytes, so a 512-byte cut can split a UTF-8 rune in the log; same as `airouter.truncate`, so it matches the convention.
  - `googleErrorReason(raw)` is decoded twice in the 403 reauth branch (once in the guard, once in the message); with an empty reason the message ends with a trailing space (`"calendar returned 403 "`).
  - Google's body is logged raw; a multi-line body produces a multi-line log entry. Harmless for Railway logs, but `%q` would keep one line per failure as CODEMAP promises.
  - Only `errors[0].reason` is inspected; acceptable for Calendar v3 / Tasks v1, and the plan's notes record the `error.status` follow-up.

## Bugs filed
- `harness/ideas/_inbox/google-403-accessnotconfigured-api-disabled-still-maps-to-re.md` — low — config-type 403s (API disabled) still force re-consent; deny-list deviates from the idea's allow-list.
- `harness/ideas/_inbox/no-test-pins-a-409-from-calendar-events-patch-to-502-google-.md` — low — the idea's required events.patch 409 test is missing.

No blockers.

## Verdict
**pass-with-bugs.** All five tasks landed as planned, verification and CI reproduce, and all three ideas are delivered. Two low bugs filed. Heads-up for the daily integration: `harness/CODEMAP.md` conflicts with current `origin/main` and needs a hand merge.
