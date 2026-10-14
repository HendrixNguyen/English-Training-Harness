---
idea: harness/ideas/_inbox/retro-roadmap-loses-centered-column-and-mobile-gutters.md
status: approved
priority: high
merged: false
amends: harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md
branch: harness/2026-10-13-high-retro-screens
worktree: .worktrees/retro-screens
design: harness/designs/retro-final-review-amendments.md
---
# Retro roadmap loses centered column and mobile gutters — Plan

**Design:** `harness/designs/retro-final-review-amendments.md`; existing compatibility and six screen designs remain binding.
**Idea:** `harness/ideas/_inbox/retro-roadmap-loses-centered-column-and-mobile-gutters.md`
**Goal:** Roadmap uses the same centered max-w-md column and 16px gutters at mobile and desktop; focus and zoom remain usable.

## Tasks

### Task 1: Correct final review I2 on the parent branch
Files: `frontend/pages/roadmap.vue` and narrowly related page/store tests, production proof runner, CODEMAP when needed.
- Reproduce the final-review finding and fix the specified contract without changing graded learning, server truth, account lifetimes or auth behavior.
- Roadmap uses the same centered max-w-md column and 16px gutters at mobile and desktop; focus and zoom remain usable.
- Add behavior regression evidence: Production browser container/region geometry at 375 and 1280 pixels and CSS zoom 200%; no hidden controls or clipped focus.
- Implement in the one pooled final-review fix wave with the other amendments. Commit and write scoped verification evidence; controller dispatches one follow-up review.

## Verification
Production browser container/region geometry at 375 and 1280 pixels and CSS zoom 200%; no hidden controls or clipped focus. Frontend lint/typecheck/unit/build for changed source; targeted production reproduction and visual/browser checks as relevant. Backend is unchanged and its independent passed checks remain valid. Final exact-head CI and parent review cover this amendment.
