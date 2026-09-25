# Design: `/roadmap` — the real 28-day plan, with true per-day completion

**Idea:** `harness/ideas/2026-09-25-run-01/roadmap-tree-shows-the-real-plan-module-and-day-titles-with-.md`
**Inherits:** `harness/designs/frontend-shell.md` (palette, type, card/button shapes, Vietnamese UI register, page-state convention, accessibility floor). This document replaces the shell design's §2.5 for `/roadmap`; the shell's "completed = before today — open question" is closed here.
**Wire contract:** `GET /api/v1/roadmap` (backend spec §6.2 after this plan) → `{roadmap_id, title, cefr_level, created_at, day_number, modules[4]{week, title, focus, days[7]{day_number, date, title, tasks[3]{task_type, title, duration_minutes}, minutes_spent, is_target_met}}}`; `404 {error: "no_active_roadmap"}`; the usual `{error: code}` envelope otherwise. No exercise content is on this page.

**Subject and job.** A learner on day 9 of an AI-written 28-day plan, opening the roadmap because they want to know two things: *how am I actually doing* and *what is coming*. The page's single job is to be the honest record — every day shows what the plan called it and what the learner did with it, and the same skipped day the plant lost health for is not awarded a star here. Secondary job: show that the plan is a plan (four weeks with a focus each, three named tasks a day), which is the reason to have an AI roadmap at all.

**Register.** Vietnamese UI copy, sentence case. Day titles, module titles and task titles are the roadmap's own text (whatever language the generator wrote them in — usually English for a B1 learner); the chrome around them is Vietnamese. Status lines state a fact, never a judgement: "Bỏ lỡ" (missed) is a fact, "Bạn đã lười" is not. No exclamation marks.

---

## 1. The signature: a calendar spine that tells the truth

The zig-zag of pills (wireframe 7.4, shell §2.5) worked because the nodes were nameless. Named nodes need width, so the page becomes **one column on a vertical spine**: a 2 px rail on the left carries the 28 day markers and the four week milestones in order; each day is a full-width row to the right of its marker. The structure encodes a true fact — a roadmap is a sequence of calendar days — which is why the rail is allowed to be the structural device, and why each day also shows its real date ("T3 30/9") next to its number: the tree is pinned to the learner's calendar, the same dates their Google Tasks list already carries.

The one bold move is spent on the markers: they are the only coloured objects on the page, and their colour is earned, not scheduled. A `streak` star appears only on a day whose `daily_progress.is_target_met` is true. A skipped day is a hollow `mute` ring. A day with minutes but no target is a half-filled `streak` ring with the minutes written beside it. Today's marker is the only `growth` object. Locked days are `mute` locks. Scrolling down the spine, the learner reads their month at a glance: stars, gaps, a green sprout, then locks.

---

## 2. Layout (mobile, one column, `max-w-md`, 8-pt grid)

```
┌──────────────────────────────────────────┐
│ AppHeader (avatar · name · streak · …)   │
│ LỘ TRÌNH HỌC 28 NGÀY            (eyebrow)│
│ Business English for meetings   (title)  │  ← roadmap.title, Fraunces 28/32
│ Trình độ B1 · Đã hoàn thành 9/28 ngày    │  ← one line, body 16, mute
├──────────────────────────────────────────┤
│ ● TUẦN 1 · Everyday small talk     7/7   │  ← module milestone on the rail
│ │ Greetings and introductions      focus │  ← module.focus, caption, mute
│ │                                         │
│ ★ Ngày 1 · T3 22/9                        │  ← day row: number, date
│ │ Meeting a new colleague           title │  ← day.title, body 16 semibold
│ │ Đã hoàn thành                     status│  ← status line, caption
│ │                                         │
│ ○ Ngày 2 · T4 23/9                        │
│ │ Small talk at lunch                     │
│ │ Bỏ lỡ                                   │
│ │                                         │
│ ◐ Ngày 3 · T5 24/9                        │
│ │ Asking follow-up questions              │
│ │ 12/30 phút                              │
│ ⋮                                         │
│ ● TUẦN 2 · Business email writing   1/7  │
│ │ Formal and informal tone                │
│ 🌱 Ngày 9 · T4 30/9            HÔM NAY   │  ← growth marker, growth chip
│ │ Opening and closing an email            │
│ │ Đang học · 10/30 phút                   │
│ │ ┌─────────────────────────────────────┐ │  ← expanded by default for today
│ │ │ Từ vựng   Ten email openers   10 phút│ │
│ │ │ Đọc hiểu  Two sample emails   10 phút│ │
│ │ │ Thực hành Write a reply       10 phút│ │
│ │ │ [ Học ngay → ]              (primary)│ │
│ │ └─────────────────────────────────────┘ │
│ 🔒 Ngày 10 · T5 1/10                      │
│ │ Requesting information                  │
│ │ Chưa mở khóa                            │
│ ⋮                                         │
└──────────────────────────────────────────┘
```

