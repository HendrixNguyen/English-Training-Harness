---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-10-14-retro-hub-offline-browser-proof-does-not-assert-its-notice.md
---
# Retro hub offline browser proof does not assert its notice

## Why
Waiting on a preexisting task button does not prove the offline notice appears.

## Expected output
The offline browser case asserts the actual single notice while preserving cached navigation.

## Evidence
Final review M3, parent plan `harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md`, `frontend/tests/e2e/retro-visual.mjs:61`. Production and source evidence at `/workspace/cloud-setup/retro-screens/final-review/`; final review retained in this plan’s SDD archive.

## Evaluation
Select low: bounded correction to the already authorized screens. Root cause reproduced by the independent review. Existing screen contracts remain binding; the pooled designer amendment confirms acceptance before planning.
