# Design: Retro kit — tokens, fonts, base components, companion sprite (plan 1 of 6)

**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Inherits:** `harness/designs/frontend-shell.md` §4 (the v1 component inventory this replaces), `harness/designs/retro-README.md` (build order; "The kit plan (1) in one paragraph").
**Kit:** `harness/UI-KIT.md` v2 — the source for every token, type role, state and motion rule. This doc does not repeat the kit; it adds only what an executor needs to build `frontend/components/retro/*` without guessing and what the reviewer needs to check them.
**Spec wireframe:** none — no page changes in plan 1. Screens come in plans 2–6 (`retro-hub.md` … `retro-settings.md`).

## 0. Research
- **Learner's job:** none yet — nothing a learner navigates changes in this plan. The kit's job is to make plans 2–6 pure page work: every later screen composes these components and adds no token.
- **What the executor builds from:** the kit's component table (props, states) plus this doc's sizes, DOM contracts, sprite data format and tests. Anything not stated here or in the kit is the executor's call and is recorded in the plan's Notes.
- **What today does wrong:** `components/ui/*` are v1 (16/12-px radii, spinner loaders, Fraunces); `PlantSvg` is vector art with easing sway; `QuestRow`/`RoadmapNode` use `NuxtLink` and emoji glyphs; `tailwind.config.ts` has seven v1 tokens. None of it is deleted here — v1 pages keep building until each screen plan swaps its components (README rule).
- **Open questions, answered from the kit and the code:**
  1. *Visible change in plan 1?* Yes, small and intended: `font-display`/`font-body` swap to VT323/Nunito on every page (v1 sizes 14–34 px all satisfy the VT323 rule), and `StateBlock`/`CountdownTimer` are restyled in place. The `html` ground, the global focus ring and the `paper` tokens stay v1 until plan 6 — flipping the ground now would put v1's white cards under `ink-0` text on light-scheme devices.
  2. *Sprite technique?* Inline SVG `<rect>` grid from string rows (§4), not a PNG sheet: agents cannot author binaries, rects recolour through CSS variables (health tint, unknown-stage `ink-2`, the level-up flash), and Vitest can count pixels. Reactions are stepped transforms/palette swaps of one base drawing per stage — six drawings, not twenty.
  3. *Navigation inside kit components?* None. `QuestNode`/`MapNode` emit `enter`/`select`; pages call `navigateTo`. Keeps the kit Nuxt-free so `@vue/test-utils` mounts it without stubs (v1 `QuestRow` embeds `NuxtLink` and is untested for that reason).
  4. *Portrait 48 px on a 32-grid?* 48 = 16 units × 3: the portrait is a `viewBox` crop of the face region (§4). Every size the screen docs use is a whole multiple: full 32/64/128, face 48.

## 1. Token migration (`frontend/tailwind.config.ts`)
`tokens` stays a flat `Record<string, string>` (the contrast test iterates it). Hex is the kit's; roles are in the kit table.

| Name | Hex | Fate |
|---|---|---|
| `ground-0` `ground-1` `ground-2` | `#0B0A1F` `#151434` `#1F1D4A` | new |
| `line-lit` `line-dim` | `#C9C4F4` `#3B3A78` | new |
| `ink-0` `ink-1` `ink-2` | `#F4F1FF` `#B7B3DC` `#8783B5` | new; flat keys — `text-ink-0`, not `ink.0` |
| `growth` | `#3DE1B0` (was `#10B981`) | **kept, hex changes**; `growth-deep` `#178A69` new |
| `torch` `torch-deep` | `#F2A83B` `#B8641E` | new |
| `ember` `ember-deep` | `#FF5A4E` `#B3261E` | new |
| `streak` → `torch`, `alert` → `ember`, `mute` → `ink-2` | same hex as their target (`streak: tokens.torch`) | aliases until plan 6, then deleted; v1 components keep compiling and pick up the v2 hue |
| `ink` `paper` `paper-dark` | `#1E293B` `#F8FAFC` `#0F172A` unchanged | v1-only; deleted in plan 6 with the `html` rule in `main.css` |
| `borderRadius` | `card: 16px`, `btn: 12px` unchanged | v1-only, deleted in plan 6. Retro components use built-ins only: `rounded-none` (0) and `rounded-sm` (2 px). No new radius token. |
| `fontFamily` | `display: ['VT323', 'monospace']` · `body: ['Nunito', 'system-ui', 'sans-serif']` | replaces Fraunces / Source Sans 3 |

