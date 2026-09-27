# Design: Retro revive — the companion is down (`/revive`)

**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Inherits:** `harness/designs/frontend-shell.md` §2.6 — the revive contract (`POST /api/v1/pet/revive` → `{revival_passed, pet_state}`; the challenge is "15 minutes of study on today's counter after it starts"), the `notWilted` and unknown-state rules, `plant-name.md` (the name in the alarm).
**Kit:** `harness/UI-KIT.md` v2.
**Spec wireframe:** frontend spec §7.5. Kept: alarm band, the wilted companion, the "you missed n days" line, one big button, 15-minute quiz framing. Departures: the challenge progress is drawn as a single 15-minute HP-style segment (shell §2.6 already did this); "quiz" becomes "nhiệm vụ hồi sinh" because the challenge is real study, not a separate quiz.

## 0. Research
- **Learner's job:** understand that the companion is down because they missed days, and bring it back with fifteen minutes of real study.
- **The moment that earns the next minute:** the companion getting up — `down` frame → `levelup` flash → `sprout` at HP 50 — after the bar fills. The screen must make that outcome visible before the learner starts.
- **What today does wrong:** `pages/revive.vue` is a red banner, a rotated SVG and two stacked buttons ("Vào học ngay" and "Kiểm tra hồi sinh") whose relation is unclear; nothing shows what "revived" will look like.
- **Open questions, answered:** (a) *Two buttons or one?* One primary that changes label with the state ("Hồi sinh (15 phút)" → "Vào học ngay" → "Hồi sinh"); the secondary "Kiểm tra" of today is folded into the primary once the bar is full. (b) *Is the day's normal path still available?* Yes — the hub stays reachable and the 15 minutes are the same minutes as the day target (shell §2.6).

## 1. The signature
**KO screen, not an error page.** The alarm band is the only ember thing; under it the companion lies on its side in the `down` frame inside a panel whose outer line is ember, HP bar at `HP 0/100`. The 15-minute segment sits *between* the companion and the button so the eye reads: down → fill this → up. When `revival_passed` returns, the panel line turns growth, the sprite plays the get-up frames and the bar re-labels to `HP 50/100`.

## 2. Flow
Entry: hub wilted band/button; a push notification's URL; direct load. Exits: primary "Vào học ngay" → `/` (quests); "Về trại" after revival → `/` with the growth-moment before/after (0 → 50, wilted → sprout). Not wilted on load → the healthy state with "Về trại".

## 3. Layout (mobile-first, `max-w-md`)
```
│ ▌ MẦM NON ĐÃ GỤC ▐                       │ 1 alarm band: RetroPanel tone=ember, VT323 22, full width
│ ┌ Mầm Non ────────────────────────┐      │ 2 companion panel tone=ember
│ │      [down sprite 128]           │      │   CompanionSprite react=down (lying), no idle
│ │      HP 0/100 [░░░░░░░░░░░░]     │      │   HpBar ember at 0
│ │ "Cậu bỏ tớ 2 ngày liền. Học 15   │      │   SpeechBox (font-body)
│ │  phút để tớ đứng dậy nhé."       │      │
│ └──────────────────────────────────┘      │
│ NHIỆM VỤ HỒI SINH            04/15       │ 3 eyebrow + DayBar segments=1 segmentSeconds=900 (VT323 20 counter)
│ [████░░░░░░░░░░░░░░░░░░░░░░░░]           │   growth cells; caption "Còn 11 phút học nữa"
├──────────────────────────────────────────┤
│ [       HỒI SINH (15 PHÚT)        ]      │ 4 bottom bar: RetroButton danger → primary as state changes
```
Data: `pet.status` (`plant_name`, `health_points`, `stage`, `last_practiced_at` → `daysSince`), `pet.challenge` (local, dated), `quest.accumulatedSeconds` → `challengeProgress`, `REVIVE_SECONDS`.
- **State A — wilted, no challenge:** bar hidden; button danger "Hồi sinh (15 phút)" → `pet.revive()`.
- **State B — challenge running:** bar visible; button primary "Vào học ngay" → `/`; when `canCheck` (≥ 900 s) the label becomes "Hồi sinh" and taps `pet.revive()` again.
- **State C — passed:** panel `tone=growth`, sprite get-up (`levelup` frames, 300 ms) → `sprout`, `HP 50/100`, line "Tớ dậy rồi. Cảm ơn cậu."; button primary "Về trại".
Desktop: same column.

