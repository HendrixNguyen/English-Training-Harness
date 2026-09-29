---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# Already-onboarded learners now see the English default 'My Green Buddy' as the plant's name on the hub, banners and bubble

## Why
The branch now **speaks** the plant's name in five places — hub caption, hub wilted banner, `/revive` alarm, and two bubble lines (`frontend/pages/index.vue`, `pages/revive.vue`, `utils/plant.ts`). Only learners who onboard after this ships get "Mầm Non" or their own name. Everyone whose pet row already exists keeps the DDL default: `pet_states.plant_name` was always `'My Green Buddy'` (`backend/internal/pet/repo.go:82` `COALESCE(p.plant_name, 'My Green Buddy')`; `ensureSQL` inserts `(user_id)` only), and the re-submit path deliberately never renames. So every existing learner's Vietnamese hub now reads "My Green Buddy" as a heading above the plant, "⚠️ My Green Buddy đang bị héo rũ!" and "Cảm ơn bạn, hôm nay My Green Buddy đủ nước rồi 🌿" — the localisation wart the idea set out to remove, made more prominent, with no rename UI (`/settings` rename is a separate follow-up). The plan covered pets created before onboarding by the same learner, not learners already onboarded.

## Expected output
Pick one: (a) the client treats the DDL default `'My Green Buddy'` as "unnamed" (no caption, "Cây xanh" / "tớ" fallbacks as design §3 already specifies for a missing name); or (b) a data migration renames existing `'My Green Buddy'` rows to `'Mầm Non'` (DDL default untouched, so spec §3.2 still equals `0001_init`). Either way an already-onboarded learner never sees the English default on the hub. Test for the chosen path.

## Evidence
- Plan: `harness/plans/2026-09-25-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md` (notes: "Pets created before onboarding …", "Re-submit stays a no-op"); design `harness/designs/plant-name.md` §2/§3; branch head `55650d5`.
- Browser check (reviewer rv-frontend) on the branch build: with `plant_name` from `GET /pet/status`, the hub renders the name as a caption, in the banner and in the low-health bubble — whatever string the row holds.
