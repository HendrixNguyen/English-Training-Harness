# Design: Learning answers that teach and remain saved

**Idea:** `harness/ideas/_inbox/learning-sessions-discard-answers-and-hide-feedback-while-un.md`
**Inherits:** `harness/designs/frontend-shell.md` §2.4; `harness/designs/tap-a-hard-word-vietnamese-glosses-on-every-reading-passage-.md`; `harness/designs/retro-learning-room.md` for answer feedback intent only.
**Kit:** `harness/UI-KIT.md` v2 supersedes frontend spec §6.1 appearance.
**Spec wireframe:** frontend spec §7.3: retain back/timer, one question, lettered options and one primary action. Add separate submission/feedback/advance phases and a saved-result summary to make each answer useful. This amendment supersedes inherited timer-bypass completion, viewer restart and local-only grading. Full retro-room restyle, flashcard ratings, XP, chest, audio and offline write queue are outside scope.

## 0. Research
- **Learner's job:** study the actual word or passage, answer its questions, understand each result, and know that the task's answers were saved.
- **The moment that earns the next minute:** a wrong choice stays visible beside the correct one and the explanation; the learner can reread the passage before moving on.
- **Real UI evidence:** read-only source at `.worktrees/reading-flow` HEAD `7df8116`, specifically `ContentViewer.vue`, `pages/learn/[id].vue`, `utils/content.ts`, and `stores/quest.ts`. The viewer advances on submission without feedback, permits unanswered advance, and words-first classification loses vocabulary quizzes/examples. Page completion permits timer expiry. Local answers/cursor reset on reopen. Progress accepts an answer map but the backend drops it.
- **Visual evidence:** inspected `/workspace/cloud-setup/reading-flow/evidence/reading-flow-task4-gloss-390x844.png`: existing dark passage/gloss panel inside the light AppCard, with working timer/back and inline lookback primitives. Keep this accepted interim shell; use existing retro primitives for changed inner regions rather than redesigning the route.
- **Evidence limit:** production URL `https://english-learning-e6a.pages.dev/learn/b1a0f1bc-1ee0-4ea2-b51e-d4f203b9df84` has no supplied authenticated question payload; observed production `2d9cd639` lacks this new flow. No live question reproduction or linguistic-quality claim is made.
- **Answered questions:** repeated correct answer letters can be valid; do not shuffle merely to make them differ. Generated questions require valid keys/explanations; valid legacy questions may lack keys, and legacy passage/word-only material remains explicitly ungraded. A score comes from server-stored content, never elapsed time. No new token, dependency or component is needed.

## 1. The signature
The answer changes the same question into a short lesson. “Trả lời” reveals the selected choice, correct choice and explanation; only “Tiếp tục” leaves that lesson. The summary distinguishes answered questions, saved score and study time.

## 2. Flow
Hub quest → owner-checked task hydration → reading passage / vocabulary word study / practice quiz → question selection → submitted feedback → next question → unsaved summary → “Hoàn thành” → saved summary → “Về trang chính” → hub with existing time/completion/pet updates.
- Reading: passage first, then Q questions. “Xem lại đoạn” remains available during selection and feedback, with glossary interactions intact.
- Vocabulary: show every word's term, definition and optional example, labelled “Từ n / W”; after the last word “Làm câu hỏi” starts associated questions. Word-only legacy material goes to an ungraded summary.
- Reading with zero legacy questions: “Đọc xong →” reaches an ungraded summary. Question-only legacy/practice starts with “Câu 1 / Q”.
- Final question: “Tiếp tục” is required after feedback before reaching summary. No automatic save or redirect.
- Back preserves unsaved draft and timer anchors. Reopening completed tasks reads the server summary, with no grading or POST. Existing hub reward flow remains, using the real progress response.

