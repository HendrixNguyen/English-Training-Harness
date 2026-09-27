# Design: Retro roadmap — the world map (`/roadmap`)

**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Inherits:** `harness/designs/roadmap-tree.md` — its wire contract `GET /api/v1/roadmap` (`modules[4]{week, title, focus, days[7]{day_number, date, title, tasks[3], minutes_spent, is_target_met}}`) and its truth rules (a star is earned by `is_target_met`, a missed day is missed) are kept unchanged; only the drawing changes. `frontend-shell.md` page-state convention.
**Kit:** `harness/UI-KIT.md` v2.
**Spec wireframe:** frontend spec §7.4 (zig-zag of 28 pills). Kept: 28 nodes, three states from the wireframe (cleared/today/locked), today scrolled into view. Departures: nodes are grouped into four **regions** (the modules are real), two more states exist (`partial`, `missed`) because the roadmap-tree contract knows them, and the path winds inside each region instead of one long zig-zag.

## 0. Research
- **Learner's job:** see where they are on the 28 days, what they have really done, and what the next region is about.
- **The moment that earns the next minute:** the companion sprite standing on today's node with the next region still under fog — the map says "you are here, and there is more".
- **What today does wrong:** `pages/roadmap.vue` is 28 identical pills whose "completed" means "before today", so a skipped day gets a star (closed by roadmap-tree); nothing shows the module structure the AI actually wrote.
- **Open questions, answered:** (a) *Region names:* `module.title` (the roadmap's own text, usually English), with "Vùng {week}" as the eyebrow so the Vietnamese chrome stays. (b) *What does tapping a node do?* Today → `/`; cleared/partial/missed → expands the day's three task titles and minutes under the region panel; locked → nothing. No day is playable from here (the quest endpoint only serves today). (c) *Region locking:* a region is under fog when every day in it is `locked`.

## 1. The signature
**Four regions under fog.** Each module is a `RetroPanel` "region" with its name on the speaker tab; inside, seven `MapNode` tiles on a winding path (3 per row, direction alternating, connectors in cells). Regions after the current one are drawn under a `ground-2` fog at 60 % with their names still legible — the shape of the journey is visible, its content is not. Clearing the last day of a region lifts its fog on the next visit with one stepped fade (300 ms). The companion stands on today's node.

## 2. Flow
Entry: hub streak badge, hub "Xem hành trình", settings back link. Exits: "‹ Trại" → `/`; today node → `/`; other nodes expand in place. Returning to the hub changes nothing there.

## 3. Layout (mobile-first, `max-w-md`)
```
│ ‹ Trại                       [B1]       │ 1 top bar: back; RankPlate small (cefr_level)
│ HÀNH TRÌNH 28 NGÀY                      │ 2 eyebrow VT323 16
│ Business English for meetings           │   roadmap.title VT323 28
│ Đã xong 9/28 ngày · 🔥 x7               │   font-body 16 ink-1 + Badge
│ ┌ Vùng 1 · Everyday small talk ───┐     │ 3 region panel (module 1): speaker tab; focus caption under it
│ │ Greetings and introductions      │     │
│ │ [★1]──[★2]──[○3]                 │     │   MapNode ×7, winding path, connectors in 4-px cells
│ │            │                     │     │
│ │ [★6]──[◐5]──[★4]                 │     │
│ │  │                               │     │
│ │ [★7]        7/7 ngày             │     │   region tally VT323 20
│ └──────────────────────────────────┘     │
│ ┌ Vùng 2 · Business email writing ─┐    │ 4 current region: outer line growth
│ │ Formal and informal tone          │    │
│ │ [★8]──[🌱9]──[ 10]               │    │   today node: growth tile + companion sprite 32 standing on it
│ │  ┌ Ngày 9 · T4 30/9 ── HÔM NAY ┐ │    │   expanded day (today by default): 3 task rows, minutes
│ │  │ Opening and closing an email │ │    │
│ │  │ 📖 Email openings   Đã xong  │ │    │
│ │  │ 📜 Reading a recap  10/30 ph │ │    │
│ │  │ 🗡 Reply to a client  Khoá   │ │    │
│ │  └──────────────────────────────┘ │    │
│ │ [ 13]──[ 12]──[ 11]  …            │    │
│ └───────────────────────────────────┘    │
│ ┌ Vùng 3 · Meetings and calls ─────┐    │ 5 fogged region: ground-2 60 % overlay, name visible, nodes ink-2
│ │ ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │    │
│ └──────────────────────────────────┘     │
│ ┌ Vùng 4 · … ──────────────────────┐    │
```
Node glyphs (never colour alone): `★` cleared (`torch`) · `🌱` today = companion sprite (`growth` tile) · `◐` partial (`growth` half cell, "12/30 phút") · `○` missed (hollow `ember` ring, "Bỏ lỡ") · padlock locked (`ink-2`). Data: `roadmap.title`, `cefr_level`, `day_number`, per day `is_target_met`, `minutes_spent`, `date`, `title`, `tasks[].task_type/title`. Desktop: same column.

## 4. States
- **Loading:** header from cache if any; four region panels with seven `ground-2` tiles each (`StateBlock loading` in the first).
- **Empty (`404 no_active_roadmap`):** one panel "Cậu chưa có hành trình." + "Bắt đầu hành trình 28 ngày" → `/onboarding`.
- **Error:** ember panel "Không tải được hành trình. Thử lại." with retry; a cached map stays visible underneath when present.
- **Offline:** cached roadmap renders; toast "Đang ngoại tuyến — bản đồ là bản đã lưu."
- **First-time (day 1):** region 1 current, today = node 1 with the sprite, regions 2–4 fogged; the expanded day shows the three open tasks.
- **Done (day 28 cleared):** all fog lifted; header "Đã xong 28/28 ngày"; the companion stands on node 28 in `fruitful` if the stage says so; no button.
- **Reduced motion:** fog lifts at once; the sprite idle is off; the cursor does not blink.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| Today node / expanded today "Vào" | tile insets | `/` | — |
| Cleared/partial/missed node | tile insets, `aria-expanded` | expands the day (one at a time; today collapses) | expanded day (page state) |
| Locked node | nothing | `aria-disabled`, tooltip-free | — |
| "‹ Trại" | drop | `/` | — |
| Retry | drop | refetch `GET /roadmap` | — |
On open the page scrolls today's region into view (`scrollIntoView block=center`), not the node, so the region name is read first.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel` | kit | `speaker`, `tone=growth` for the current region, `fog` | `fog` is a new boolean prop (kit addition) |
| `MapNode` | kit | `day`, `state`, `title`, `expanded`; emits `select` | today renders the `CompanionSprite` at 32 px on top |
| `CompanionSprite` | kit | `stage`, `size=32`, idle | — |
| `RankPlate` | from `retro-onboarding.md` | `level`, `size=small` | — |
| `Badge` | kit | `kind=streak`, `count` | — |
| `DayDetail` | new (kit addition) | `day`; slots task rows; emits `enter` for today | the expanded box inside a region; three rows with task icons and per-task state |
| `StateBlock`, `RetroToast`, `RetroButton` | kit | — | — |
**Kit additions:** `RetroPanel fog` (60 % `ground-2` overlay, `aria-hidden` content beneath, name stays readable — reason: locked regions must keep their shape); `DayDetail`. Path layout is CSS grid (3 columns, connectors as pseudo-elements), not a component.

## 7. Copy
"‹ Trại" · "Hành trình 28 ngày" · "Đã xong {n}/28 ngày" · "Vùng {week} · {module.title}" · region tally "{n}/7 ngày" · day header "Ngày {n} · {weekday} {d/m}" · "Hôm nay" · statuses "Đã xong" · "Bỏ lỡ" · "{m}/30 phút" · "Chưa mở" · task states "Đã xong" · "Đang học" · "Khoá" · "Vào" · empty "Cậu chưa có hành trình." / "Bắt đầu hành trình 28 ngày" · error "Không tải được hành trình. Thử lại." · offline "Đang ngoại tuyến — bản đồ là bản đã lưu." Weekday abbreviations as roadmap-tree: T2…CN.

## 8. Self-critique
- Traded away: the linear spine of `roadmap-tree.md`, which showed every day's title at once. The map shows titles only when expanded; the region focus line carries the "what is coming" job instead. If review finds learners want titles at a glance, add the title as a second line under each node in the current region only.
- Fog hides content that the learner technically owns (the plan exists). This is the deliberate risk: anticipation over transparency. The region name and focus stay visible so nothing important is hidden.
- Winding paths on a 3-column grid need care at `max-w-md` (each tile 40 px + connectors); the executor must not shrink tiles below 44 px tap size — use 48-px cells with 40-px art.
- Executor traps: `partial` vs `missed` come from `minutes_spent` and `date < today`, not from position; today's region is the one containing `day_number`; do not fog a region that contains a `missed` day.
- Review checks: a skipped day shows a hollow ring and "Bỏ lỡ"; today node navigates to `/`; locked nodes are inert; the page opens scrolled to the current region; keyboard can expand a day.

## Acceptance
1. Four region panels render from `modules[]`, named "Vùng n · {title}", with the module focus under the name.
2. Every day renders as one of five states with a glyph and a status word; `★` appears only where `is_target_met` is true.
3. Today's node carries the companion sprite, sits in a region with a `growth` outer line, and navigates to `/` on tap.
4. Regions with only locked days are fogged but their names stay readable (≥ 4.5:1).
5. Tapping a cleared, partial or missed node expands that day's three task titles and minutes; only one day is expanded at a time.
6. The page opens scrolled to the current region.
7. Empty, error and offline states show the kit strings; a cached map remains under an error panel.
8. Tap targets on the path are ≥ 44 px; under reduced motion nothing animates.
