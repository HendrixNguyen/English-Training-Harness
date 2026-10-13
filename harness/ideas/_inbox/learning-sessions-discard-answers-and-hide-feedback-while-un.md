---
type: bug
status: planned
source: human
run: _inbox
priority: high
blocks: harness/plans/2026-10-11-reading-flow.md
plan: harness/plans/2026-10-12-learning-sessions-discard-answers-and-hide-feedback-while-un.md
---
# Learning sessions discard answers and hide feedback while untrusted drafts bypass generation

## Why
The owner reports wrong learning progress/data and “same answer, meaningless” at https://english-learning-e6a.pages.dev/learn/b1a0f1bc-1ee0-4ea2-b51e-d4f203b9df84. A session must teach from real task content, explain submitted answers, and persist the learner's actual results. Silently dropping answers and hiding vocabulary quizzes breaks that happy path. The owner authorized a subagent-driven correction; no production publishing is authorized in this amendment.

## Expected output
An authenticated task hydrates generated content rather than trusting outline drafts. Generated questions reject identical choices, repeated prompts and obvious placeholders; existing invalid stored content follows a bounded repair policy. Vocabulary study includes examples then its quiz; reading retains passage/glossary then each question requires a choice, shows feedback/explanation, and advances separately. The summary separates answered-question progress, actual server-derived score and study time. Answers and score are durably saved with completion on the owned task and returned on reload. Valid zero-question legacy reading/word-only material remains usable and explicitly ungraded. Timer anchors, study-time semantics, tracking write order and glossary keyboard interactions remain intact.

## Evidence
Read-only findings at reviewed feature HEAD `7df8116` in `.worktrees/reading-flow`: `/workspace/cloud-setup/reading-flow/learning-data-diagnosis.md` and `learning-ui-diagnosis.md`. Confirmed paths: `backend/internal/quests/handler.go` accepts then drops `user_answers`; `service.go` saves time/completion only; `frontend/utils/content.ts` drops vocabulary questions via words-first classification and considers passage-only outlines usable; `components/learn/ContentViewer.vue` skips unanswered questions and omits correctness/explanation; `airouter/content.go` accepts duplicate visible choices/prompts/placeholders. Schema-1 stored content returns without revalidation in `quests/exercise_content.go` and conditional persistence excludes it. Learner-local question state restarts on reload while timers resume.

Production GitHub branch was observed at `2d9cd639`, which lacks this lazy reading flow; `7df8116` is not its ancestor. No question text, private live task payload, production DB or authenticated production access is available. These reproducible code defects justify the correction but do not establish the exact live question's cause or reproduce its choices. A repeated correct answer letter across questions can be valid. Structural rejection is not proof of pedagogical or linguistic correctness.

## Evaluation
Selected, high impact blocker of `harness/plans/2026-10-11-reading-flow.md`. The reviewed-but-unmerged reading branch must include meaningful quiz feedback and durable answers before its end-to-end learning claim is sufficient. This bounded amendment is feasible in one concentrated agent-day with sequenced content, persistence and UI work; it uses existing task content, answer keys, quizzes and stores. It needs no essay AI grading, mastery model, world map, nonexistent study_logs table or new external service. The schema change uses the existing exercises table with a later additive migration. No broader timer/tracking refactor is included.

Root causes are traced in the focused diagnostics and source read-only review: outline and generated provenance are conflated; validators compare keys but not visible text; cached schema markers bypass current validation; words-first rendering hides associated questions; submission conflates choice and advance; the progress service signature never receives answers. The root checkout preserves older code and historical artifacts; the current amendment is defined against the existing feature worktree, not inferred to be deployed.

Required designer role will bind the small in-room state transition design before frontend tasks are finalized. Execute only on `.worktrees/reading-flow`, `harness/2026-10-11-high-reading-flow`, starting from `7df8116`; producer interfaces precede consumers. Fresh genuine-provider/API/PostgreSQL/Redis/browser proof, branch CI and reviews must establish the implemented result. External PR, merge and production release remain separate gates. Preserve original done frontmatter/reviews.
