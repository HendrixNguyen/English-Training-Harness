---
plan: harness/plans/2026-09-27-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/a-failed-google-sync-on-a-bad-refresh-token-logs-two-lines-f.md]
---
# Review — A wrong or rotated ENCRYPTION_SECRET_KEY silently turns every Google sync into reauth_required with no log line

**Plan:** `harness/plans/2026-09-27-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever.md`
**Branch/worktree:** `harness/2026-09-27-medium-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever` / `.worktrees/a-wrong-or-rotated-encryption-secret-key-silently-turns-ever`
**Diff:** `git diff main...harness/2026-09-27-medium-a-wrong-or-rotated-encryption-secret-key-silently-turns-ever --stat`

## Plan vs idea
Delivered, with one exception. The idea asked for:
- One log line for `ErrOpen` with the user id and no value;
- A note for `ErrNotSealed`, and silence for an empty value in `openStored`;
- A distinguishable error;
- A unit test;
- The rotation sentence in `.env.example` / spec §9.

All of it exists. The docs sentence was already present: `backend/.env.example` says rotating or losing the key forces a re-consent, and so do `deploy/README.md` and the spec §9 addendum. The optional boot canary is out of scope, as the plan says. The plan's root-cause premise was stale: `SyncHandler` already logged every failure. So a failure now writes two lines (low, filed).

## Code vs plan
Reviewed at origin head `7b66edc` in a detached reviewer worktree. The diff is 3 files, +84/−7, and it merges cleanly with `origin/main`.
- Task 1, tests: followed. The `captureLog` deviation is justified, because the helper already exists in the package and a copy would not compile.
- Task 2, implementation: followed. `errors.Is` works for both sentinels through the double `%w`, and the comment on the sentinel-only errors is present.
- Task 3, CODEMAP: followed.
- `openStored` has a single caller (`PgRefreshTokenSource.RefreshToken`, called from `Service.Sync`), so the signature change is contained.

```
go test ./internal/google/ -count=1 -v -run 'OpenStored|SyncHandler'  -> 14 PASS, ok
env -u … make check                                                  -> fmt/vet silent, all packages ok under -race
go test ./... -run Integration -p 1 -count=1 (rv-auth stack)          -> all ok
gh run list --branch <branch> --limit 1 -> completed success (run 36296768886)
Live: API with ENCRYPTION_SECRET_KEY=bb…, users row sealed under aa…, POST /api/v1/integrations/google/sync
  {"error":"reauth_required"} HTTP 409
  google: refresh token for user=2222… did not decrypt — check ENCRYPTION_SECRET_KEY if this repeats across users (secrets: ciphertext did not authenticate)
  google: sync user=2222… failed: … stored value unusable: secrets: ciphertext did not authenticate
  leak check (sealed value or plaintext in log): 0
```

## Quality
- Security: no value is logged. `secrets` returns fixed sentinels, and both the unit test and the live leak check confirm it.
- Tests are honest. They cover wrong key and tamper, plus the legacy and empty cases, asserting both the sentinel `errors.Is` and log content and absence.
- Logs: there are two lines per failure (low, filed). An approved Google-tasks plan would call `RefreshToken` on the progress path, where a legacy row would log on every call. Worth keeping in mind; no bug yet.
- Boundaries: `google` → `secrets` (already a dependency). The CODEMAP sentence is accurate.

## Bugs filed
- `harness/ideas/_inbox/a-failed-google-sync-on-a-bad-refresh-token-logs-two-lines-f.md` (low): `openStored` and `logSyncFailure` both log the same failure.

## Verdict
**pass-with-bugs.** Delivered, verified live, CI green, no blockers.
