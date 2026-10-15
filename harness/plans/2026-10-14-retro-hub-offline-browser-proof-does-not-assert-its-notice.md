---
idea: harness/ideas/_inbox/retro-hub-offline-browser-proof-does-not-assert-its-notice.md
status: done
priority: low
merged: false
amends: harness/plans/2026-10-13-complete-the-remaining-retro-screens-while-preserving-graded.md
branch: harness/2026-10-13-high-retro-screens
worktree: .worktrees/retro-screens
design: harness/designs/retro-final-review-amendments.md
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/64"
---
# Retro hub offline browser proof does not assert its notice — Plan

**Design:** `harness/designs/retro-final-review-amendments.md`; existing compatibility and six screen designs remain binding.
**Idea:** `harness/ideas/_inbox/retro-hub-offline-browser-proof-does-not-assert-its-notice.md`
**Goal:** The offline browser case asserts the actual single notice while preserving cached navigation.

## Tasks

### Task 1: Correct final review M3 on the parent branch
Files: `frontend/tests/e2e/retro-visual.mjs` and narrowly related page/store tests, production proof runner, CODEMAP when needed.
- Reproduce the final-review finding and fix the specified contract without changing graded learning, server truth, account lifetimes or auth behavior.
- The offline browser case asserts the actual single notice while preserving cached navigation.
- Add behavior regression evidence: Actual offline notice is asserted exactly once before screenshot; retained task navigation remains tested in the real-service suite.
- Implement in the one pooled final-review fix wave with the other amendments. Commit and write scoped verification evidence; controller dispatches one follow-up review.

## Verification
Actual offline notice is asserted exactly once before screenshot; retained task navigation remains tested in the real-service suite. Frontend lint/typecheck/unit/build for changed source; targeted production reproduction and visual/browser checks as relevant. Backend is unchanged and its independent passed checks remain valid. Final exact-head CI and parent review cover this amendment.

## Execution summary
Implemented in pooled commit `6451df3`, independently scoped-reviewed with no open findings. The offline browser case asserts the actual single notice while preserving cached navigation.

Verification: focused31 tests; final frontend474/51 plus lint/typecheck/build; production ordering/edit/geometry/cached online-offline probes and visual45/7zoom pass. Exact source-head CI run38575020623 passes all five jobs. Unchanged backend25 real integration zero skips and real-API browser19 were independently reproduced by the parent whole-branch review. The parent final passing review covers this amendment.
