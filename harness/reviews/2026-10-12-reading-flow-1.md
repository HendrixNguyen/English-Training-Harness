---
plan: harness/plans/2026-10-11-reading-flow.md
verdict: fail
bugs: []
---
# Reading flow review — code passed; external acceptance blocked

Plan: `harness/plans/2026-10-11-reading-flow.md`. Branch: `harness/2026-10-11-high-reading-flow`. Range: `249d9c0..7df8116`.

All four implementation tasks passed task-scoped reviews. Final whole-branch review found one minor phrase-collision omission; `7df8116` fixes it and scoped review found no introduced breakage. No open code findings or bug artifacts remain.

Actual API browser proof passed3/3, disposable database integration21/21 without skips, final frontend339 unit tests and build/lint/typecheck, backendcheck and harness50 tests/validation passed. SQL/Redis confirm durable completion and time; reopening persisted content needs no new provider call. All five remote jobs passed on the final head: https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/38194501857.

Overall acceptance verdict is fail only because genuine `English-bot-free` output through `https://9router.huyndg.io.vn/v1` remains unverified. `NINE_ROUTER_API_KEY` is unset and environment settings report no saved binding. The deterministic provider proves integration, not live AI feasibility. Supply the key securely, verify real generation latency/tokens/schema and cached reuse, then re-review. No secret was copied from chat; no PR/merge/deploy occurred.

Scope decisions: lazy per-exercise generation preserves outline roadmap/level guidance; normal Go decoding permits optional blank/null glosses and the client drops invalid entries; a new timer starts when content is usable while existing anchors persist. SQL completion/daily minutes and Redis seconds replace nonexistent study-log assertions. Offline app re-entry is verified; full-document offline reload requires separate app-shell work. Concurrent first calls may generate more than once but persist one winner. Ownership/activity is authorized before generation. Existing timer/progress behavior, answer grading/history, speech, full redesign and unrelated backlog items remain separate. Native keyboard/focus and accessibility semantics pass; screen-reader combinations remain unverified. If these decisions are wrong, follow-up work affects provider coalescing/latency, timing expectations, grading/storage, revocation, offline app shell, speech or manual accessibility testing; no passing claim was made for those deferred behaviors.

Detailed local reports, all rulings and screenshot/database evidence are retained under `/workspace/cloud-setup/reading-flow/evidence`; Task1 report remains `/workspace/cloud-setup/reading-flow/task-1-report.md`. Reproducible proof instructions are tracked in the feature branch's `frontend/tests/e2e/READING.md`.