## 4. States
- **Loading:** band hidden (never alarm before knowing); panel with `StateBlock loading`; no button.
- **Not wilted (`409 pet_not_wilted` or `health_points > 0`):** no band; panel `tone=growth` with the current sprite, "{plant_name} vẫn khoẻ." + "Về trại".
- **Unknown (load failed, nothing cached):** ember `StateBlock` "Không tải được trạng thái của {plant_name}. Chưa biết bạn ấy có gục không." + "Thử lại". Never guess "down".
- **Error on revive:** ember strip above the button "Chưa bắt đầu được nhiệm vụ hồi sinh. Thử lại." with the button restored.
- **Offline:** State B keeps counting from the store; the check waits — toast "Cần mạng để hồi sinh. Tớ vẫn đếm phút cho cậu."
- **First-time:** n/a (a wilted companion is never first-time).
- **Done:** State C; "Về trại".
- **Reduced motion:** the get-up is one frame swap; bar snaps; no cell sweep.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| "Hồi sinh (15 phút)" | drop → loading dots | `POST /pet/revive`; `revival_passed:false` → State B (challenge stored with today's date) | `pet.challenge` (local) |
| "Vào học ngay" | drop | `/` | — |
| "Hồi sinh" (bar full) | drop → loading | `POST /pet/revive`; `revival_passed:true` → State C; pet store gets `pet_state` | server |
| "Về trại" | drop | `/` with before/after for the growth moment | — |
| "Thử lại" | drop | `pet.load()` | — |
The bar updates when the hub/room posts progress (store), so returning from a task shows the new cells at once.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel` | kit | `tone`, `speaker` | band = a panel with no padding beyond 8 px and VT323 text |
| `CompanionSprite` | kit | `react=down`, then `levelup` | the `down` frame is drawn once (kit) |
| `HpBar`, `DayBar` | kit | `segments=1`, `segmentSeconds=900` | caption "Còn {n} phút học nữa" |
| `SpeechBox`, `RetroButton`, `StateBlock`, `RetroToast` | kit | — | — |
**Kit additions:** none. The band is a `RetroPanel` variant `band` (full-bleed, 8-px padding) — proposed as a prop, not a component.

## 7. Copy
Band: "{plant_name} đã gục". Lines: "Cậu bỏ tớ {n} ngày liền. Học 15 phút để tớ đứng dậy nhé." · (days unknown) "Cậu bỏ tớ lâu quá. Học 15 phút để tớ đứng dậy nhé." · passed "Tớ dậy rồi. Cảm ơn cậu." · healthy "{plant_name} vẫn khoẻ." Eyebrow "Nhiệm vụ hồi sinh"; counter "{mm}/15"; caption "Còn {n} phút học nữa" · full "Đủ 15 phút rồi". Buttons: "Hồi sinh (15 phút)" · "Vào học ngay" · "Hồi sinh" · "Về trại" · "Thử lại". Errors: "Chưa bắt đầu được nhiệm vụ hồi sinh. Thử lại." · "Không tải được trạng thái của {plant_name}. Chưa biết bạn ấy có gục không." Offline: "Cần mạng để hồi sinh. Tớ vẫn đếm phút cho cậu."

## 8. Self-critique
- Traded away: the explicit "Kiểm tra hồi sinh" button. Folding it into the primary means a learner with a full bar must notice the label changed; the caption "Đủ 15 phút rồi" and the button turning primary carry it.
- "Đã gục" (KO) is stronger than "héo rũ"; it matches the JRPG frame but must never shame — the line blames the days, not the learner.
- Executor traps: keep the `notWilted`/unknown branches exactly (never alarm on a failed load); the challenge date check (`pet.challenge.date === today`) stays; the growth-moment values on "Về trại" are 0/wilted → 50/sprout.
- Review checks: band absent while loading and for a healthy companion; the second `revive` call happens only at ≥ 900 s; the bar reflects minutes recorded after the challenge started, not the whole day.

## Acceptance
1. A wilted companion shows the ember band, the `down` sprite, `HP 0/100` and one danger button; a healthy one shows none of these.
2. Starting the challenge stores it for today and reveals the 15-minute bar with the primary "Vào học ngay".
3. The bar fills in cells from study recorded after the start; at ≥ 15 minutes the button reads "Hồi sinh" and calls revive again.
4. `revival_passed: true` turns the panel growth, plays the get-up once, shows `HP 50/100` and "Về trại".
5. A failed status load shows the unknown state with retry and no alarm.
6. Revive errors keep the button and show the ember strip.
7. Offline keeps counting and shows one toast; nothing posts.
8. Reduced motion: frame swaps only.
