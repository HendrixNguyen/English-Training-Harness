# Ideation run 2026-09-24-run-01

**Mode:** features
**Read:** `AGENTS.md`; `.agents/roles/ideator.md`; `harness/CODEMAP.md` (all packages); `harness/STATE.md`; prior runs `2026-09-22-run-02/_run.md`, `2026-09-22-run-01/_run.md`; Frontend spec §4–§5 (API→UI table, settings row); `frontend/pages/settings.vue`, `frontend/pages/learn/[id].vue`, `frontend/utils/content.ts`; `backend/internal/airouter/prompt.go`, `roadmap.go`; `backend/internal/quests/day.go`. The worktree was 20 commits behind `origin/main` and was synced (`git merge origin/main`) before reading. Web research on reminder retention, corrective feedback and progress visibility (URLs are in each idea's Evidence). Memory skill not queried; prior runs recorded that it holds nothing about this project.
**Inbox noted:** 111 inbox files, nearly all reviewer bugs. The ones deliberately not duplicated and cited instead:
- `nothing-tells-the-pwa-to-flatten-pushsubscription-tojson-so-.md` — the settings idea must honour it.
- `two-concurrent-assessment-submits-double-spend-the-ai-and-or.md` — the day-28 idea has the same race.
- `route-returns-upstream-provider-error-bodies-verbatim-to-its.md` and `route-has-no-overall-deadline-so-one-call-can-take-90-second.md` — the writing idea makes both user-visible.
- `the-api-sends-no-cors-headers-so-the-deployed-pwa-on-its-own.md` (high) — gates every frontend idea in production. It stays a bug and is not re-proposed.

## Proposed
- `harness/ideas/2026-09-24-run-01/settings-screen-wires-web-push-reminders-and-google-calendar.md` — Settings screen wires Web Push reminders and Google Calendar sync to the shipped backend
- `harness/ideas/2026-09-24-run-01/typed-task-content-with-answer-keys-so-every-quest-renders-a.md` — Typed task content with answer keys so every quest renders and gives instant feedback
- `harness/ideas/2026-09-24-run-01/day-28-checkpoint-cefr-re-assessment-and-the-next-roadmap.md` — Day-28 checkpoint: CEFR re-assessment and the next roadmap
- `harness/ideas/2026-09-24-run-01/ai-graded-writing-practice-through-the-essay-grading-route.md` — AI-graded writing practice through the essay_grading route

## Notes
- Theme of this run: the MVP is merged, but three user-facing promises are unmet.
  - Retention levers are shipped server-side with no UI (idea 1).
  - The daily content has no learning signal (ideas 2 and 4).
  - Progression stops at day 28 (idea 3).
  - Idea 1 needs no backend change, so it is the cheapest.
- Dependencies (information only; ranking is the evaluator's job):
  - Ideas 2 and 4 both touch `RoadmapSchema`. Either can go first.
  - Idea 3 benefits from idea 2's scores but does not require them.
  - Every frontend idea is inert in production until the CORS inbox bug is fixed.
- Not re-proposed: run-01's *streak shield*, *spaced repetition* and *adaptive reminder timing*. All three are still `proposed` or `planned` in `2026-09-22-run-01`, so they are already awaiting the evaluator.
- Considered and dropped:
  - Speaking/pronunciation practice: needs audio capture, STT and a new provider type. Too large before idea 4 proves the grading loop.
  - Leaderboards/social: no social surface in any spec.
  - Two-way Calendar sync: contradicts the spec's one-way design (§5.1).
  - Offline progress queueing: the PWA already blocks completion offline with a clear message. Lower impact than the ideas above.
  - A "progress dashboard" (minutes per week, CEFR history): folded into idea 3's roadmap history rather than filed as its own thin idea.