`darkMode: 'media'` stays; no `dark:` variant is written in `components/retro/`. `tests/unit/tokens.test.ts` is rewritten for v2 (§6).

**Kit clarification (computed 2026-09-26, no token change):** `ink-2` on `ground-2` is **4.46:1** — below the floor. `ink-2` text sits only on `ground-0`/`ground-1` (5.0:1); on a `ground-2` surface (secondary button, selected option, HP track, fog) muted text uses `ink-1`. The `locked` `MapNode`/`QuestNode` keep `ink-2` because they are `aria-disabled` and carry the padlock glyph.

## 2. Fonts
- **Packages (fontsource v5; names by reasoning, verified by the executor with `ls node_modules/@fontsource/vt323` after `npm install` — this worktree has no `node_modules`):** `@fontsource/vt323` (400 only) and `@fontsource/nunito` (static; 400 and 700). Remove `@fontsource-variable/fraunces` and `@fontsource/source-sans-3` from `package.json`. Static Nunito over `@fontsource-variable/nunito`: two weights, smaller, no variable-axis rendering differences between engines.
- **Imports in `nuxt.config.ts` `css:`** — per-subset files only, so Vite emits exactly nine `woff2` and the precache carries nothing else: `@fontsource/vt323/{latin,latin-ext,vietnamese}-400.css`, `@fontsource/nunito/{latin,latin-ext,vietnamese}-{400,700}.css`. The `vietnamese` subset covers only the Vietnamese code points (`U+1EA0–1EF9`, tone marks, `đ`); `latin` carries A–Z, so all three subsets are required. If a per-subset file is missing from the installed package, fall back to `400.css`/`700.css` (all subsets, `unicode-range` split) and note the larger precache in the plan.
- **Precache:** unchanged mechanism — the per-subset CSS references its `files/*.woff2`, Vite emits them under `_nuxt/`, `injectManifest.globPatterns` already includes `woff2`, `precacheAndRoute(self.__WB_MANIFEST)` caches them at install. Verify after `npm run build`: nine `*vietnamese*|*latin*` `.woff2` under `.output/public/_nuxt/` and each listed in the generated `sw.js` manifest.
- **Roles and sizes:** the kit Type table is binding. Tailwind classes to use: display title `font-display text-[34px] leading-9` · heading `text-[28px] leading-8` · name/button `text-[22px] leading-6` · counter `text-xl leading-6` (20/24) · eyebrow `text-base leading-5 uppercase tracking-[0.05em]`; body `font-body text-[17px] leading-[26px]` · passage `text-lg leading-[30px]` · caption `text-sm leading-5`. Counters and timers add `tabular-nums` (VT323 is monospaced; the class documents intent).