## 3. Layout (mobile-first, max-w-md; desktop keeps the same column)
Retain existing outer page/top bar. Changed regions use v2 tokens and existing primitives; no pagewide cosmetic migration.
1. Top bar: “‹ Quay lại”, existing CountdownTimer; timer remains a countdown holding at 00:00, with aria-live off.
2. Study panel: existing ItemPassage; vocabulary uses an existing RetroPanel with English term/definition/example in font-body 17/26, not pixel type. Word example only appears when supplied. Keep passage 18/30 and existing 50/40-vh frames.
3. Quiz panel: RetroPanel, eyebrow “Câu n / Q” in display counter type plus “Đã trả lời a / Q”; n is cursor position, a counts submitted question IDs only. Passage and vocabulary cards never increment a.
4. Reading lookback: existing RetroButton secondary and ItemPassage sheet above prompt; keep the h-11 eyebrow slot free.
5. Prompt/options: English font-body 17/26; options A–D in 56-px rows, radius ≤2 px, ground-1/ground-2, line-lit borders. Selection has ▶ and aria-checked; keyboard focus uses torch. Submitted rows are locked. Selected wrong row has ember + “Bạn chọn”; correct row has growth + “Đáp án đúng” + ✔; correct selected row carries both labels.
6. Feedback: existing RetroPanel tone growth/ember/plain below options; one polite announcement of “Trúng!”, “Trượt…” or “Đã ghi nhận lựa chọn”. English explanation renders as text below. No correctness/key/explanation in DOM before submission. Legacy without key shows no correct option or invented explanation.
7. Primary RetroButton: “Tiếp tục” for words, “Làm câu hỏi” on final word with quiz, “Đọc xong →” for passage, “Trả lời” during selection, “Tiếp tục” after feedback. Only this action progresses an active item; page save control does not appear during active quiz.
8. Summary RetroPanel: task title, “Đã trả lời a / Q”, actual elapsed study-time line. Unsaved: “Chưa lưu kết quả”, no server score claim. Saved graded: “Đúng c / Q · s%”; saved ungraded: “Không chấm điểm”. Saved answers list prompt, selected option, returned correct option/is_correct/explanation when non-null. Content text remains English and escaped. Long lists flow vertically; no second column. Primary “Hoàn thành” before save; “Về trang chính” after save.

## 4. States
| State | Learner-visible behavior / copy |
|---|---|
| Loading | Existing StateBlock; “Đang tải nội dung…”; no answer/save action; start timer only once usable trusted content exists, preserving existing anchors. |
| First entry | No tutorial modal. Unselected question shows disabled “Trả lời” plus “Chọn một đáp án rồi trả lời.” Existing glossary caption remains. |
| Empty / unavailable task | StateBlock “Nhiệm vụ này không còn khả dụng.” + “Về trang chính”; no raw JSON. |
| Invalid generated content | 409 exercise_content_invalid: StateBlock “Nội dung này cần được tạo lại.” + explicit “Tạo lại nội dung”. No silent AI regeneration, no invalid quiz or timer-bypass save. |
| Repair failure | “Chưa tạo lại được nội dung. Thử lại.”; action remains explicit. Server permits at most two malformed attempts, then “Chưa tạo được nội dung phù hợp. Quay lại sau.” Completed results are never repaired. Network/rate-limit failures keep existing bounded/error policy. |
| Offline trusted cache | Cached validated generated/legacy task remains fully usable; “Cần kết nối để ghi tiến độ”. Draft and feedback work; summary save disabled. Zero-question material shows “Không chấm điểm”. |
| Offline outline / ambiguous old cache | “Cần kết nối để tải nội dung. Kết nối rồi thử lại.” + “Thử lại”; do not treat roadmap nested drafts as generated content. |
| Keyless legacy question | Selection still required; after submit “Đã ghi nhận lựa chọn” and “Câu này chưa có đáp án để chấm.” No correct/wrong claim. |
| Unsaved summary | Answer count and time, “Chưa lưu kết quả”; no claim that draft is durably saved. |
| Saving | Primary loading/disabled; retain draft; no duplicate POST from double click. |
| Confirmed save rejection | “Chưa ghi được tiến độ. Thử lại.”; choices and time snapshot retained; explicit save retry after reconciliation if needed. |
| Unknown save outcome | “Đang kiểm tra kết quả đã lưu…” while refetching daily. Failure: “Chưa xác nhận được kết quả. Kết nối rồi kiểm tra lại.” + “Kiểm tra lại”; keep re-save disabled until matching saved result found or confirmed incomplete. |
| Content changed | Discard incompatible draft; “Nội dung đã thay đổi. Học lại từ đầu.”; hydrate new revision before any new save. |
| Storage unavailable | Session remains usable in memory; “Tiến độ chưa lưu chỉ giữ trong lần mở này.”; server save still works. |
| Done / completed reopen | “Đã lưu kết quả” and authoritative quiz_result; completed task with no historical result: “Không có kết quả câu hỏi đã lưu”; never fabricate score, never POST. |
| Reduced motion | Feedback/lookback appears in one step; no new animation. Existing button pressed state remains; no sprite reaction, typing or page transition is introduced. |

