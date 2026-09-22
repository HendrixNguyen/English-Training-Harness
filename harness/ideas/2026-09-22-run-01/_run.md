# Ideation run 2026-09-22-run-01

**Mode:** features
**Read:** `harness/CODEMAP.md`; spec `1st-thinking-architecture-doc.md` §1, §3.1, §4, §5.1–5.2, §6.1 (no §7 exists — the spec ends at §6); `harness/STATE.md`; `harness/runs/20260922T152444.log`. No prior `_run.md` files exist (first ideation run).
**Inbox swept:** none — `harness/ideas/_inbox/` contained only `.gitkeep`.

## Proposed
- `harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md` — Pet Streak Shield Earned by Target Days
- `harness/ideas/2026-09-22-run-01/spaced-repetition-vocabulary-review-in-the-daily-quest.md` — Spaced Repetition Vocabulary Review in the Daily Quest
- `harness/ideas/2026-09-22-run-01/adaptive-reminder-timing-and-pre-decay-rescue-push.md` — Adaptive Reminder Timing and Pre-Decay Rescue Push

## Notes
- Coverage: idea 1 targets the pet loop / loss-aversion churn point, idea 2 targets CEFR progression (the only idea that adds a learning-outcome signal), idea 3 targets the ≥30 min/day target via the notify package. All three sit on planned packages (pet, quests, notify, store) and none needs an airouter call on the hot path.
- Considered and dropped: CEFR re-assessment at the end of module 4 (strong, but depends on onboarding + airouter landing first; re-propose next run); offline quest caching in the PWA shell (infra more than feature; better as part of the frontend-shell MVP slice); two-way Google Calendar sync (spec §5.1 is explicitly one-way — would contradict the spec); leaderboards/social (no social surface in the spec).
- App code does not exist yet (CODEMAP: all packages "none exist yet"), so every idea's Expected output is written against the spec's tables and endpoints, not existing code. The evaluator may want to sequence these after the MVP slices.
- Memory lookup: the episodic-memory search returned only conversations about an unrelated `claude-harness` repo; no prior decisions about this project were found.
- Research done via web search; URLs are under each idea's `## Evidence`.
- Question for the human (could not ask): should ideas that depend on unbuilt MVP packages be proposed at all in features mode, or should the harness run `--mvp` first? Proceeded on the assumption that features can be queued behind the MVP slices.