## 3. Shared mechanics
- **`assets/css/retro.css`** (new, imported after `main.css`): `.retro-pixel { image-rendering: pixelated; shape-rendering: crispEdges }`; the stepped keyframes `retro-breath` (translateY 0 → −1 unit, `steps(1)`, 1000 ms, infinite), `retro-hop` (0 → −2 units → 0, 200 ms), `retro-shake` (−1 → +1 unit, 200 ms), `retro-flash` (palette to `torch` → `ink-0` → base, 300 ms), `retro-dots` (loader cells, 900 ms, `steps(3)`), `retro-blink` (cursor, 600 ms, `steps(2)`), `retro-rise` (toast, translateY 8 px → 0, 150 ms, `steps(1)`); and one `@media (prefers-reduced-motion: reduce)` block that sets `animation: none` and `transition: none` on every `[class^="retro-"]` and `.retro-anim` element. Reduced motion is CSS-only — no JS media-query reads — so happy-dom tests exercise the static frame by mounting with the `reduced` prop described per component. Units: a "unit" is `var(--px)`, the scale in px (`--px: 4` at 128 px), so transforms stay on the pixel grid.
- **Focus:** every focusable retro element sets `focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch` in its own template (the global `ring-growth` rule in `main.css` is v1 and stays until plan 6).
- **Pressed within 100 ms:** no transitions on press. `RetroButton` uses `:active` classes; tiles use `:active` inset (border colour → `line-dim`, translateY 2 px).
- **Glyph text:** `▶`, `★`, `○`, `✔`, `✖` are drawn as pixel glyphs (§4 `PixelArt`), never as font characters — VT323's coverage of U+25xx is not guaranteed and a fallback glyph breaks the pixel look. Emoji strings passed by pages (e.g. `Chest` items "🔥 x7") render as-is in `font-body`.

## 4. Pixel art: `PixelArt` and the companion
**`PixelArt.vue` (internal kit addition, reason: one renderer and one test for every sprite, icon and glyph).** Props `rows: string[]` (N strings of N chars), `palette: Record<char, cssColor>`, `size: number` (px; snapped down to a whole multiple of N), `label?` (sets `role="img"` + `aria-label`; without it `aria-hidden="true"`). Renders `<svg viewBox="0 0 N N" class="retro-pixel" shape-rendering="crispEdges">` with one `<rect height="1">` per horizontal run of the same char (run-length merged; `.` is transparent and emits nothing). Fill is `var(--px-<char>)`, the variables set on the `<svg>` from `palette`, so a parent recolours by overriding variables. Data lives in `utils/pixelArt.ts`: `COMPANION[stage]: string[]` (32 rows), `GLYPHS[name]: string[]` (16 rows: `book`, `scroll`, `sword`, `flame`, `shield`, `star`, `padlock`, `ring`, `chestClosed`, `chestOpen`, `check`, `cross`; 8 rows: `cursor`) and `PALETTE` (below). `tests/unit/pixelArt.test.ts` asserts every row set is square and uses only palette chars.

**Palette chars** (the only ones allowed in any row): `.` none · `k` `ground-0` outline · `g` `growth` · `G` `growth-deep` · `i` `ink-0` highlight · `t` `torch` · `T` `torch-deep` · `e` `ember` · `E` `ember-deep` · `d` `line-dim` · `l` `line-lit`.

**Companion anatomy (32×32, one silhouette).** Rows 22–31 are the pot and never change between stages: rim rows 22–24 (`t` light band on 23, cols 7–24), body rows 25–31 narrowing to cols 9–22, the **face on the pot**: eyes 2×2 `k` at cols 12–13 and 18–19 rows 26–27 with an `i` catchlight at (13,27) and (19,27), a smile `k` at (13,28) (18,28) and cols 14–17 row 29. Because the face lives on the pot, every stage has the same expression, the portrait crop is identical for all stages, and reactions read at 32 px. The plant grows in rows 3–21, stem always cols 15–16 (`g` left, `G` right, `k` outline at 14 and 17), leaves 1-px outlined with one `i` glint each.

