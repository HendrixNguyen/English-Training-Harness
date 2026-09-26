---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md
---
# smoke-api.sh stops at the first unreachable check instead of reporting every check

## Why
`deploy/smoke-api.sh` runs under `set -eu` and assigns `body=$(curl -sS … "$api/healthz")`. When the API is unreachable or times out, curl's non-zero exit aborts the script right there: exit 7/28, no `FAIL` line and no summary. `deploy/smoke-web.sh` behaves differently: its curls sit inside command-substitution arguments, so it prints a `FAIL` line for every check. The owner and a future routine read these outputs after every deploy. A bare `curl: (7)` with no labelled result is the least useful output exactly when the deploy is broken. The exit code is still non-zero, so this is cosmetic rather than a false pass.

## Expected output
- Every check in `smoke-api.sh` prints `ok`/`FAIL <label>` even when the host is down. For example, use `curl … || true` on the assignments, or drop `-e`, and keep `exit $fail`.
- The same output shape as `smoke-web.sh`.

## Evidence
- Plan: `harness/plans/2026-09-25-containerised-deploy-dockerfiles-production-compose-runbook-.md` (branch origin/harness/2026-09-25-high-containerised-deploy-dockerfiles-production-compose-runbook- @ c46b1df); `deploy/smoke-api.sh` lines 6 and 15.
- Reproduced: `deploy/smoke-api.sh http://127.0.0.1:28099 http://x` printed only `curl: (7) Failed to connect …` and exited `7`.

## Evaluation
_Evaluator, 2026-09-26._ **Select — low, folded into** `harness/plans/2026-09-26-caddyfile-serves-index-html-with-a-one-year-immutable-cache-.md` (the Dokploy hardening plan: Caddyfile miss handling + shell `no-cache`, the six AI base-URL/model variables through compose and the runbook table, `smoke-api.sh` reporting every check). The deploy branch it was filed against is on `origin/main` now.
