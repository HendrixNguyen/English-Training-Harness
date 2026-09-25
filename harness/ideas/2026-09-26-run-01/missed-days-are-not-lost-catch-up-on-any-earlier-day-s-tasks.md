---
type: feature
status: proposed
source: ideator
run: 2026-09-26-run-01
order: 4
---

## Why
`day_number` is calendar days since the roadmap was created, so a learner who skips Tuesday and Wednesday opens the app on Thursday to day 4 — days 2 and 3, with their three tasks each, are gone for good. Their plant already paid (−30 twice); losing the lessons on top of it is a second punishment the spec never asks for, and it is the one a learner remembers: "I missed a day and the app threw my lesson away". The roadmap-tree plan (planned) will soon make this visible — a muted node with "0 phút" — with nothing to do about it. Letting the learner open any earlier day and do its tasks, with the minutes still counting toward *today's* 30, does three things at once: it makes a missed day recoverable (the same class of mechanic Duolingo's streak-repair quiz and "complete three lessons to restore" use, which lifted D7 retention in their tests), it gives a learner who finishes early something on-plan to keep going with rather than a locked screen, and it keeps the content pacing the roadmap prompt promised (4 modules × 7 days) honest even when life intervenes. It changes no plant math and no daily target: the day is still judged on 1800 s in the local date.

## Expected output
User-visible:
- On `/roadmap`, any day before today that is not target-met shows "Học bù" (a missed day) or "Học tiếp" (a partly done day). Tapping it opens the hub's quest list for that day — header "Ngày 3 · học bù", the three tasks with their real done/undone state — and each task opens the normal learning room.
- Minutes spent count toward today's target and today's plant verdict, exactly as if it were today's task; the segmented day bar on the hub moves. The earlier day's node on the roadmap shows its tasks ticked, but its star stays as the day was judged (a catch-up never rewrites history: no retroactive `is_target_met`, no streak change).
- Future days stay locked. Today's tasks remain the default; the catch-up view has one "Về hôm nay" exit.

Technical (backend `quests`, frontend roadmap + hub):
- `GET /api/v1/quests/daily?day=N` (additive query param; `1 ≤ N ≤ today's day_number`, else `400 invalid_request`) returns the same body for that day with `day_number: N` and an additive `is_today: false`; `date`, `accumulated_seconds` and `is_target_met` stay **today's** values so the client keeps one counter.
- `POST /api/v1/quests/progress`: `QuestRepo.CheckExercise` accepts an exercise whose `day_number ≤ today` instead of `= today` (still the caller's active roadmap; still `404 exercise_not_found` for a future day). Everything after the check — INCRBY on today's date, `daily_progress` upsert, `Pet.OnTargetMet`, `MarkComplete` — is untouched, so the ordering the service tests pin does not change.
- Frontend: `stores/quest.ts` gains `loadDay(n)` / `viewingDay` alongside `daily`; `pages/index.vue` renders the catch-up header and exit when `viewingDay !== null`; `pages/roadmap.vue` (after the roadmap-tree plan) adds the action to past nodes; the retro-roadmap design's world map is where the designer places it. Task timers already key on task id, so a catch-up task's timer persists like any other (task-timer plan).
- Specs: backend spec §6.2 gains the query param and `is_today`; 1st-thinking §5.2 unchanged. CODEMAP `quests` and `shell` updated. Tests: the day bound (1, today, today+1, non-integer), the relaxed `CheckExercise` (past ok, future 404, other roadmap 404), the integration test extended with one past-day exercise, and unit tests for the store's two-day state.
- Estimate: one working day. Depends on the roadmap-tree plan for the natural entry point (its outline read already carries per-day `minutes_spent`/`is_target_met`); without it, a "Học bù ngày N" row on the hub for yesterday alone would still deliver the value.

## Evidence
- Spec: 1st-thinking §5.2 (progress is per calendar day; the pet judges 1800 s per local date — unchanged here); backend spec §6.2 (`GET /quests/daily`, `POST /quests/progress` contracts); §8 (miss = −30 — the punishment that already happened); frontend spec §7.4 (roadmap nodes; "🔒 Chưa mở khóa" stays for future days only).
- Code: `backend/internal/quests/repo.go:57` ("day_number, or the call is rejected with ErrExerciseNotFound"); `service.go:94,227` (`DayNumber` computed once for both endpoints); CODEMAP `quests` (calendar-day `day_number`, clamp 1..28).
- Harness context: `harness/plans/2026-09-25-roadmap-tree-shows-the-real-plan-…` (adds `GET /api/v1/roadmap` with per-day `minutes_spent`/`is_target_met` — the read this idea's entry point uses); `harness/designs/retro-roadmap.md` (world map of 28 nodes); `harness/plans/2026-09-25-task-timer-…` (timers keyed by task id).
- Research: Duolingo's recovery mechanics (streak repair quiz, "complete three lessons to restore") and their A/B result — D7 retention +14% for a recovery offer — https://making.duolingo.com/how-streaks-keep-duolingo-learners-committed-to-their-language-goals and https://www.digia.tech/post/duolingo-habit-forming-reminders-retention-architecture/