## 5. Interactions and feedback
- Select option: cursor/aria-checked changes within 100 ms. Persist pending selection in draft, separately from submitted answers. Arrow keys move selection; Enter/Space select; primary “Trả lời” submits. Focus does not jump when selecting.
- Submit: disabled without selection; both UI and handler guard it. Record question ID→A–D once in draft, lock choices, reveal feedback in place within 100 ms, and announce once. Submission does not advance. Repeated Enter/tap cannot resubmit or skip feedback.
- Continue: close any gloss card; move to next question or unsaved summary; focus next question heading or summary heading. Persist cursor/phase. Back/reload on feedback restores locked selection and explanation without adding another answered count.
- Glossary/lookback: inherit exact text preservation, Tab/Enter/Space, Esc, focus return, aria-controls/expanded and one-open-card rules. Lookback open choice survives questions and draft resume; open gloss card itself need not persist. Timer advances during lookback and feedback.
- Timer expiry: keep counting semantics/elapsed anchors unchanged; no “Hết giờ — Hoàn thành”, skip, automatic completion, answer selection or score effect. Complete eligibility is all Q questions submitted plus final Continue; zero-question legacy eligibility is all study material viewed.
- Save: POST once per intentional attempt through quest store; retain duration snapshot on failed/uncertain attempt. Disable while saving/reconciling. This UI always sends content_revision and all required submitted questions. Server validates owner, matching revision, question IDs, selections and required completeness before any tracking writes on this path. Only a successful persisted quiz_result enables “Đã lưu kết quả”. Rolling compatibility for older revisionless callers is defined below and does not relax this UI's full-answer gate.
- Ambiguous network outcome: refetch daily before permitting manual re-save. If same task/revision is completed with matching saved answer map/result, apply authoritative result and clear draft without a second POST; if confirmed incomplete, permit explicit retry. If still unknown, offer read-only “Kiểm tra lại”. This is client reconciliation, not a claim of globally exactly-once tracking.
- Account/task switches: existing generation/session guards must suppress late hydration, repair, progress and reconciliation responses. Never restore another owner's/task's draft. Auth token renewal alone must not change draft identity.

**API and persistence contract (additive; controller ruling binds rolling compatibility):**
- Task from exercise GET and daily: existing fields plus `content_origin: 'generated'|'legacy'|'outline'`, `content_revision: string` (opaque stable fingerprint of server-stored content and task type), and `quiz_result` nullable.
- Outline nested roadmap content requires hydration. Invalid persisted generated content is a 409, repaired only through explicit authenticated `GET /api/v1/quests/exercises/:id?repair=true`, with bounded attempts and original-row CAS. Trust recognized root-level legacy shapes or explicit legacy provenance; do not infer trust from schema marker alone.
- Progress POST from the new UI: `{exercise_id, duration_seconds, content_revision, user_answers: {[questionId]: 'A'|'B'|'C'|'D'}}`; always send revision and all required answers; zero-question legacy sends an empty map. Do not send client score or answer key. Supplied revision mismatch returns 409 before tracking writes. Matching revision enables server grading from stored content; keyless/zero-question content remains ungraded.
- Older API callers may omit content_revision: preserve their existing valid time/completion behavior and allow partial/no answers. Any supplied valid selections are stored as UNGRADED even when stored keys exist: correct_count, score_percent and per-answer correct_option/is_correct/explanation are all null, avoiding grading against changed content. answered_count honestly counts supplied valid answers and question_count reflects actual stored questions. Malformed answer maps, unknown question IDs or invalid options still return 400 before writes. An existing completed quiz_result is never overwritten by a legacy time-only/partial request. This compatibility path does not permit the new UI to skip questions or claim a score.
- Progress response adds `quiz_result: {status: 'graded'|'ungraded', answered_count, question_count, correct_count: number|null, score_percent: number|null, answers: [{question_id, selected_option, correct_option: string|null, is_correct: boolean|null, explanation: string|null}]}`, derived from stored content, persisted with completion. Same result returns on daily task for reload. Treat all server counts/nulls as authoritative; an ungraded result has no numerical score. Existing progress/pet fields keep their meaning/write order.
- Local feedback may compare a supplied valid key after submission; it is a preview. Saved summary always replaces preview with returned values.
- Unsaved draft key: versioned namespace + stable owner ID + exercise ID + content_revision; body stores study/quiz/summary phase, word/question cursor, pending selection, submitted map, feedback state, passageOpen and failed-save duration snapshot. Validate shape/question IDs/options on restore. Read only after trusted content/revision obtained; discard on mismatch or completed task. Sign-out/account change clears former owner's active draft; task changes use that task's isolated draft. No token, score, timer anchor or private cross-account cache belongs in the draft.
- Success clears matching draft and preserves existing timer cleanup; saved result is server state. LocalStorage denial/malformed draft cannot block study. No change to `aelp.timers` or progress tracking algorithm is included.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| ContentViewer | existing | content includes words/examples/questions and optional answer/explanation; controlled session draft; answer, session-change, finished events | Study→selection→feedback→advance; no timer access, no new ItemQuestion/Flashcard dependency. |
| ItemPassage / GlossTerm / GlossCard | existing | existing props/events/eyebrow-action slot | Preserve layout, text and keyboard behavior. |
| RetroPanel / RetroButton / StateBlock / CountdownTimer | existing kit | existing contracts | Reuse for quiz feedback, summary, repair/save states; keep outer shell. |
| Learn page / quest store | existing | owner/task/revision-bound draft, provenance hydration, complete response/result | Page orchestrates save/reconciliation; store keeps authoritative task/result and existing progress/time rules. |
**Kit additions:** none. No new token, font, radius, animation, XP/reward mechanism or dependency.

