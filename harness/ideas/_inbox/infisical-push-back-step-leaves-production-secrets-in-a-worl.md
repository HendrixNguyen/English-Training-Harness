---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Infisical push-back step leaves production secrets in a world-readable /tmp file when the upload fails

## Why
Step 3 of the runbook's new "Env source of truth — Infisical" section tells the owner to run `grep -E '^[A-Z_]+=.+' deploy/.env > /tmp/nonblank.env && infisical secrets set --env prod --file /tmp/nonblank.env && rm /tmp/nonblank.env`. That has two problems. (1) Under the owner's umask (`022`, the macOS default), the redirect creates `/tmp/nonblank.env` mode 644, so every non-blank production secret (JWT, encryption key, OAuth secret, DB/Redis URLs) sits world-readable in a shared directory. (2) The `&&` chain skips `rm` whenever `infisical secrets set` fails (not logged in, network, a rejected key), so the file survives exactly when things go wrong. Separately, the *Environment* table's "Railway split" column still says `set on the service` for every secret (`deploy/README.md:14` onward). Step 5 of the same section says Railway variables are "never edited by hand". The caddyfile plan adds six more rows with the same wording.

## Expected output
Step 3 no longer writes secrets to a predictable shared path. Use either `infisical secrets set --env prod --file <(grep -E '^[A-Z_]+=.+' deploy/.env)`, if the CLI accepts a FIFO (verify first), or `f=$(mktemp) && chmod 600 "$f" && trap 'rm -f "$f"' EXIT && grep … > "$f" && infisical secrets set --env prod --file "$f"`. The Railway column of the env table says "synced from Infisical `prod`" (or the column header says so once), consistent with step 5. Agents still never handle the values; this is a text change to the runbook.

## Evidence
- Plan: `harness/plans/2026-09-26-infisical-is-the-source-of-truth-for-deploy-secrets-link-fil.md`, branch `harness/2026-09-26-medium-infisical-is-the-source-of-truth-for-deploy-secrets-link-fil`, `deploy/README.md` section "Env source of truth — Infisical" step 3 (≈ line 37) and step 5 (line 39); env table lines 7-30 (`JWT_SECRET … | set on the service |`).
- `umask` on the owner's machine → `022` (new files 644).
