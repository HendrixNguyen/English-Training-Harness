# UI kit v2 — the retro quest kit

Canonical for the designer, executor and reviewer. Supersedes kit v1 (Fraunces, emerald/amber/coral, 16-px cards) for visual style; v1's rules and flow survive below, restated. Origin: human idea `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (owner decisions: 16-bit JRPG, night-dungeon palette, the plant as party companion, mobile-first). Frontend spec §6.1 "Modern Gamified Minimalist" is superseded for *look*; spec §7 flows and states still bind. Values become real only through the kit plan that changes `frontend/tailwind.config.ts` and `frontend/components/`; until then this file is the contract the screen designs inherit.

**The world in one line.** A learner walks a night dungeon with a small plant companion; every quest is a dialogue box, every reward a chest, every 30-minute day a room cleared. Chrome is hard pixels; the words a learner *reads* (passages, questions, options) are never pixels.

## Tokens (proposed `tailwind.config.ts` names; hex is binding)

| Token | Hex | Role | Contrast (WCAG, on `ground-1`) |
|---|---|---|---|
| `ground-0` | `#0B0A1F` | page ground, gaps between panel lines, text on growth/torch fills | — |
| `ground-1` | `#151434` | panel fill (the dialogue box) | — |
| `ground-2` | `#1F1D4A` | raised/inset fill: secondary buttons, selected option, HP track | — |
| `line-lit` | `#C9C4F4` | outer panel line, cursor, dividers that must be seen | 10.7:1 |
| `line-dim` | `#3B3A78` | inner panel line, track borders, locked-node outlines (decorative only) | 1.8:1 — never carries text |
| `ink-0` | `#F4F1FF` | primary text, headings, dialogue | 16.0:1 |
| `ink-1` | `#B7B3DC` | secondary text: hints, captions, definitions | 8.9:1 |
| `ink-2` | `#8783B5` | muted: disabled labels, locked nodes, timestamps | 5.0:1 (floor for small text) |
| `growth` | `#3DE1B0` | progress made, HP ≥ 60, primary action, hit feedback (teal glow) | 10.7:1; `ground-0` on it 11.7:1 |
| `growth-deep` | `#178A69` | primary button shadow/pressed, bar shading | decorative |
| `torch` | `#F2A83B` | torchlight amber: streak, XP, chests, badges, "today", focus ring | 8.8:1; `ground-0` on it 9.7:1 |
| `torch-deep` | `#B8641E` | torch shadow/pressed | decorative |
| `ember` | `#FF5A4E` | danger: HP < 30, wilted, miss feedback, destructive | 5.8:1; `ground-0` on it ≥ 6:1 |
| `ember-deep` | `#B3261E` | danger shadow/pressed, wilted tint | `ink-0` on it 5.9:1 |

Semantic mapping kept from v1: growth = progress, torch (was `streak`) = reward, ember (was `alert`) = danger. Migration: the kit plan registers the new names and keeps `streak`/`alert`/`mute` as aliases until the last screen migrates, then deletes them with `paper`/`paper-dark`. No other saturated colour exists; there is no "blue" — indigo is the ground, not an accent.

## Type

| Role | Face (Google Fonts, self-hosted for the SW precache) | Size/leading (px) | Where |
|---|---|---|---|
| `font-display` | **VT323** (400) — the only pixel/terminal face on Google Fonts that ships a `vietnamese` subset (verified 2026-09-25) | title 34/36 · heading 28/32 · name/button 22/24 · counter 20/24 · eyebrow 16/20 uppercase, tracking 0.05em; **never below 14 px, never above 34 px** (it is condensed) | screen titles, panel headings, the speaker name over a dialogue box, quest and region names, button labels, and every counter: `HP 80/100`, `XP +10`, `20/30`, `09:42`, `x7` |
| `font-body` | **Nunito** (400/700; `vietnamese` subset verified) — rounded, tall x-height, reads at 16 px on indigo | body 17/26 · passage 18/30 · caption 14/20 | dialogue text, passages, questions, options, definitions, explanations, hints, errors |

VT323 is monospaced, so timers and counters never jitter. Body text is never below 14 px, never all-caps, never letter-spaced. English learning content is always `font-body`.

**Font rule (owner, 2026-09-25):** every face in the kit must ship a Vietnamese subset — check the Google Fonts css2 response (browser UA) for a `/* vietnamese */` block, or `subsets: "vietnamese"` in the family's `METADATA.pb`, before adopting one. Press Start 2P, Silkscreen, Pixelify Sans, Tiny5, Micro 5, Jersey and Sixtyfour do not have one and are excluded; Atkinson Hyperlegible Next does not either. Two faces only: VT323 and Nunito.

