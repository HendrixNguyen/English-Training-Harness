# Ideation run 2026-09-25-run-01

**Mode:** features
**Read:** `AGENTS.md`; `.agents/roles/ideator.md`; `README.md`; `harness/CODEMAP.md` (every package); 1st-thinking §1, §5, §7; backend spec §6, §7, §8; frontend spec §5, §7 (all five wireframes); prior runs `2026-09-24-run-01/_run.md` (+ its four ideas' Expected output) and `2026-09-22-run-02/_run.md`, plus the three still-`selected` features in `2026-09-22-run-01` (streak shield, spaced repetition, adaptive reminder) to avoid overlap; frontmatter of every plan and review dated 2026-09-22..24 and the findings sections of the 2026-09-24 reviews; the inbox titles (127 files). Code read to ground the ideas: `frontend/pages/{index,learn/[id],roadmap,revive,onboarding}.vue`, `frontend/stores/{quest,pet,auth}.ts`, `frontend/utils/{plant,roadmap,content}.ts`, `frontend/components/learn/CountdownTimer.vue`, `backend/internal/auth/{token,scopes}.go`, `backend/internal/store/keys.go`, `backend/internal/pet/repo.go`, `0001_init.up.sql`. Web research per idea (URLs under each `## Evidence`). Memory skill not queried (prior runs recorded it holds nothing for this project; unattended run).
**Inbox noted:** all inbox files are reviewer bugs; none were moved or edited. Deliberately not duplicated:
- `signing-in-on-a-second-device-silently-logs-the-first-one-ou.md` — the stay-signed-in idea keeps the single-session design and cites it.
- `the-api-state-cache-still-has-no-expiry-bound-so-a-stale-res.md` — the roadmap idea adds `/api/v1/roadmap` to the same NetworkFirst cache and inherits whatever bound that bug settles on.
- `the-revive-missed-days-line-renders-bo-hoc2-ngay-with-no-spa.md` (planned) — the plant-name idea touches `/revive` copy but does not re-file this.
- `a-re-submitted-assessment-silently-discards-target-goal-noti.md` — the plant-name idea explicitly keeps the re-submit path a no-op and leaves that bug to its own plan.

## Proposed
- `harness/ideas/2026-09-25-run-01/task-timer-keeps-counting-through-reloads-and-background-tab.md` — Task timer keeps counting through reloads and background tabs so studied minutes are never lost (frontend; ~half day)
- `harness/ideas/2026-09-25-run-01/growth-moment-after-every-task-health-gain-streak-and-target.md` — Growth moment after every task: health gain, streak and target-met celebration on the hub (frontend; ~1 day)
- `harness/ideas/2026-09-25-run-01/roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md` — Roadmap tree shows the real plan: module and day titles with true per-day completion (backend read + frontend; ~1 day)
- `harness/ideas/2026-09-25-run-01/stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md` — Stay signed in: sessions renew on use so a daily learner never sees the Google consent screen again (backend auth/middleware + frontend; ~1 day)
- `harness/ideas/2026-09-25-run-01/name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md` — Name your plant at onboarding and see it greet you by name on the hub (backend onboarding additive field + frontend; ~half day)

## Notes
- Theme: the nine MVP slices are merged and the last run covered content (typed tasks, writing), settings and day-28. This run looked at the **daily loop as a returning learner experiences it on day 2** and found four silent gaps: the countdown forgets time on reload/background (idea 1), the plant grows without any moment (idea 2, §5.2 step 5 is simply not implemented), the roadmap awards stars to skipped days and names nothing (idea 3), and every learner is put through the Google consent screen once a day because sessions are a hard 24 h (idea 4). Idea 5 is a small ownership hook that also removes an English default name from the Vietnamese UI.
- Dependencies (information only): idea 2 is what the pending *streak shield* idea (2026-09-22-run-01) assumes exists ("completion animation announces Shield earned"). Idea 3's outline read is what the selected *day-28 checkpoint* idea can extend for roadmap history. Ideas 1 and 2 touch `stores/quest.ts`/`stores/pet.ts` and should not run in parallel worktrees. Idea 4 touches `middleware` CORS headers, which the approved settings plan does not.
- Sizing: ideas 1 and 5 are half-day; 2, 3 and 4 are a working day each. All five stay inside one execute run.
- Considered and dropped:
  - Personalised reminder body ("còn 12 phút nữa") — overlaps the selected *adaptive reminder timing and pre-decay rescue push* idea, which already specifies a rescue push with exact minutes remaining.
  - In-app "cây sẽ mất 30 máu lúc nửa đêm" countdown on the hub — same overlap; folded partially into idea 2's context-aware speech bubble instead.
  - Offline progress queueing — still lower impact than the above; idea 1 removes the worst symptom (lost minutes) without needing a queue.
  - Plant rename in settings — deferred until the approved settings plan lands; idea 5 covers first naming only.
  - Larger placement quiz bank (ten fixed items) — real but not something a learner hits daily.
