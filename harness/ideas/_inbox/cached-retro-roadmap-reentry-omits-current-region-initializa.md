---
type: bug
status: planned
source: reviewer
run: _inbox
priority: low
plan: harness/plans/2026-10-14-cached-retro-roadmap-reentry-omits-current-region-initializa.md
---
# Cached retro roadmap reentry omits current region initialization

## Why
Returning with cached same-day data skips today disclosure and initial scroll.

## Expected output
A new page entry initializes today from an accepted cached snapshot, including offline; mounted unchanged refresh preserves selected history and scroll.

## Evidence
Final review M1, parent plan `harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md`, `frontend/pages/roadmap.vue:44`. Production and source evidence at `/workspace/cloud-setup/retro-screens/final-review/`; final review retained in this plan’s SDD archive.

## Evaluation
Select low: bounded correction to the already authorized screens. Root cause reproduced by the independent review. Existing screen contracts remain binding; the pooled designer amendment confirms acceptance before planning.
