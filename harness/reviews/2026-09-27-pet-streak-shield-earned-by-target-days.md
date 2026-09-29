---
plan: harness/plans/2026-09-25-pet-streak-shield-earned-by-target-days.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/the-shield-earn-line-shows-every-day-the-streak-sits-on-a-mu.md, harness/ideas/_inbox/two-done-branches-both-add-migration-0004-pet-shields-and-rl.md]
---
# Review — pet: a streak shield, earned every 7th met day, is spent in place of the miss penalty

**Plan:** `harness/plans/2026-09-25-pet-streak-shield-earned-by-target-days.md`
**Branch/worktree:** `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` / `.worktrees/pet-streak-shield-earned-by-target-days`
**Diff:** `git diff main...harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days --stat`

## Plan vs idea
Delivered. Every 7th consecutive met day awards a shield (cap 2); a missed day with a shield held spends it (no health loss, streak kept, no revive, `judged_through` still advances); with no shield the §8 path is unchanged. Migration `0004_pet_shields` matches the idea's DDL, `GET /pet/status` carries `shields` and `last_shield_used_on`, and the hub draws the rack and the seven-day spend caption. The idea's "completion animation announces Shield earned" became the bubble line by design (§6 self-critique) — accepted. The bubble line is where the one real defect sits (bug 1): it shows on the day after the milestone as well, not only on the earn day.

## Code vs plan
Reviewed at origin head `55d303e` in a detached scratch worktree (22 files, +435/-64). All six tasks followed as written, including the SQL folds (award inside `saveTargetMetSQL`, spend inside `penaliseMissSQL` with `RETURNING COALESCE(last_shield_used_on = $2::date, FALSE)`), the tri-return `PenaliseMiss`, `Sweep` counting only unshielded misses, and `Save` never writing the two columns. Deviations, all justified: `store/migrations_test.go` updated (hardcoded version list) plus a `TestMigration0004AddsPetShields` mirroring the 0003 test; the plan's `grep -c shield CODEMAP` ≥ 6 is unmeetable because CODEMAP is one line per paragraph (3 lines match, all three edits present); the executor flagged the `0004` collision with the Supabase RLS branch (bug 2).

Re-run evidence:
```
$ gh run list --branch <branch> --limit 1
completed success harness: CODEMAP — streak shield in pet, store and shell  CI ... 36227762960
$ env -u … make check              -> go vet silent; 13 packages ok under -race
$ env -u … go test ./internal/pet/ -count=1 -race -v | grep -c '^--- PASS'
48
Integration against scratch pg 5443 / redis 6393 (compose project rv-backend), -p 1:
--- PASS: TestIntegrationVerdictWritesAreConditional (0.16s)      (section 6: award/cap/spend/fallthrough)
--- PASS: TestIntegrationMigrateAppliesToAnEmptyDatabaseAndIsIdempotent (0.94s)
--- PASS: TestIntegrationConcurrentMigrateDoesNotRace, ...EnsureCreatesExactlyOnePetRow, ...DailyAndProgress..., (13 PASS, 0 FAIL)
$ diff <(spec "Added by migration 0004" block) <(0004_pet_shields.up.sql ALTER)   -> SPEC_DDL_SAME
$ npm ci && npm run lint && npm run typecheck && npm run test:unit
 ✓ plant.test.ts (7)  ✓ petStore.test.ts (7)  ✓ ShieldRow.test.ts (6)   Test Files 17 passed, Tests 91 passed
Boot: migrations applied: [0001_init 0002_google_sync 0003_pet_verdict_dates 0004_pet_shields]
GET /api/v1/pet/status (scratch user + minted session) -> 200
{"plant_name":"My Green Buddy","stage":"sprout","health_points":100,"current_streak":0,"last_practiced_at":null,"shields":0,"last_shield_used_on":null}
```

Design walk (`nuxi dev` against the booted API, mobile 375×812, pet row set directly in Postgres):
- §4.1 `shields=0, null` (light theme): two `mute/30` outlines, no caption, aria "Khiên: 0 trên 2". ✓
- §4.1 `shields=1, last_shield_used_on=2026-09-26` (dark): held amber + spent-with-tick, caption "Khiên đã đỡ cho ngày 26/09.", aria "Khiên: 1 trên 2, một chiếc vừa đỡ cho ngày 26/09", bubble the ordinary nudge. ✓
- §2 layout: row sits under `HealthBar`, label in the "Máu cây:" column, nothing at the right edge. ✓
- §4.2 earn line: `streak=7, shields=1`, today's target not met → bubble "Tròn 7 ngày liên tiếp! …" ✗ — the morning after the milestone still announces it (bug 1).
- §4.3: header pill, wilted banner, loading/error branches untouched. ✓ Motion: only `transition-colors … motion-reduce:transition-none`. ✓
- Kit: `harness/UI-KIT.md` is now v2 (retro), but its tokens are "proposed" and the live `tailwind.config.ts` is still v1; the design doc and the component use the live v1 tokens (`streak`, `mute`). Not counted as a kit violation for this plan; the v2 migration plan will need to restyle `ShieldRow` with the rest of the hub.

## Quality
- SQL/Go parity is enforced by the integration section 6 (ran green against real Postgres). `RETURNING` reasoning checked: `last_shield_used_on` is only ever set to a day that becomes `judged_through`, and the predicate needs `judged_through < $2`, so it can equal `$2` only when this write set it.
- Concurrency: award and spend live in the once-per-day conditional UPDATEs, so concurrent sweepers/progress calls cannot double-award or double-spend. `Save` staying off the columns avoids a revive erasing an award.
- Maintainability nit (not filed): `ShieldRow`'s aria label recovers the date with `caption.slice(-6, -1)` from the Vietnamese caption; a copy change silently breaks the screen-reader sentence. A `shieldSpentDate` helper (the plan offered it) would be sturdier.
- Merge: against current `main` only `harness/CODEMAP.md` conflicts. Against the Supabase RLS branch (also done, also `0004`) four files conflict (bug 2).

## Bugs filed
- `harness/ideas/_inbox/the-shield-earn-line-shows-every-day-the-streak-sits-on-a-mu.md` — medium.
- `harness/ideas/_inbox/two-done-branches-both-add-migration-0004-pet-shields-and-rl.md` — medium (merge coordination with `harness/2026-09-26-high-every-public-table-is-readable-and-writable-through-supabase`).

## Verdict
pass-with-bugs — backend mechanic, migration, wire and rack all verified live; the earn-line condition misfires the morning after a milestone (medium), and the `0004` number collides with the RLS branch at integration.