## Grid, shape, depth
- **8-px grid.** Spacing steps 4 (hairline gaps only) / 8 / 16 / 24 / 32 / 48. One column, `max-w-md`, 16-px side gutter; panels stacked with 16 px between; the primary button sits in a bottom bar with 16 px padding and `env(safe-area-inset-bottom)`.
- **Radius 0–2 px.** Panels 0, buttons and chips 2 px. Nothing rounder, ever — a rounded corner reads as v1.
- **Depth is drawn, not blurred.** No `box-shadow` blur, no gradients. A button's depth is a hard 4-px bottom shadow in its `-deep` colour; a panel's depth is its double line. Backdrop: `ground-0` with an optional 2-px-dot pattern at 4 % `line-lit` (`background-image` on `<main>`), off under `prefers-reduced-transparency`.
- **Pixel rendering.** Sprites and pixel icons: `image-rendering: pixelated` / `shape-rendering: crispEdges`, drawn on a 16- or 32-unit grid and scaled by whole numbers (×3, ×4). Never scale a sprite by 1.06.
- Desktop: the same column, centred, on the dotted `ground-0` backdrop. No second column.

## Components (`frontend/components/retro/` — proposed; each replaces the v1 component in brackets)

| Component | What it is | Props / events | States |
|---|---|---|---|
| `RetroPanel` [AppCard] | The dialogue box: `ground-1` fill, 2-px `line-lit` border, 2-px `ground-0` gap, 2-px `line-dim` outer line (`box-shadow: 0 0 0 2px ground-0, 0 0 0 4px line-dim`), 16-px padding. Optional `speaker` (VT323 name tab overlapping the top line) and `portrait` slot (48×48 sprite, left) | `speaker?`, `tone?: plain\|growth\|torch\|ember` (recolours the outer line), slots `portrait`, default | plain · toned · `aria-live` when `speaker` is set |
| `RetroButton` [AppButton] | 48-px tall, full-width on mobile, VT323 22 px label, 2-px radius, 4-px hard bottom shadow | `variant: primary\|secondary\|danger`, `loading`, `disabled`, `block` | primary `growth`/text `ground-0`/shadow `growth-deep`; secondary `ground-2`/`line-lit` border/text `ink-0`; danger `ember`/text `ground-0`/shadow `ember-deep`. Pressed: translate 2 px down, shadow 2 px (100 ms). Loading: label → `…` stepping 3 frames at 300 ms, width kept. Disabled: 50 % opacity, no shadow |
| `HpBar` [HealthBar] | HP-style bar: 12-px track `ground-2` with 2-px `line-dim` border; fill in **4-px cells** (width snaps to multiples of 4), `growth` ≥ 60 · `torch` 30–59 · `ember` < 30; label left VT323 20 `HP 80/100` | `value`, `max=100`, `label='HP'` | fill animates in cell steps (`steps(n)`, 300 ms); reduced motion snaps |
| `DayBar` [SegmentedProgress] | The 30-minute day as three 10-minute segments of `HpBar` cells with 4-px gaps; `20/30` in VT323 20, caption in body | `valueSeconds`, `segments=3`, `segmentSeconds=600`, `met` | met: all cells `growth` + caption "Phòng hôm nay đã xong" |
| `QuestNode` [QuestRow] | 56-px square tile on the hub path: pixel icon by `task_type` (book = vocabulary, scroll = reading, sword = practice), title + `(10 phút)` beside it, connector line to the next tile | `task`, `index`, `state: done\|current\|locked\|open` | done: `torch` star overlay, "Đã xong"; current: `growth` border + blinking `▶` cursor (steps(2), 600 ms), "Vào"; locked: `ink-2` + padlock, "Khoá"; open: `line-lit`, "Vào" |
| `Chest` (new) | Reward reveal: 64-px closed chest → open (2 frames, 200 ms) → contents list in `torch` | `items: {icon, label}[]`, `open` | closed · open · reduced-motion (open frame only) |
| `Badge` (new) | 32-px pixel emblem with a VT323 20 count (`x7` streak flame, shield) | `kind: streak\|shield\|star`, `count?`, `earned` | earned `torch` · empty outline `line-dim` |
| `MapNode` [RoadmapNode] | World-map node: 40-px tile on the path; day number in VT323 | `day`, `state: cleared\|today\|partial\|missed\|locked`, `title` | cleared `torch` star · today `growth` + companion sprite standing on it · partial half-filled `growth` · missed hollow `ember` ring · locked `ink-2` + fog (`ground-2` at 60 %) |
| `CompanionSprite` [PlantSvg] | The party companion, 32×32 grid ×4 = 128 px, one silhouette across stages | `stage: seed\|sprout\|sapling\|flowering\|fruitful\|wilted`, `health`, `size=128`, `react?: idle\|hit\|miss\|levelup\|down` | idle 2 frames / 1000 ms (breath); hit: 2-frame hop 200 ms; miss: 2-frame shake 200 ms; levelup: 3-frame flash `torch` 300 ms; down (wilted): lying frame, no idle. Unknown stage → sprout in `ink-2` |
| `SpeechBox` [SpeechBubble] | `RetroPanel` with the companion's portrait and `speaker=plant_name`; text types in at 30 ms/char (skippable by tap; off under reduced motion) | `line`, `name` | typing · settled |
| `RetroToast` (new) | 1-line `RetroPanel` sliding 8 px up from the bottom bar, 2 s, one at a time; `tone` sets the outer line | `line`, `tone` | — |
| `StateBlock` (kept) | `loading` = three `…` cells stepping; `empty`/`error` = one sentence + one `RetroButton` | as v1 | error uses `tone=ember`, never a red wall |
| `CountdownTimer` (kept) | VT323 20 `mm:ss`; measures, never gates | as v1 | — |

