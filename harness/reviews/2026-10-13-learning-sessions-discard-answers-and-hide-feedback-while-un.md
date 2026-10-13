---
plan: harness/plans/2026-10-12-learning-sessions-discard-answers-and-hide-feedback-while-un.md
verdict: pass
bugs: []
covers: [harness/plans/2026-10-11-reading-flow.md]
---

# Learning sessions save answers and explain each question — accepted

Reviewed application source815c9bed9371840dabf24a3ba6468c1caadae2e1 on harness/2026-10-11-high-reading-flow. This review covers the learning-session amendment and original reading-flow plan.

Learning starts with usable generated or recognized legacy material, retains vocabulary examples/questions and reading passages/glosses, requires submitted answers and separate feedback/Continue, then explicitly saves server-derived choices/results/duration. Owned content revisions, bounded explicit repair, strict progress validation and atomic completion/result storage prevent grading replacement content or inventing historical scores. Older revisionless calls stay ungraded.

Fresh whole-branch review and scoped re-reviews resolved every Important finding: stale reads before and after save; persisted completion/cache coherence; central off-route sign-out cleanup; stable-account daily/pet cache isolation against late prior-owner writes; same-owner session and previous-token lifetime guards; immediate request-frame checks after helper awaits. No Critical or Important finding remains. CODEMAP's old dropped-answer contract was corrected.

Verification:393 frontend unit tests/45 files, lint/typecheck/fresh build;17 actual authenticated API/PostgreSQL/Redis/service-worker browser scenarios plus one targeted scenario covering four real session/renewal responses;23 serialized race-enabled integration tests with zero skips/failures; backend build/fmt/vet/race checks;50 harness tests. Exact-source [CI38332659347](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/38332659347) passes backend-unit, backend-integration, frontend, docker-images and harness-tooling. Final artifact-head CI is rechecked after this bookkeeping commit. No check was skipped or weakened.

Fresh selected9Router English-bot-free samples pass actual generation, manual grounding inspection, correct/wrong browser grading and SQL/Redis result/time/cache/reload proof for reading, context-independent practice and vocabulary. Latest vocabulary (managed configuration11) has six words/examples/four questions; submitted B,A,A,C against B,B,A,C yields3/4,75%,66seconds,1minute and onePOST. Prior INVALID_MODEL_ID runs remain failures in the retained history, not passing evidence. Deterministic fixtures prove transport/recovery only.

Limits remain disclosed: one genuine practice option gives a supported but imprecise causal answer; some generated sentences omit articles. These bounded samples and structural validation do not establish universal future language quality/difficulty. Existing cross-store tracking can require manual time reconciliation after partial failure; denied storage/deletion only permits in-memory fallback; full-document offline shell reload is outside scope. Exact private production choices were not retrieved.

Full task/review/probe/provider/browser evidence is retained under /workspace/cloud-setup/reading-flow and the plan-specific SDD directory until archived. Reproducible commands and contracts are in frontend/tests/e2e/READING.md and canonical specs. Production is unchanged; integration/release remains a separate controller step.
