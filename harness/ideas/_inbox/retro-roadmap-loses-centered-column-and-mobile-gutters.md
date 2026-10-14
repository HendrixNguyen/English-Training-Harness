---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md
plan: harness/plans/2026-10-14-retro-roadmap-loses-centered-column-and-mobile-gutters.md
---
# Retro roadmap loses centered column and mobile gutters

## Why
The roadmap stretches to desktop width and has no mobile gutters.

## Expected output
Roadmap uses the same centered max-w-md column and 16px gutters at mobile and desktop; focus and zoom remain usable.

## Evidence
Final review I2, parent plan `harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md`, `frontend/pages/roadmap.vue:72`. Production and source evidence at `/workspace/cloud-setup/retro-screens/final-review/`; final review retained in this plan’s SDD archive.

## Evaluation
Select high: bounded correction to the already authorized screens. Root cause reproduced by the independent review. Existing screen contracts remain binding; the pooled designer amendment confirms acceptance before planning.