## Interaction states
Every control: rest → **pressed within 100 ms** (button drop, tile inset, option cursor jump) → result. Focus: 2-px `torch` outline, 2-px offset, on everything focusable. Selected option: `ground-2` fill + `▶` cursor in `line-lit` + `aria-checked`. Disabled: 50 % opacity and `aria-disabled`, never removed from the DOM. Hit/miss feedback: the option row's outer line turns `growth`/`ember` **and** a text label ("Trúng!" / "Trượt…") and an icon appear — never colour alone.

## Motion budget
- One place per screen spends motion: the companion (hub, revive), the answer feedback (learning room), the chest (task done), the map cursor (roadmap).
- All UI motion is **stepped** — `animation-timing-function: steps(2|3)`, 150–300 ms; sprite idles 2 frames per second. Bars fill in 4-px cells. No easing curves, no blur, no scale tweens, no page transitions, no bouncing loaders.
- `prefers-reduced-motion: reduce`: idles off, reactions become a single frame swap, typing text appears at once, bars snap, toasts fade in one step. The reduced variant is designed, not disabled: every reaction still has its static frame.
- Sound-free by default; no audio API in scope.

## Copy register
- Vietnamese, sentence case, short. The companion speaks as **"tớ"** and calls the learner **"cậu"**; the app never exclaims more than once per line; nothing says "chúc mừng" — the chest is the congratulation.
- Quest vocabulary, one word per concept, kept through a flow: **nhiệm vụ** (a task), **phòng** (the day's 30 minutes: "Phòng hôm nay"), **hành trình** (the roadmap), **vùng** (a module/week), **kho báu** (a reward chest), **hồi sinh** (revive), **HP** (companion health; "máu" in sentences), **XP** (session points), **streak** stays "chuỗi ngày". Buttons are verbs: "Vào", "Trả lời", "Tiếp tục", "Mở kho báu", "Hồi sinh".
- Learning content (passages, questions, options, explanations) is English and set in `font-body`; the chrome around it is Vietnamese.
- Errors say what happened and what to do next, in the app's voice, in a `tone=ember` panel; empty states are an invitation with one button.

## Accessibility floor
- Body text ≥ 4.5:1 on `ground-1` (all three inks pass; `ink-2` is the floor and is never used below 14 px). `line-dim` never carries meaning.
- Targets ≥ 44 px (buttons 48, tiles 56, options 56). Focus visible everywhere; the `▶` cursor follows keyboard focus.
- Never colour alone: every state has a glyph or a word (star/padlock/hollow ring + status text; hit/miss labels; HP number beside the bar).
- Sprites carry `aria-label` with stage and HP ("Mầm Non, giai đoạn sapling, 80 HP"); timers are `aria-live="off"`; feedback and toasts are `aria-live="polite"`, announced once.
- The pixel face is display-only: any string over ~40 characters is `font-body`.

## Dark/light
Dark-native: `ground-*` is the only ground. `darkMode: 'media'` stays in the config but no `dark:` variants are written — the screen is the same under both schemes. A "daylight" variant (parchment grounds, ink inverted) is out of scope and would be a separate kit revision.

## Rules (v1 intent, restated)
- **Mobile-first, thumb-first.** One primary `RetroButton` per screen in the bottom bar; ≥ 44 px targets; the learner never reaches for the top of the screen to progress.
- **Every async region has three states** through `StateBlock`; never a blank panel, never raw JSON. Offline shows cached content plus a quiet `RetroToast`, never an error wall.
- **Semantic colour is not decoration.** Growth = progress, torch = reward, ember = danger; indigo is the ground.
- **Copy** as above; the companion speaks, the app instructs.
- **Motion** as above; one place per screen, stepped, removed-but-designed under reduced motion.
- **Feedback within 100 ms** of any tap: pressed state first, then the real result.
- **Complete by doing.** A task ends when its last item is answered; the timer measures time for `daily_progress`, never gates the button.
- **Reuse before inventing.** A screen design uses this component set; a new component or token is proposed under "Kit additions" in the design doc and lands only through a plan that also changes the code.

## Flow (spec §7, unchanged routes)
Title screen `/login` → Onboarding `/onboarding` (goal → companion name → placement as the first encounter → waiting → roadmap ready with calibration) → **Hub** `/` (companion, day bar, three quests as a path) ↔ **Learning room** `/learn/:id` (one item per dialogue box, hit/miss per answer, chest on completion → back to the hub with the growth moment) · World map `/roadmap` · Revive `/revive` (HP 0) · Inn `/settings`. Screen docs: `harness/designs/retro-*.md`; build order in `harness/designs/retro-README.md`.
