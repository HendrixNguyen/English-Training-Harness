---
type: feature
status: proposed
source: ideator
run: 2026-09-27-run-01
order: 5
---
# Region clear: an end-of-week recap on the hub with days met, minutes, words and the plant's stage

## Why
The roadmap is four 7-day modules, and the retro world map already draws them as four *regions* whose fog lifts when the last day is cleared (`harness/designs/retro-roadmap.md`). But nothing ever tells the learner "you finished a region": day 8 opens exactly like day 7. The week boundary is precisely where D7 retention — a named success metric in `docs/PRODUCT.md` — is decided, and the numbers the owner wants to watch (target hit rate, streak, words) are the numbers a learner would most like to be shown about themselves.

Weekly progress reports are the most-copied retention mechanic in language apps (Duolingo's weekly report is the reference), and celebration-of-milestone moments measurably lift long-term retention when they are tied to real persisted progress rather than confetti. Every fact needed already exists or is one field away: per-day minutes and `is_target_met` from `GET /api/v1/roadmap` (roadmap-tree plan, done/unmerged), the plant's stage and streak from `/pet/status`, and the module's own title and focus from the roadmap JSON. The growth moment (done/unmerged) celebrates a *day*; this celebrates a *week*, on the hub, once.

## Expected output
User-visible:
- On the first hub open on or after day 8, 15, 22 and 29 (region 1–4 finished by the calendar, whether or not every day was met), a `RetroPanel` card above the quests, the companion as speaker: "**Vùng 1 · Everyday small talk — đã xong!**" with four tiles: *ngày đạt mục tiêu* n/7, *phút học* total, *từ đã gặp* (words in the completed vocabulary tasks of the region), *chuỗi* current streak; a line for the plant "{plant_name}: mầm → cây non" when the stage changed during the week; an honest line when the week was weak ("Cậu đạt 2/7 ngày. Vùng 2 bắt đầu hôm nay — 30 phút thôi.") — never shaming, always pointing at today. Actions: **Xem bản đồ** → `/roadmap` (the freshly un-fogged region), and dismiss. Shown once per region per roadmap; dismissed stays dismissed (`localStorage['aelp.recap']` keyed by `roadmap_id` + region, through the existing `storageOrNull()` guard).
- On day 29 the card is the roadmap's summary (28 days) and hands over to the day-28 checkpoint's own card when that ships — information only; this idea does not depend on it.
- No push in this idea (the selected adaptive-reminder idea owns push copy); the card is in-app only.

Technical (backend `quests` small additive read; frontend hub):
- `GET /api/v1/roadmap` days gain additive `words_seen` (count of `content_json->'content'->'words'` for completed `vocabulary` exercises that day — one `jsonb_array_length` in the existing per-day query; 0 for legacy content) and modules gain `words_seen` summed. The plant-stage transition needs no backend: the card compares `pet.status.stage` with the stage recorded in the recap key when the previous region's card was shown (first region: stage at onboarding is always `sprout`).
- Frontend: `composables/useRegionRecap.ts` decides visibility from `roadmap.day_number`, the region boundaries and the storage key; `components/hub/RegionRecap.vue` renders the panel; `pages/index.vue` mounts it after the growth-moment timeline is `done` (never during it, never on `/learn` or `/revive`). Copy and layout from the designer role inside `harness/designs/retro-hub.md`. Reduced motion: no animation beyond the kit's stepped fade.
- Tests: backend — `words_seen` for a day with one completed vocabulary task of 6 words, a legacy raw task (0), an incomplete task (0); integration read-back. Frontend — boundary table (day 7 no, day 8 yes, day 8 again after dismiss no, new `roadmap_id` yes), the weak-week copy branch, stage-change line present/absent, absence while a growth delta is pending.
- Estimate: about half a working day frontend + ~2 h backend. Depends on the roadmap-tree plan's endpoint (done, unmerged) — the evaluator should sequence it after that lands.

## Evidence
- `harness/designs/retro-roadmap.md` — four regions, fog lifts when a region clears; `harness/plans/2026-09-25-roadmap-tree-…` — `GET /api/v1/roadmap` day/module shape (`minutes_spent`, `is_target_met`, module `title`/`focus`).
- `harness/plans/2026-09-25-growth-moment-…` — the per-day celebration and its `done` state this card waits for; `frontend/pages/index.vue` — the hub today (plant, health bar, bubble, day bar, quests) has no week boundary.
- `docs/PRODUCT.md` "What success looks like" (daily target hit rate, D7/D30, streak, roadmap completion); 1st-thinking §1 (≥ 30 min/day retention), §6.1 (4 modules × 7 days).
- Prior run 2026-09-26 notes: a 7-day met/missed *strip* was dropped as a second view of roadmap-tree data; this is a once-per-week moment, not a second view.
- Duolingo-style progress reports and why they retain: https://trophy.so/blog/how-to-create-duolingo-style-progress-reports-for-your-app ; celebrating user milestones tied to real progress: https://www.appcues.com/blog/celebrate-user-success-improve-retention