| Stage | Rows used | Silhouette |
|---|---|---|
| `seed` | 19–21 | no stem; a 6×3 dome of `G` with a `g` top row and `k` outline on the soil, cols 13–18, one `i` glint at (14,20) |
| `sprout` | 9–21 | the sketch below: stem 7 rows, one left leaf (rows 11–16) and one higher right leaf (rows 9–15) |
| `sapling` | 6–21 | stem 13 rows; four alternating leaves (right 8–12, left 10–14, right 13–16, left 15–18, each 6 wide); a 3-row `g` bud at the tip (rows 6–8, cols 14–17) |
| `flowering` | 4–21 | sapling + three 3×3 flowers (`i` petals, `t` centre) at the tip (rows 4–6) and on the outer tip of the two upper leaves |
| `fruitful` | 3–21 | flowering with leaves one px wider; flowers become 3×3 fruits (`t` with a `T` bottom row); a fourth fruit on the lower right leaf |
| `wilted` | 9–21 | sapling's shape with `g→e`, `G→E`; the top six stem rows step one column right per row (bend), leaves mirrored to point down, no bud; eyes closed (row 26 `kk` only), smile inverted (`k` at cols 14–17 row 28, at (13,29) (18,29)) |

Sprout at 1× (verified 32×32; `.` transparent):
```
................................
................................
................................
................................
................................
................................
................................
................................
................................
...................kkk..........
..................kgggk.........
........kkk......kggigk.........
.......kgggk.....kgggGk.........
......kgggigk....kgGGk..........
......kggggGk.kgGGGk............
.......kgggGGkkgGk..............
.........kkkkkkgGk..............
..............kgGk..............
..............kgGk..............
..............kgGk..............
..............kgGk..............
..............kgGk..............
........kkkkkkkkkkkkkkkk........
.......kttttttttttttttttk.......
.......kTTTTTTTTTTTTTTTTk.......
........kTTTTTTTTTTTTTTk........
........kTTTkkTTTTkkTTTk........
........kTTTkiTTTTkiTTTk........
........kTTTTkTTTTkTTTTk........
.........kTTTTkkkkTTTTk.........
.........kTTTTTTTTTTTTk.........
.........kkkkkkkkkkkkkk.........
```

**`CompanionSprite.vue`** wraps `PixelArt` with `rows = COMPANION[normalizeStage(stage).stage]` (unknown → sprout with `--px-g`/`--px-G` overridden to `ink-2`, as v1). Props: `stage: string`, `health: number`, `name?: string`, `size = 128`, `crop: 'full' | 'face' = 'full'` (face = `viewBox="8 16 16 16"`, so 48 px is ×3), `react: 'idle' | 'hit' | 'miss' | 'levelup' | 'down' = 'idle'`, `reduced = false`. Emits `reacted(react)` on `animationend` for `hit|miss|levelup`, then returns to idle. Health tint: `health < 30` sets `--px-g` to `ember`, `30–59` to `torch`, else `growth` (reuse `healthTone()`; map `streak→torch`, `alert→ember`). Reactions are applied to the plant `<g>` (rows 0–21), never the pot, except `levelup` (whole sprite) and `down`:

| `react` | Frames | Implementation |
|---|---|---|
| `idle` | 2, 1000 ms loop | `retro-breath` on the plant group; off when `stage='wilted'` |
| `hit` | 2, 200 ms once | `retro-hop` |
| `miss` | 2, 200 ms once | `retro-shake` |
| `levelup` | 3, 300 ms once | `retro-flash`: all `--px-*` except `k` → `torch`, then `ink-0`, then base |
| `down` | 1, static | the `wilted` rows with `transform="rotate(90 16 16) translate(0 7)"` on the whole drawing (pot on the left, plant lying right, pot base on row 31); no idle. `aria-label` adds ", đã gục" |
| `reduced` (prop or media query) | 1 | idle static; hit = plant raised 2 units for 300 ms then base; miss = base; levelup = torch palette for 300 ms then base — a frame swap, no keyframe |

`aria-label`: `${name ? name + ', ' : ''}giai đoạn ${stage}, ${health} HP`; `data-stage`, `data-react` and `data-frame` attributes for tests.

## 5. Components (`frontend/components/retro/`)
Sizes are at 1× CSS px on the 8-px grid. States and colours are the kit table's; only what the kit leaves open is added here. Every component: `rounded-none`/`rounded-sm` only, no `dark:`, no `transition` except where a stepped one is named, `font-display`/`font-body` per the kit Type table.

