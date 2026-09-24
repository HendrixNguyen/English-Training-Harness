---
type: feature
status: proposed
source: ideator
run: 2026-09-25-run-01
---
# Task timer keeps counting through reloads and background tabs so studied minutes are never lost

## Why
The 30-minute target (1st-thinking §1, §5.2) is credited from the duration the client reports on `POST /quests/progress`, and the client derives that duration from a countdown it runs itself. Today `stores/quest.ts` keeps the countdown in memory and advances it one tick per `setInterval` firing (`startTimer`/`tick`, `learn/[id].vue` `setInterval(..., 1000)`). Two everyday behaviours break it:

- **Reload, tab close, or the PWA being evicted from memory on a phone** wipes `timers` — the learner comes back to a fresh 10:00 countdown and the minutes already spent are gone. On iOS an installed PWA is suspended as soon as the learner switches apps, which is exactly what someone doing a reading task with a dictionary open does.
- **Background throttling.** Chrome 88+ runs timers of a page hidden for more than five minutes once per minute, Firefox clamps to one second or more; the tick-based countdown falls behind wall-clock time, so ten real minutes of study is recorded as four or five. The learner then has to wait out a countdown that lies, or gives up. Both cases are a direct tax on the one metric the product is judged by (≥ 30 min/day) and on the plant: under-counted minutes mean a −30 health penalty at midnight for a day that was actually studied.

This is the cheapest retention fix available: no backend change, one store and one page. The revive challenge (`/revive`, `aelp.revive`) already persists its anchor in `localStorage` — the daily timer should get the same treatment.

## Expected output
User-visible:
- Leaving a task (back button, reload, switching apps, closing the tab) and coming back within the same local day shows the countdown where wall-clock time says it should be, not where the last tick left it. Ten minutes away means the timer reads 00:00 and the button says "Hết giờ — Hoàn thành".
- A task's countdown is anchored to the moment the learner first opened it. The "Hoàn thành" button becomes available at exactly the task's `duration_minutes` of real time (or on `finished`, unchanged).
- A timer started yesterday (different `quest.daily.date`) is discarded, never resumed, so a learner cannot carry an old anchor into a new day's exercise.
- Nothing changes on the wire: `duration_seconds` is still `clampDuration(elapsed)` in 1..3600 and the server's `MaxDurationSeconds` guard still applies.

Technical:
- `stores/quest.ts`: `Timer` becomes `{ startedAt: epoch-ms, totalSeconds, date }`; `remainingSeconds` and `elapsedSeconds` are computed from `Date.now()` (a `now` argument for tests) — `tick` only triggers re-render. Persist the timer map at `localStorage['aelp.timers']` on start and remove entries on `complete()`; hydrate on store creation and drop entries whose `date` is not today's `daily.date` or whose task is already `is_completed`.
- `learn/[id].vue`: recompute on `visibilitychange` and `focus` so a resumed tab snaps to the correct value immediately, not at the next throttled tick.
- Unit tests (Vitest, existing `tests/unit` conventions): elapsed equals wall-clock delta regardless of tick count; hydrate after simulated reload; stale-date entry discarded; completed task entry discarded; `elapsed` never exceeds `totalSeconds` for the button gate but `duration_seconds` posted is the real elapsed clamped to 3600.
- CODEMAP `shell` paragraph updated (the "re-entry resumes" sentence is currently true only within one page lifetime).

## Evidence
- 1st-thinking §5.2 step 1 (`Complete Task (Duration: 10m)`), backend spec §6.2 (`duration_seconds` is client-reported), §8 inactivity logic (under-counted minutes → −30 at midnight).
- Frontend spec §7.3 wireframe: the learning room shows a running `⏱️ Thời gian: 09:42` — a clock the learner trusts.
- Code: `frontend/stores/quest.ts` (`timers`, `startTimer`, `tick`, `elapsedSeconds` — in-memory, tick-based), `frontend/pages/learn/[id].vue` (`setInterval(..., 1000)`, comment "the store keeps remainingSeconds, so re-entry resumes"), `frontend/stores/pet.ts` (`REVIVE_STORAGE_KEY` — the persisted-anchor pattern to copy).
- CODEMAP `shell`: the store timers are described as per-task countdowns with no mention of persistence.
- Chrome intensive throttling of hidden-page timers and the wall-clock fix: https://dev.to/work_hau_cb718f47075930f9/javascript-countdown-timers-why-setinterval-drifts-and-how-to-fix-it-26fe ; Firefox clamping breaking countdown scripts: https://bugzilla.mozilla.org/show_bug.cgi?id=652472 ; background-tab polling drift: https://dev.to/phpner/the-background-tab-bug-i-missed-in-my-javascript-polling-code-3ol3
