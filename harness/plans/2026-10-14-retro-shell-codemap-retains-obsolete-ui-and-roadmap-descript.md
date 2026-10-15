---
idea: harness/ideas/_inbox/retro-shell-codemap-retains-obsolete-ui-and-roadmap-descript.md
status: done
priority: low
merged: false
amends: harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md
branch: harness/2026-10-13-high-retro-screens
worktree: .worktrees/retro-screens
design: harness/designs/retro-final-review-amendments.md
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/64"
---
# Retro shell CODEMAP retains obsolete UI and roadmap description — Plan

**Design:** `harness/designs/retro-final-review-amendments.md`; existing compatibility and six screen designs remain binding.
**Idea:** `harness/ideas/_inbox/retro-shell-codemap-retains-obsolete-ui-and-roadmap-descript.md`
**Goal:** The shell overview describes current interfaces and visuals without obsolete derived roadmap or scale-era claims.

## Tasks

### Task 1: Correct final review M2 on the parent branch
Files: `harness/CODEMAP.md` and narrowly related page/store tests, production proof runner, CODEMAP when needed.
- Reproduce the final-review finding and fix the specified contract without changing graded learning, server truth, account lifetimes or auth behavior.
- The shell overview describes current interfaces and visuals without obsolete derived roadmap or scale-era claims.
- Add behavior regression evidence: Read all shell overview claims against current implementation and module/API entries; harness validate.
- Implement in the one pooled final-review fix wave with the other amendments. Commit and write scoped verification evidence; controller dispatches one follow-up review.

## Verification
Read all shell overview claims against current implementation and module/API entries; harness validate. Frontend lint/typecheck/unit/build for changed source; targeted production reproduction and visual/browser checks as relevant. Backend is unchanged and its independent passed checks remain valid. Final exact-head CI and parent review cover this amendment.

## Execution summary
Implemented in pooled commit `6451df3`, independently scoped-reviewed with no open findings. The shell overview describes current interfaces and visuals without obsolete derived roadmap or scale-era claims.

Verification: focused31 tests; final frontend474/51 plus lint/typecheck/build; production ordering/edit/geometry/cached online-offline probes and visual45/7zoom pass. Exact source-head CI run38575020623 passes all five jobs. Unchanged backend25 real integration zero skips and real-API browser19 were independently reproduced by the parent whole-branch review. The parent final passing review covers this amendment.