**`RetroPanel`** — `<section>`; `border-2 border-line-lit bg-ground-1 p-4` plus `box-shadow: 0 0 0 2px #0B0A1F, 0 0 0 4px <outer>`, `<outer>` = `line-dim` or the `tone` colour. Needs 4 px of outside room: pages stack panels with 16 px. `speaker` renders a tab `<h2>` (VT323 22, `bg-ground-1 text-ink-0 px-2`, `absolute -top-3 left-3`) and the section gets `mt-3` and `aria-labelledby`; `aria-live="polite"` when `speaker` is set. `portrait` slot: a 48-px box on the left with `gap-4`, content column grows. Props also accept `band?` (full-bleed, `p-2`, no outer line — revive) and `fog?` (a `ground-2`/60 % overlay child, `inert` content — roadmap) as booleans; both are one class each and cost nothing now.
```
 ╔══════════════════╗  line-dim  (4 px ring, recoloured by tone)
 ║┌────────────────┐║  ground-0  (2 px gap)
 ║│ line-lit 2 px   │║
 ║│  ground-1, p-4  │║
```

**`RetroButton`** — `<button>` `h-12 min-w-[48px] px-5 rounded-sm font-display text-[22px] leading-6`; `block` → `w-full`. Depth `box-shadow: 0 4px 0 0 <deep>` (zero blur); `:active` → `translate-y-[2px]` and `0 2px 0 0`, no transition. Variants: primary `bg-growth text-ground-0` shadow `growth-deep`; secondary `bg-ground-2 text-ink-0 border-2 border-line-lit` shadow `line-dim`; danger `bg-ember text-ground-0` shadow `ember-deep`. v1 `ghost` maps to `secondary` when pages migrate. `loading`: label kept in the DOM at `opacity-0` (width kept), an absolutely centred `…` where dots light 1→2→3 with `retro-dots`; `aria-busy`; disabled while loading. `disabled`: `opacity-50`, `shadow-none`, `aria-disabled` and `disabled`. Props `type`, `variant`, `loading`, `disabled`, `block`; slot default.
```
 ┌──────────────────┐
 │   VÀO NHIỆM VỤ   │  48 px, growth
 └──────────────────┘
 ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀  4 px growth-deep (2 px when pressed)
```

**`HpBar`** — `role="meter"` with `aria-valuenow/min/max`, `aria-label` = `label`. Track `h-3` (12 px) `bg-ground-2 border-2 border-line-dim`, width fixed at `cells × 4 + 4` px (default `cells = 25` → 104 px; hub passes 40). Fill `<i data-fill>` `h-full` with inline `width: ${lit * 4}px`, `lit = Math.round(clamp(value/max) × cells)`; colour by `healthTone(value/max×100)`. Fill change: `transition: width 300ms steps(${|Δlit|})` (inline), none under reduced motion. Label left: VT323 20 `HP 80/100` (`label value/max`). Props `value`, `max = 100`, `label = 'HP'`, `cells = 25`, `reduced`.
```
 HP 80/100 [████████████████████░░░░░]   25 cells × 4 px; 20 lit
```

**`DayBar`** — three (`segments`) `HpBar`-style tracks in a row with `gap-1` (4 px), each `cells = 20` (84 px wide; 3 × 84 + 8 = 260 px, fits 320-px phones), no label, one `role="progressbar"` on the group (`aria-valuenow` seconds, max `segments × segmentSeconds`). Per-segment fill from `segmentFills()` (utils/progress). Right of the eyebrow: counter VT323 20 `20/30` (minutes, `minutesOf`); caption `font-body text-sm text-ink-1` = `${pct}%` or, when `met`, "Phòng hôm nay đã xong" in `text-growth`. Props `valueSeconds`, `segments = 3`, `segmentSeconds = 600`, `met = false`, `reduced`. Revive passes `segments=1 segmentSeconds=900 cells=40`.

