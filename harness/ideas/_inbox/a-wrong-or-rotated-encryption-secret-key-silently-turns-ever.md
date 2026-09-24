---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# A wrong or rotated ENCRYPTION_SECRET_KEY silently turns every Google sync into reauth_required with no log line

## Why
Google Calendar/Tasks sync is how the product keeps the 30-minute daily habit on the learner's calendar (§5.1 steps 6–7). After this plan, `google.openStored` folds three different conditions into one `ErrNoRefreshToken`: an empty column, a pre-encryption plaintext row (`secrets.ErrNotSealed`), and a value that does not authenticate (`secrets.ErrOpen`). `ErrOpen` covers a tampered row and also a value sealed under a different key. `Service.Sync` turns all three into `ErrReauthRequired`, and `SyncHandler` answers `409 reauth_required`. **Nothing logs on this path.** `grep -n 'log\.' backend/internal/google/*.go backend/internal/auth/*.go` (non-test) finds nothing.

So if an operator deploys with the wrong key (a typo, the staging key in prod, a rotation), every user's sync fails with 409 and the server log shows only `[GIN] 409` lines. That is indistinguishable from a normal "please re-consent", and there is no signal until users complain. Every user then has to re-consent before sync works again. A tampered row is equally silent, and GCM is there precisely to detect that. The test comment at `backend/internal/google/token_test.go:41` says the reason is visible to "the operator [who] reads this in the sync log", but no such log exists.

Rollout path: `.env.example:44` documents what rotating or losing the key does, but nothing at boot or at runtime tells the operator that the key in the environment is not the one the rows were sealed with.

## Expected output
- When `openStored` gets `secrets.ErrOpen`, the process logs one line at warn/error level with the user id and the reason, never the stored value or any plaintext. It stays quiet for empty rows and at most notes `ErrNotSealed` (the expected cutover case). A burst of `ErrOpen` is then visible as a misconfigured key rather than as user churn.
- Optional but cheap: seal a fixed canary at boot and compare it with a stored canary (or store a key fingerprint such as HMAC(key, "fingerprint")). A key mismatch then becomes a loud boot-time warning instead of a mass reauth.
- `.env.example` / backend spec §9 add one sentence on rolling out a key: set it once, never change it without accepting a mass re-consent. A future `v2:` prefix with a two-key `Open` is the rotation path.
- A unit test pins the logging (or the distinguishable error) for `ErrOpen` vs `ErrNotSealed`.

## Evidence
- Plan: `harness/plans/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md` (review: `harness/reviews/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`).
- `backend/internal/google/token.go:58`: `fmt.Errorf("%w: stored value unusable (%v)", ErrNoRefreshToken, err)` collapses ErrOpen and ErrNotSealed.
- `backend/internal/google/service.go:46-47` maps it to `ErrReauthRequired`. `backend/internal/google/handler.go:34-37` answers 409 and logs nothing.
- Reproduced on the reviewer's `rev-secrets` stack (API :18093): a legacy row gives `{"error":"reauth_required"} HTTP 409`, and `grep -c 'ciphertext\|reauth\|no refresh token' api.log` gives `0`.
