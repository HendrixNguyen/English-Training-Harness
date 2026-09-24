---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# config.go still says JWT_SECRET is not in the 1st-thinking env list, and boot refusals print config: config:

## Why
Documentation drift introduced by the plan itself, plus an operator-facing stutter:
1. `backend/internal/config/config.go:29` still reads `// NOTE: JWT_SECRET is NOT in the 1st-thinking §8 list — see CODEMAP auth.` The same branch added `JWT\_SECRET` to 1st-thinking §8 and to backend spec §9, and CODEMAP now says "both are now in backend spec §9 and 1st-thinking §8". The comment contradicts both.
2. Every boot refusal prints a doubled prefix: `config: config: JWT_SECRET must be at least 32 bytes (…)`. `cmd/api/main.go:49` does `log.Fatalf("config: %v", err)` on errors that already start with `config:`. This predates the plan (origin/main `main.go:48`), but the plan's new operator-facing messages are exactly what an operator will now read on Railway. The execution summary also recorded these lines without the doubled prefix, so it paraphrased rather than pasting real output.

## Expected output
- The `config.go:29` comment is removed or rewritten to point at backend spec §9 / 1st-thinking §8.
- The boot refusal prints one `config:`: either `main.go` logs `%v` alone, or `config` drops its own prefix. Every `config_test.go` substring check still passes.

## Evidence
- Plan: `harness/plans/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md` (review: `harness/reviews/2026-09-24-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`).
- Reviewer re-run: `env -i … JWT_SECRET=x ./api` → `2026/09/24 22:14:37 config: config: JWT_SECRET must be at least 32 bytes (generate one with: openssl rand -base64 32)`, exit 1.