**`QuestNode`** — `<li>` containing a 56×56 tile `<button>` (`bg-ground-1 border-2`, icon `PixelArt GLYPHS[book|scroll|sword]` at 32 px, by `task.task_type` vocabulary/reading/practice) and beside it title `font-body text-[17px]` + `(10 phút)` `text-sm text-ink-1`, action word right in VT323 20. Left of the tile an 8-px column: the `▶` cursor glyph for `current` (`retro-blink`), and below the tile the connector: `w-1 h-4` cells `bg-growth` (`connector='lit'`), `bg-line-dim` (`'dim'`), none. States: `done` border `line-lit`, `star` glyph overlaid top-right at 16 px in `torch`, "Đã xong", `aria-disabled`; `current` border `growth`, "Vào", `aria-current="step"`; `open` border `line-lit`, "Vào"; `locked` everything `text-ink-2`, `padlock` overlay, "Khoá", `aria-disabled` (still in the DOM, no `enter`). Props `task`, `index`, `state`, `connector = 'none'`, `reduced`; emits `enter(task.id)`.

**`MapNode`** — 40×40 `<button>` tile, day number VT323 20 centred; `cleared` `border-torch` + `star` 12 px top-right; `today` `border-growth bg-ground-2`, `aria-current="step"`, slot `sprite` above the tile (roadmap places `CompanionSprite size=32`); `partial` `growth` fill on the lower half; `missed` `border-ember` + `ring` glyph; `locked` `text-ink-2 border-line-dim` under a `ground-2`/60 % overlay, `aria-disabled`. Props `day`, `state`, `title` (→ `aria-label` "Ngày 9: {title}, {state word}"), `expanded?` (→ `aria-expanded`); emits `select(day)`.

**`Chest`** — 64 px `PixelArt` (`chestClosed`/`chestOpen`, 16-grid ×4) centred; when `open` becomes true the open frame replaces the closed one after 200 ms (one `setTimeout`, cleared on unmount; at once when `reduced`) and the list appears: `<ul>` of `items` rows, VT323 22 `text-torch`, `icon` string then `label`. Emits `opened`. `aria-live="polite"` on the list. Props `items: {icon: string, label: string}[]`, `open`, `reduced`.

**`Badge`** — inline `<span>`: 32-px `PixelArt` (`flame|shield|star` by `kind`) + optional count VT323 20 `x7`. `earned`: palette as drawn (`torch`); not earned: every non-`k` char → `line-dim`, `k` → transparent (outline only). `aria-label` "chuỗi 7 ngày" / "khiên" / "sao"; count also visible as text. Props `kind`, `count?`, `earned = true`.

**`SpeechBox`** — composes `RetroPanel speaker=name` with `portrait` = `CompanionSprite crop=face size=48 stage health` and the `line` in `font-body text-[17px] text-ink-0`. Typing: a `visibleChars` counter advanced every 30 ms (`setInterval`, cleared on unmount, restarted when `line` changes); tap anywhere on the panel or `reduced` shows all. The full line is always in a visually-hidden `<span>` (`aria-live` announces it once); the typed copy is `aria-hidden`. Emits `settled`. Props `line`, `name`, `stage`, `health`, `reduced`.

**`RetroToast`** + **`useRetroToast()`** (kit addition: `composables/useRetroToast.ts`, module-level queue — reason: "one at a time" needs one owner; pages call `show(line, tone?)`). The component renders the head of the queue as a one-line `RetroPanel tone` `fixed inset-x-4 bottom-[calc(80px+env(safe-area-inset-bottom))] max-w-md mx-auto`, `role="status"`, entering with `retro-rise`, auto-dismissed after 2000 ms, next item after. Props none; the composable exposes `queue`, `show`, `dismiss`. Pages mount `<RetroToast />` once (plan 2+).

