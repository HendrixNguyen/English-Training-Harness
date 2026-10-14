---
idea: harness/ideas/_inbox/retro-settings-late-read-overwrites-a-confirmed-reminder-sav.md
status: approved
priority: high
merged: false
amends: harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md
branch: harness/2026-10-13-high-retro-screens
worktree: .worktrees/retro-screens
design: harness/designs/retro-final-review-amendments.md
---
# Retro settings late read overwrites a confirmed reminder save — Plan

**Design:** `harness/designs/retro-final-review-amendments.md`; existing compatibility and six screen designs remain binding.
**Idea:** `harness/ideas/_inbox/retro-settings-late-read-overwrites-a-confirmed-reminder-sav.md`
**Goal:** A delayed older read never overwrites a successful newer write or an active user edit; ownership guards remain intact.

## Tasks

### Task 1: Correct final review I1 on the parent branch
Files: `frontend/stores/settings.ts` and narrowly related page/store tests, production proof runner, CODEMAP when needed.
- Reproduce the final-review finding and fix the specified contract without changing graded learning, server truth, account lifetimes or auth behavior.
- A delayed older read never overwrites a successful newer write or an active user edit; ownership guards remain intact.
- Add behavior regression evidence: Deferred same-owner GET→POST→GET resolution, pending read during edit, accepted save retention, account replacement and page lifetime guards; focused settings store/page tests and production browser reproduction.
- Implement in the one pooled final-review fix wave with the other amendments. Commit and write scoped verification evidence; controller dispatches one follow-up review.

## Verification
Deferred same-owner GET→POST→GET resolution, pending read during edit, accepted save retention, account replacement and page lifetime guards; focused settings store/page tests and production browser reproduction. Frontend lint/typecheck/unit/build for changed source; targeted production reproduction and visual/browser checks as relevant. Backend is unchanged and its independent passed checks remain valid. Final exact-head CI and parent review cover this amendment.
