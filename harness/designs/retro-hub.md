# Design: Retro hub — the camp screen (`/`)

**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Inherits:** `harness/designs/frontend-shell.md` §2.3 (data binding, page states), `growth-moment.md` (the two seconds after a task), `pet-streak-shield.md` (shield rack), `plant-name.md` (the companion's name). Their wire contracts are unchanged; this doc restyles what they draw.
**Kit:** `harness/UI-KIT.md` v2 — every token, component and copy rule below comes from it.
**Spec wireframe:** frontend spec §7.2. Departures: the quest list becomes a vertical path of tiles (the wireframe's three rows, drawn as nodes); the streak chip moves into a status bar; the primary action becomes a bottom-bar button ("Vào nhiệm vụ") instead of a per-row button. Reasons under §0.

## 0. Research
- **Learner's job:** open the app, see how the companion is doing, and get into today's next quest in one tap.
- **The moment that earns the next minute:** the companion's dialogue line addressing the learner by what they did ("Hôm qua cậu xong cả phòng, tớ lớn thêm rồi") and the path showing that quest 2 of 3 is *right there*, with the day bar two-thirds lit.
- **What today does wrong:** three text rows with `[✓] [▶] [ ]` glyphs and small "Học" links (`quest/QuestRow.vue`) look like a to-do list; the primary action is a 36-px link mid-screen, not under the thumb; the plant card and the progress card are two separate white boxes, so the causal link "study → plant" is not drawn. Owner (2026-09-25): "not interesting… not attractive".
- **Open questions, answered:** (a) Which task does the bottom button open? `quest.nextTaskId`; when `is_target_met` and tasks remain, the first incomplete one; when all done, the button reads "Xem hành trình" → `/roadmap`. (b) Does the hub show XP? No — XP is a session-only number (see `retro-learning-room.md` §0); the hub shows the backend truths: HP, streak, minutes.

## 1. The signature
The **path lives under the companion**. The three quest tiles hang in a vertical chain from the companion's panel, the connector between them lit in `growth` up to the current tile, so the screen reads as one scene: companion at the camp, road ahead. When the learner finishes a task and returns, the connector to the next tile lights cell by cell (stepped, 300 ms) and the `▶` cursor moves down one tile — the growth-moment sequence plays on the companion at the same time. Nothing is a list.

## 2. Flow
Entry: after login/onboarding, after `/learn/:id` completes (with `growth-moment` before/after values), from the `/roadmap` and `/settings` back links, on PWA launch. Exits: bottom button → `/learn/:nextId`; any `open`/`current` tile → `/learn/:id`; status bar streak badge → `/roadmap`; avatar → `/settings`; wilted band → `/revive`. Returning from a task: the day bar, the tile states and the companion's HP/line update from the `POST /quests/progress` response (`daily_seconds_spent`, `is_target_met`, `pet_health`, `streak_count`).

## 3. Layout (mobile-first, `max-w-md`; desktop: same column centred on the dotted ground)
```
┌────────────────────────────────────────┐
│ [av] Hendrix        [🔥 x5]  [🛡][🛡]   │ 1 status bar: name VT323 22; Badge streak (torch, count VT323 20); shield rack
├────────────────────────────────────────┤
│ ┌ Mầm Non ─────────────────────────┐   │ 2 RetroPanel speaker=plant_name
│ │  [sprite 128]  HP 80/100          │   │   CompanionSprite stage/health; HpBar right of it
│ │                [████████░░░░]     │   │
│ │  "Tưới cho tớ 10 phút đi, cậu."   │   │   SpeechBox line (font-body 17)
│ └───────────────────────────────────┘   │
│ PHÒNG HÔM NAY                20/30      │ 3 eyebrow VT323 16 + DayBar (VT323 20 counter)
│ [████][████][░░░░]                      │
│   │                                     │ 4 path: connector 4 px, growth up to current, line-dim after
│ [📖] Từ vựng email công việc   Đã xong  │   QuestNode done (torch star)
│   │                                     │
│ ▶[📜] Đọc hiểu mẫu thư         Vào     │   QuestNode current (growth border, blinking cursor)
│   │                                     │
│ [🗡] Viết phản hồi khách hàng   Khoá    │   QuestNode locked (ink-2, padlock)
│                                         │
├────────────────────────────────────────┤
│ [        VÀO NHIỆM VỤ 2 →        ]      │ 5 bottom bar: RetroButton primary, sticky
└────────────────────────────────────────┘
```
(Emoji above stand for the kit's pixel icons: book, scroll, sword.)
1. **Status bar** — `full_name` from the auth store; `Badge kind=streak count=current_streak`; `Badge kind=shield` ×2 from `shields` (pet-streak-shield). Tapping the streak badge opens `/roadmap`.
2. **Companion panel** — `GET /pet/status`: `plant_name` as speaker, `stage` + `health_points` → sprite and `HpBar`; line from `speechLine()` restyled (§7). Wilted: `tone=ember`, the line is replaced by the revive band (§4).
3. **Day bar** — `GET /quests/daily`: `accumulated_seconds`, `total_minutes_required`, `is_target_met`.
4. **Path** — `tasks` in `task_type` order; state from `rowState()` as today. The connector is a 4-px column of cells, `growth` down to the current tile.
5. **Bottom bar** — label "Vào nhiệm vụ n →" for the current tile; "Xem hành trình" when every task is done; hidden while the quest region is loading.

## 4. States
- **Loading:** status bar from the stored user at once; panel 2 shows the sprout sprite in `ink-2` and `StateBlock loading`; the path shows three `ground-2` tiles; the bottom bar is empty (no button to mis-tap).
- **Empty (`404 no_active_roadmap`):** companion panel renders; the day bar and path are replaced by one `RetroPanel` "Cậu chưa có hành trình." + `RetroButton` "Bắt đầu hành trình 28 ngày" → `/onboarding`.
- **Error:** each region fails alone — a `tone=ember` panel "Không tải được nhiệm vụ. Thử lại." with a secondary button that refetches only that region.
- **Offline:** cached `pet/status` and `quests/daily` render as normal; `RetroToast` "Đang ngoại tuyến — tớ nhớ tiến độ giúp cậu." once; tiles stay tappable (the room works from the store).
- **First-time (day 1, no minutes):** line "Chào cậu. Tớ là Mầm Non — cùng đi phòng đầu tiên nhé?"; the first tile is `current`.
- **Done (`is_target_met`):** all cells `growth`, caption "Phòng hôm nay đã xong"; remaining tiles `open`; line "Hôm nay tớ đủ nước rồi, cảm ơn cậu."
- **Wilted (`health_points == 0`):** panel 2 `tone=ember`, sprite `down`, `HpBar` at 0 in ember; under it a danger `RetroButton` "Hồi sinh Mầm Non" → `/revive`; the path stays usable.
- **Growth moment (after a task):** exactly `growth-moment.md` §2 timeline with kit motion: HP cells step up, `Chest` chips ("+20 HP", "🔥 x7") rise 8 px in one step, the sprite plays `levelup` when the stage changed, else `hit`. ≤ 2 s.
- **Reduced motion:** cursor static, connector lights at once, sprite idle off, growth moment = final frame with the chips shown for 2 s.

## 5. Interactions and feedback
| Control | Tap → within 100 ms | Then |
|---|---|---|
| Bottom `RetroButton` | button drops 2 px | `navigateTo('/learn/' + nextTaskId)` |
| `QuestNode` open/current | tile insets, cursor jumps to it | same navigation; locked tiles do nothing and `aria-disabled` |
| Streak badge | badge insets | `/roadmap` |
| Avatar | — | `/settings` |
| Region retry | secondary button drop | `pet.load()` / `quest.load()` only |
Nothing on the hub writes to the server; state comes from the pet and quest stores.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel` | kit | `speaker`, `tone`, `portrait` slot | replaces `AppCard` |
| `CompanionSprite` | kit | `stage`, `health`, `react` | replaces `PlantSvg`; `react` driven by growth-moment |
| `SpeechBox` | kit | `line`, `name` | replaces `SpeechBubble`; typing skippable |
| `HpBar`, `DayBar` | kit | as kit | replace `HealthBar`, `SegmentedProgress` |
| `QuestNode` | kit | `task`, `index`, `state` | replaces `QuestRow`; emits `enter` |
| `Badge` | kit | `kind`, `count`, `earned` | streak + shields in the status bar |
| `RetroButton`, `RetroToast`, `StateBlock` | kit | — | — |
**Kit additions:** none. The connector is a `QuestNode` prop (`connector: lit|dim|none`), not a component.

## 7. Copy
Status bar: "🔥 x{n}" (badge, count only). Eyebrow: "Phòng hôm nay". Day caption met: "Phòng hôm nay đã xong". Tile actions: "Vào" · "Đã xong" · "Khoá". Bottom: "Vào nhiệm vụ {n} →" · "Xem hành trình".
Companion lines (replace `speechLine()`): HP ≥ 60 "Tưới cho tớ 10 phút đi, cậu." · 30–59 "Tớ hơi khát rồi… 10 phút thôi?" · 1–29 "Tớ sắp héo mất. Học một chút nhé?" · met "Hôm nay tớ đủ nước rồi, cảm ơn cậu." · first day "Chào cậu. Tớ là {plant_name} — cùng đi phòng đầu tiên nhé?" · wilted: none (the band speaks).
Empty: "Cậu chưa có hành trình." / "Bắt đầu hành trình 28 ngày". Errors: "Không tải được {plant_name}. Thử lại." · "Không tải được nhiệm vụ. Thử lại." Offline toast: "Đang ngoại tuyến — tớ nhớ tiến độ giúp cậu." Wilted button: "Hồi sinh {plant_name}".

## 8. Self-critique
- Traded away: the per-row "Học" buttons. A learner who wants task 3 while task 2 is current must tap the tile, which is less obvious than a button; the `▶` cursor and "Vào" label on `open` tiles carry that.
- The status bar packs name, streak and two shields into 48 px; long names truncate with an ellipsis at 14 characters — check the reviewer sees the full name in the `aria-label`.
- Executor traps: keep the growth-moment before/after logic exactly (`healthFrom`/`stageFrom`), only swap the visuals; do not animate the sprite with scale — use frame swaps; the connector must light *after* the response lands, never optimistically; `plant_name` may be the default "Mầm Non" — never hard-code it.
- Review checks: bottom button hidden while loading; locked tile inert; offline renders cache + one toast; every state above has its Vietnamese string; no `rounded-*` above 2 px in the diff.

## Acceptance
1. The hub renders companion, day bar and three tiles from `pet/status` + `quests/daily` with no text-row list and no `AppCard`.
2. The bottom button opens the current task within one tap; it is absent while the quest region loads and reads "Xem hành trình" when all three tasks are complete.
3. A locked tile is `aria-disabled` and does not navigate; an open/current tile does.
4. HP and day bars fill in 4-px cells; the HP label shows "HP n/100" in VT323 beside the bar.
5. After a completed task, the connector to the next tile lights and the growth-moment sequence plays within 2 s and only once.
6. Wilted status shows the ember panel, the `down` sprite and the "Hồi sinh" danger button; the path stays tappable.
7. Offline: cached data renders and exactly one toast appears.
8. Under `prefers-reduced-motion` no keyframe runs; every state is still distinguishable by glyph and text.