**`StateBlock`** (restyled in place, `components/ui/StateBlock.vue`; props and `role="status"` unchanged — `revivePage`/`onboardingPage` tests click `[role="status"] button`) — `loading`: three 8×8 `bg-ground-2` cells lighting `line-lit` in turn (`retro-dots`), `aria-busy`, `aria-label="Đang tải"`; `empty`/`error`: `<p>` `font-body text-[17px] text-ink-0` (error prefixed by the `cross` glyph 16 px in `ember` — the red wall is gone) + `RetroButton` (`secondary` for error, `primary` for empty). No `text-alert`.

**`CountdownTimer`** (restyled in place, `components/learn/CountdownTimer.vue`; prop `remainingSeconds` unchanged — plan 3 changes the semantics) — `font-display text-xl tabular-nums text-ink-0`, `aria-live="off"`, the "Thời gian:" caption in `font-body text-sm text-ink-1`; `text-ember` at 0; no emoji.

## 6. Verification without a demo page
No committed preview. The reviewer verifies visually from a **throwaway route in the worktree**: `frontend/pages/_kit.vue` that mounts every component in every state on a `bg-ground-0` column (`npm run dev`, open `/_kit`), created by the executor, screenshotted for the plan's Notes, and **deleted before the final commit** (the acceptance list checks `git ls-files` has no `pages/_kit.vue`). Storybook is not added. Unit checks are Vitest with `@vue/test-utils` on happy-dom (existing convention), new files under `tests/unit/`:

| Test file | Pins |
|---|---|
| `tokens.test.ts` (rewritten) | v2 hex for every name in §1; aliases equal their targets; the WCAG pairs: `ink-0/1/2`, `line-lit`, `growth`, `torch`, `ember` on `ground-1` ≥ 4.5:1, `ground-0` on `growth/torch/ember` ≥ 4.5:1, `ink-0` on `ember-deep` ≥ 4.5:1 (a 15-line luminance helper in the test) |
| `fonts.test.ts` | `package.json` has `@fontsource/vt323` and `@fontsource/nunito`, not fraunces/source-sans; `nuxt.config.ts` text contains the three `vietnamese-*` css imports; `tailwind.config.ts` `fontFamily` names VT323/Nunito |
| `pixelArt.test.ts` | every `COMPANION` and `GLYPHS` entry is square; only palette chars; run-length rects render (`sprout` → `rect` count > 40 and < 400) |
| `CompanionSprite.test.ts` | each of the six stages sets `data-stage`; unknown stage → `sprout` + `ink-2` variable; `aria-label` has name, stage, HP; `react=down` rotates; `reduced` renders one frame (no `retro-*` animation class); `size=50` snaps to 32 |
| `HpBar.test.ts` | width is a multiple of 4 for values 0, 1, 33, 50, 99, 100; tone thresholds at 29/30/59/60; `aria-valuenow` |
| `DayBar.test.ts` | ports `SegmentedProgress.test.ts` (three segments, `20/30`, met caption, one 15-minute segment) |
| `RetroButton.test.ts` | height class `h-12`; three variants; loading keeps the label node and sets `aria-busy`; disabled has `aria-disabled` and stays in the DOM |
| `RetroPanel.test.ts` `QuestNode.test.ts` `MapNode.test.ts` `Chest.test.ts` `Badge.test.ts` `SpeechBox.test.ts` `RetroToast.test.ts` | one `it` per state in the kit table; emits (`enter`, `select`, `opened`, `settled`); locked nodes do not emit; typing reveals the full line on click; toast queue shows one at a time (fake timers) |
| `retroRadius.test.ts` | reads `components/retro/*.vue` and fails on any `rounded-` class other than `rounded-none`/`rounded-sm`, any `blur`, `bg-gradient`, `scale-`, `ease-`, or `dark:` |
| `StateBlock` / `CountdownTimer` | existing page tests keep passing unchanged |

