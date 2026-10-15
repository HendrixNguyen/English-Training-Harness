---
idea: harness/ideas/_inbox/cached-retro-roadmap-reentry-omits-current-region-initializa.md
status: done
priority: low
merged: false
amends: harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md
branch: harness/2026-10-13-high-retro-screens
worktree: .worktrees/retro-screens
design: harness/designs/retro-final-review-amendments.md
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/64"
---
# Cached retro roadmap reentry omits current region initialization — Plan

**Design:** `harness/designs/retro-final-review-amendments.md`; existing compatibility and six screen designs remain binding.
**Idea:** `harness/ideas/_inbox/cached-retro-roadmap-reentry-omits-current-region-initializa.md`
**Goal:** A new page entry initializes today from an accepted cached snapshot, including offline; mounted unchanged refresh preserves selected history and scroll.

## Tasks

### Task 1: Correct final review M1 on the parent branch
Files: `frontend/pages/roadmap.vue` and narrowly related page/store tests, production proof runner, CODEMAP when needed.
- Reproduce the final-review finding and fix the specified contract without changing graded learning, server truth, account lifetimes or auth behavior.
- A new page entry initializes today from an accepted cached snapshot, including offline; mounted unchanged refresh preserves selected history and scroll.
- Add behavior regression evidence: Fresh page re-entry with same cached roadmap/date and offline entry initializes today; unchanged mounted refresh retains historical disclosure/viewport; changed date still advances.
- Implement in the one pooled final-review fix wave with the other amendments. Commit and write scoped verification evidence; controller dispatches one follow-up review.

## Verification
Fresh page re-entry with same cached roadmap/date and offline entry initializes today; unchanged mounted refresh retains historical disclosure/viewport; changed date still advances. Frontend lint/typecheck/unit/build for changed source; targeted production reproduction and visual/browser checks as relevant. Backend is unchanged and its independent passed checks remain valid. Final exact-head CI and parent review cover this amendment.

## Execution summary
Implemented in pooled commit `6451df3`, independently scoped-reviewed with no open findings. A new page entry initializes today from an accepted cached snapshot, including offline; mounted unchanged refresh preserves selected history and scroll.

Verification: focused31 tests; final frontend474/51 plus lint/typecheck/build; production ordering/edit/geometry/cached online-offline probes and visual45/7zoom pass. Exact source-head CI run38575020623 passes all five jobs. Unchanged backend25 real integration zero skips and real-API browser19 were independently reproduced by the parent whole-branch review. The parent final passing review covers this amendment.