- **Header block.** Eyebrow "Lộ trình học 28 ngày" (section label 12/16 uppercase tracked — the wireframe's title becomes the eyebrow because the roadmap now has a name of its own). Title = `roadmap.title` in Fraunces 28/32, wrapping to two lines at most (`line-clamp-2`). One line under it: "Trình độ {cefr_level} · Đã hoàn thành {n}/28 ngày", where n counts days with `is_target_met`.
- **Module milestone.** Sits on the rail as a filled `ink` dot (`paper` in dark), larger than a day marker. Left: "TUẦN {week} · {module.title}" (label 12/16 uppercase for "Tuần n", the title in body 16 semibold, same line); right: "{met}/7" in tabular caption, `mute`. Below: `module.focus` as a caption in `mute`. No card around a module — the rail is the grouping.
- **Day row.** Marker on the rail (24 px, centred on the 2 px line); to the right, three lines: `Ngày {n} · {weekday} {d/M}` (caption, `mute`, tabular digits), `day.title` (body 16 semibold, `ink`/`paper`), and the status line (caption; colour by state, §4). The whole row is one `button` with `aria-expanded`; tapping toggles the task panel. Min height 56 px; touch target is the whole row.
- **Task panel.** Indented under the row, `AppCard` surface without a title: three rows, one per task in the fixed order vocabulary → reading → practice (`TASK_ORDER` in `stores/quest.ts`), each "label · title · {duration_minutes} phút". Labels are the three task categories in Vietnamese — Từ vựng · Đọc hiểu · Thực hành — defined once as `TASK_LABELS` in `utils/roadmap.ts` (the dashboard's `QuestRow` shows titles only, so there is no existing map to reuse). Today's panel ends with the primary button "Học ngay →" to `/` (this replaces the old behaviour where the today pill itself was a link). Other panels have no button.
- **Rail geometry.** Rail at x = 12 px inside the card gutter; markers 24 px; row content starts at 40 px. Rows are separated by 16 px; a module milestone gets 24 px above. Weeks are not otherwise separated — the milestone is the separator.
- **Scroll.** On open, today's row scrolls to the vertical centre (`block: 'center'`), as now; today's panel is expanded on open, every other panel is collapsed. Expansion state is component-local (not persisted).
- **Desktop.** The same column, centred; nothing rearranges (shell rule).

---

## 3. Components

| Component | Status | Role |
| --- | --- | --- |
| `AppHeader`, `AppCard`, `AppButton`, `StateBlock` | existing | shell |
| `RoadmapNode` (`components/roadmap/RoadmapNode.vue`) | **rewritten** | one day row: marker, three lines, `aria-expanded` toggle, the task panel, today's "Học ngay →". Props: the node from `utils/roadmap.ts` plus `expanded` (v-model). Emits nothing else. |
| `RoadmapModuleHeader` (`components/roadmap/RoadmapModuleHeader.vue`) | **new** | the milestone row: week, title, focus, `{met}/7` |
| `RoadmapMarker` (`components/roadmap/RoadmapMarker.vue`) | **new** | the 24 px marker for one state; `aria-hidden` (the status line carries the meaning). Kept separate so the five glyph/colour pairs live in one place and `RoadmapNode` stays about layout. |

No new icons: the markers are text glyphs (★ ◐ ○ 🌱 🔒) inside a ring, so they precache with the fonts and render offline. No custom disclosure component — a `button[aria-expanded]` plus a `v-show` panel is enough, and a native `<details>` would fight the row's layout.

---

## 4. States

### 4.1 Day states (from `utils/roadmap.ts` `dayState`)

Precedence, top first:

| State | Rule | Marker | Status line |
| --- | --- | --- | --- |
| `today` | `day_number === roadmap.day_number` | 🌱 on a `growth` ring; a small `growth` chip "HÔM NAY" at the row's right | "Đang học · {minutes_spent}/30 phút"; "Đã đủ 30 phút" once `is_target_met` |
| `completed` | `is_target_met` | ★ in `streak` on a `streak/10` fill | "Đã hoàn thành" |
| `partial` | past day, `minutes_spent > 0` | ◐ in `streak` on a hollow `streak/40` ring | "{minutes_spent}/30 phút" |
| `missed` | past day, `minutes_spent === 0` | ○ hollow `mute/40` ring, no fill | "Bỏ lỡ" in `mute` |
| `locked` | future day | 🔒 on a hollow `mute/30` ring | "Chưa mở khóa" in `mute` |

"Past" and "future" are by `day_number` relative to `roadmap.day_number`, never by the client clock — the server already decided which day it is in the learner's timezone, and the client must not disagree with it around midnight. A `completed` day past 30 minutes still says "Đã hoàn thành", not the minutes: the fact that matters is the target. A past day with `minutes_spent >= 30` but `is_target_met` false is `partial` ("30/30 phút") — it is what the pet was told, and the tree does not know better than the pet.

Title colour: `ink`/`paper` for `today`, `completed`, `partial`; `mute` for `missed` and `locked` (the day still has a name; it is just not the one to look at). Locked rows expand too — seeing tomorrow's tasks is a stated point of the page; nothing in a title is secret.

### 4.2 Page states (shell convention)

| State | Render |
| --- | --- |
| loading (`roadmap.loading && !roadmap.outline`) | `StateBlock state="loading"` inside the card |
| empty (`404 no_active_roadmap`) | unchanged: `StateBlock state="empty"` "Bạn chưa có lộ trình học." with "Tạo lộ trình 28 ngày" → `/onboarding` |
| error (`roadmap.error && !roadmap.outline`) | unchanged copy: `StateBlock state="error"` "Không tải được lộ trình." with "Thử lại" |
| offline with cache | the service worker's NetworkFirst answer renders as data; no special caption on this page (the dashboard owns the offline caption) |

The page no longer needs `quest.daily`: `roadmap.day_number` is the same number, computed by the same server function. It still calls `pet.load()` for the header's streak chip, as today.

---

## 5. Tokens (all existing; nothing new is registered)

- Colour: `growth` (today's ring, the HÔM NAY chip, "Học ngay →"), `streak` (stars and half-rings; `streak/10` fill on completed), `mute` (rail, dates, missed/locked rings and text, focus lines), `ink`/`paper` surfaces and titles per shell; `alert` only in the error block.
- Type: eyebrow and "TUẦN n" `text-xs font-semibold uppercase tracking-wider text-mute`; roadmap title `font-display text-[28px] leading-8`; day and module titles `text-base font-semibold`; dates, status lines, focus and `{met}/7` `text-[13px] leading-[18px] tabular-nums`.
- Shape: task panel 16 px radius (`AppCard`), "Học ngay →" 12 px, rings fully round; rail 2 px `mute/30`.
- Motion: the panel opens without animation (a list toggling height is noise on a phone); today's row gets the shell's 400 ms ease on the `growth` ring's appearance only. `prefers-reduced-motion` → nothing moves.
- Focus: shell's `ring-2 ring-growth ring-offset-2` on every row button and on "Học ngay →".
- Accessibility: each row's accessible name is "Ngày n, {title}, {status}" (the three lines in reading order); the marker is `aria-hidden`; `aria-current="step"` on today's row; the expanded panel is a `ul` with `aria-label="Nhiệm vụ ngày n"`.

## 6. Self-critique

Cut before shipping: a per-week progress bar under each milestone (the `{met}/7` count says it in three characters), a streak flame on consecutive stars (the header already shows the streak; drawing it twice makes the rail busy), coloured backgrounds on rows by state (colour would then carry meaning twice and the page would look like a heatmap), and the zig-zag connectors (decorative once rows have titles). The risk taken is dropping the wireframe's zig-zag for a spine; it is defensible because §7.4's promise is the *content* of each node — name and true state — and a spine is the only layout that gives a name room at `max-w-md` while keeping 28 markers readable in one scroll. The star is still a star, the sprout is still today, the lock is still a lock: a learner who knew the old page recognises every state.
