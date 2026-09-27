---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: medium
---
# The shield earn line shows every day the streak sits on a multiple of 7 including the next morning before practice

## Why
`speechLine` (`frontend/utils/plant.ts`) shows the earn line "Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️" whenever `streak > 0 && streak % 7 === 0 && shields > 0`, and it outranks every other line except wilted. Design §4.2 assumes "Tomorrow the streak is 8 (or 15, 22) and the ordinary lines return", but `current_streak` only moves when a day's target is met (`saveTargetMetSQL` streak+1) or an unshielded miss resets it. So:
- **Every learner, the morning after their 7th/14th/21st day:** streak is still 7 and a shield is held, today's target is not met, and the bubble congratulates them on the milestone instead of the study nudge ("Tưới cho tớ 10 phút học đi!") — for the whole day until they finish 30 minutes. The nudge is the hub's one call to action; this hides it on exactly the day after a milestone.
- **After a shielded miss at a multiple of 7 with a full rack** (day 14/21 with 2 shields, then a miss): the sweep spends one shield and keeps streak 14, so the next morning shows "Khiên đã đỡ cho ngày …" (a day was missed) and "Tròn 7 ngày liên tiếp!" in the same card — contradictory, at the moment the design says matters most.
Reproduced on the running hub (see Evidence).

## Expected output
The earn line shows only on the day the milestone was actually reached: add `o.targetMet` to the condition (on the earn day today's 30 minutes are met by definition; the next morning they are not), i.e. `if (o.targetMet && streak > 0 && streak % 7 === 0 && shields > 0)`. `plant.test.ts` gains the two cases above (`targetMet: false, streak: 7, shields: 1` → the health-threshold nudge; `streak: 14, shields: 1, targetMet: false` after a spend → not the earn line). Design §4.2's "Tomorrow the streak is 8" sentence corrected.

## Evidence
- Plan: `harness/plans/2026-09-25-pet-streak-shield-earned-by-target-days.md` (decision 6; Task 4 Step 3). Design: `harness/designs/pet-streak-shield.md` §4.2.
- Branch `harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days` @ `55d303e`: `frontend/utils/plant.ts` `speechLine` (earn condition), `frontend/pages/index.vue:15` (passes `targetMet: quest.targetMet` but the earn branch ignores it).
- Reviewer screen walk 2026-09-27 (API + `nuxi dev` against a scratch stack, mobile 375×812): pet row `current_streak=7, shields=1`, no progress today → header "Streak: 7 ngày", rack held+empty, bubble "Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️" with today's target not met.