## 7. Copy
New/changed strings are specified in §§2–5 and the state table. Dynamic labels: “Từ {n} / {W}” · “Câu {n} / {Q}” · “Đã trả lời {a} / {Q}” · “Thời gian học: {m} phút {s} giây” · “Đúng {c} / {Q} · {s}%” · “Bạn chọn: {letter}” · “Đáp án đúng: {letter}”. Actions: “Trả lời” · “Tiếp tục” · “Làm câu hỏi” · “Đọc xong →” · “Hoàn thành” · “Về trang chính” · “Tạo lại nội dung” · “Thử lại” · “Kiểm tra lại”. Existing passage/gloss strings and accessible names remain unchanged. App instructs; English content remains English. No companion speech is introduced.

## 8. Self-critique
- Explicit save adds one tap but makes saved vs unsaved clear; no score toast or forced redirect can substitute for the persistent summary.
- Existing outer AppCard remains an interim shell. New inner regions obey v2; applying a full retro migration or new flashcard game would exceed this bug scope.
- Structural checks cannot prove English meaning; a repeated answer letter alone is never a defect. Real-provider evidence must include actual content inspection and label that limitation.
- Drafts improve continuity, not durable result storage. Never report selection as answered, draft as saved, word/passage as question, absent legacy key as wrong, or time as score.
- A save timeout may already have committed; retry only after reconciliation. Preserve existing tracking/write order and make no new global exactly-once claim.
- Executor traps: words must retain questions; returned server grade must win; content replacement invalidates draft; first hydration and completed reload must not flash stale content; key reveal must wait for submission; old empty result must not turn into 0%.

## Acceptance
1. Real generated vocabulary shows every term, definition and supplied example before its associated quiz; words are labelled separately and do not increase answered count. Reading opens passage-first and retains exact glossary/lookback behavior.
2. Each quiz requires a selection before “Trả lời”; submit locks the choice and shows text/icon correctness, correct option and supplied explanation before a separate “Tiếp tục”. Legacy keyless questions show recorded/ungraded feedback. No key or explanation is exposed before submission.
3. Neither repeated clicks/Enter nor 00:00 can skip an unanswered question or finish a partial generated quiz. Final feedback remains until Continue; all required submitted questions are necessary for save.
4. Answered progress counts submitted question IDs independently of study cursor/time. Valid legacy zero-question passage/word material remains usable online/offline and finishes explicitly ungraded, with no fabricated 0%/100%.
5. Successful progress persists actual A–D answers and server-derived quiz_result, then shows authoritative count/score/explanation; reload of completed task returns the same result and performs no extra POST. Null ungraded fields never become numerical score.
6. Double-click Save produces one in-flight POST. Failed saves retain choices/time. Lost-response reconciliation recognizes matching server completion and shows it without a second POST; unresolved outcome permits only read-only checking.
7. Reload/back resumes owner/task/revision-bound word/question/feedback/summary and pending selection. Account/task/content changes never reuse the wrong answers; incompatible drafts are discarded, stable-owner token refresh preserves draft, success clears it.
8. Trusted cached generated/legacy content supports offline study/feedback and disabled save; outline/ambiguous cache cannot masquerade as complete content. Invalid generated content needs explicit bounded “Tạo lại nội dung”, and completed saved results cannot be regenerated.
9. Existing timer anchors, elapsed clamp, study-time semantics, glossary focus return, keyboard-scroll frames, read-aloud slot and late-response guards remain intact. Storage denial does not block study or server save.
10. At 390×844 and desktop, English text remains body type, all answer/action targets meet kit size/focus/contrast rules, feedback is announced once, and reduced motion uses immediate static changes. No appwide restyle, new kit token or dependency is required.