## 7. Copy
Fixed strings owned by the kit components (Vietnamese, sentence case): `QuestNode` "Vào" · "Đã xong" · "Khoá" · "({n} phút)"; `DayBar` "Phòng hôm nay đã xong"; `HpBar` label "HP"; `StateBlock` `aria-label` "Đang tải"; `CompanionSprite` "giai đoạn {stage}, {n} HP", ", đã gục"; `Badge` "chuỗi {n} ngày" · "khiên" · "sao"; `MapNode` state words "đã xong" · "hôm nay" · "một phần" · "bỏ lỡ" · "khoá"; `CountdownTimer` "Thời gian:". Every other string is passed in by the page.

## 8. Self-critique
- Traded away: fluid bars. Fixed 4-px cells make `HpBar`/`DayBar` deterministic and testable but the day bar is 260 px on every phone rather than full-width; the hub eyebrow/counter row above it hides the gap. If a screen plan needs a wider bar, it passes `cells`, never a percentage.
- The face-on-the-pot decision makes the portrait crop and reactions trivial but means the plant's growth carries no expression change; the wilted stage's closed eyes and inverted smile are the only face change. Acceptable — the kit's stage feedback is size and colour.
- Plan 1 is visible after all (fonts, `StateBlock`, timer). The alternative — parallel `font-pixel` names and a second font payload — would cost ~150 KB of precache and a rename in plan 6; not worth it.
- Executor traps: `tokens` must stay flat strings (`ink-0`, not `ink: {0: …}`), or the alias `mute: tokens['ink-2']` and the contrast test break; `growth` changes hex — `tokens.test.ts` will fail until rewritten; do not import `NuxtLink` or `navigateTo` in `components/retro/`; `PixelArt` rows are strings, keep them in a `.ts` module not `.json` (type-checked chars); the `box-shadow` ring needs outer room — don't put a panel flush against `overflow-hidden`; `setInterval`/`setTimeout` in `SpeechBox`, `Chest`, `RetroToast` must clear on unmount or the test run leaks timers; `pages/_kit.vue` must not be committed.
- Review checks beyond the acceptance list: the sprout at 128 px matches the sketch pixel for pixel; the five other stages keep the pot rows identical (diff rows 22–31 of each `COMPANION` entry); `ls .output/public/_nuxt | grep -c woff2` is 9.

## Acceptance
- [ ] `tests/unit/tokens.test.ts` proves every text pair in §6 ≥ 4.5:1 with the v2 hex; `streak`/`alert`/`mute` equal `torch`/`ember`/`ink-2`; `paper`, `paper-dark`, `ink`, `card`, `btn` still exist.
- [ ] No file under `components/retro/` uses a radius above 2 px, a blur, a gradient, a scale tween, an easing curve or a `dark:` variant (`retroRadius.test.ts`).
- [ ] `HpBar` fill width is a multiple of 4 px for every value 0–100 and the tone changes at 30 and 60; `DayBar` renders three segments and the revive single segment.
- [ ] Every `RetroButton` variant is 48 px tall and ≥ 48 px wide; `QuestNode` tiles are 56 px; `MapNode` tiles are 40 px with a ≥ 44 px hit area (`p-0.5` on the wrapper).
- [ ] `package.json` lists `@fontsource/vt323` and `@fontsource/nunito` only; `nuxt.config.ts` imports the `vietnamese` (and `latin`, `latin-ext`) subsets for both; after `npm run build` the nine `woff2` files are in `.output/public/_nuxt/` and in the service-worker precache manifest.
- [ ] `CompanionSprite` renders all six stages, `down`, and an unknown stage (sprout in `ink-2`); the portrait crop at 48 px is a whole ×3; `aria-label` carries name, stage and HP.
- [ ] With `reduced` (and under `prefers-reduced-motion` in the browser) no `retro-*` animation runs and every reaction still shows its static frame; typing text, chest and toast appear at once.
- [ ] No v1 component is deleted or renamed; every page still builds; `revivePage`, `onboardingPage` and every other existing unit test pass unchanged.
- [ ] `pages/_kit.vue` (or any preview page) is not in `git ls-files`; the plan's Notes carry the reviewer's screenshots.
- [ ] `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` green, and CI green on the pushed branch.
