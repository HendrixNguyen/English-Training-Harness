---
idea: harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
status: approved
priority: high
merged: false
design: harness/designs/retro-kit.md
---
# Retro kit (plan 1 of 6): v2 tokens, VT323 + Nunito, `components/retro/*`, the companion sprite — no page changes — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F2** of 2026-09-26. **Estimate:** 4 h. **Branch:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n`.

**Design:** `harness/designs/retro-kit.md` (build contracts) inside `harness/UI-KIT.md` v2 (every token, type role, state and motion rule) and `harness/designs/retro-README.md` (build order). Every task below cites the design section it implements; `## Verification` repeats the design's acceptance list.
**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` — this plan is **plan 1 of 6**; plans 2–6 (hub, learning room, onboarding, roadmap, revive/settings) are written after this one is on `main`.

**Goal:** `frontend/` carries the retro kit as code — v2 tokens in `tailwind.config.ts`, the two faces self-hosted with Vietnamese subsets and precached, `assets/css/retro.css`, `components/retro/` (`PixelArt`, `RetroPanel`, `RetroButton`, `HpBar`, `DayBar`, `QuestNode`, `MapNode`, `Chest`, `Badge`, `CompanionSprite`, `SpeechBox`, `RetroToast` + `useRetroToast`), `StateBlock` and `CountdownTimer` restyled in place — with the Vitest pins the design names, and **no page changes**; every v1 component and page keeps building.

**Architecture:** design §0–§6. Nuxt-free components (`@vue/test-utils` on happy-dom, no `NuxtLink`/`navigateTo` in `components/retro/`); pixel art as SVG `<rect>` grids from string rows in `utils/pixelArt.ts`; reduced motion CSS-only plus a `reduced` prop for tests. v1 aliases (`streak`/`alert`/`mute`) keep v1 compiling; `ink`/`paper`/radii and the `html` rule stay until plan 6.

**Visible side effect, accepted (design §0 Q1, §8):** the display/body faces swap to VT323/Nunito on every page and `StateBlock`/`CountdownTimer` are restyled. Nothing else a learner sees changes.

## Global Constraints
- Work in `.worktrees/<slug>`; run npm from `frontend/`. `rg`/`timeout` not installed: `grep -n`.
- `tokens` in `tailwind.config.ts` stays a flat `Record<string,string>` (design §1, §8 traps); `growth` keeps its name with the v2 hex.
- No `rounded-*` other than `rounded-none`/`rounded-sm`, no blur, gradient, `scale-`, `ease-`, `dark:` in `components/retro/*` (design §5; `retroRadius.test.ts`).
- Every `setInterval`/`setTimeout` in `SpeechBox`, `Chest`, `RetroToast` is cleared on unmount.
- `pages/_kit.vue` is a throwaway for screenshots and is **never committed** (`git ls-files | grep _kit` empty before every commit).
- `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` green after every task; `tests/unit/revivePage.test.ts` and `onboardingPage.test.ts` pass unchanged.

## Review Focus
1. Contrast: `tokens.test.ts` computes WCAG ratios from the hex (design §6 table) — every listed text pair ≥ 4.5:1; `ink-2` is never used as text on a `ground-2` surface in the kit components (design §1 clarification).
2. Fonts: exactly nine `woff2` under `.output/public/_nuxt/` after `npm run build`, all in the generated service-worker precache manifest; `package.json` no longer lists fraunces/source-sans.
3. `HpBar` width snaps to 4-px cells for every value 0–100 (test iterates all 101); tone thresholds at 30 and 60.
4. `CompanionSprite`: the sprout at 128 px matches design §4's sketch pixel for pixel (compare the `COMPANION.sprout` rows to the doc); rows 22–31 identical across all six stages; unknown stage → sprout in `ink-2`; `react=down` rotates; `reduced` → no `retro-*` animation class.
5. `StateBlock` keeps `role="status"` and its props; `CountdownTimer` keeps `remainingSeconds` — the existing page tests are the proof.

## File structure

| Path | Change |
| --- | --- |
| `frontend/tailwind.config.ts` | v2 tokens + aliases; `fontFamily` VT323/Nunito (design §1) |
| `frontend/package.json` | `@fontsource/vt323`, `@fontsource/nunito`; remove fraunces, source-sans-3 (§2) |
| `frontend/nuxt.config.ts` | nine per-subset font CSS imports; `assets/css/retro.css` after `main.css` (§2, §3) |
| `frontend/assets/css/retro.css` | `.retro-pixel`, stepped keyframes, reduced-motion block (§3) |
| `frontend/utils/pixelArt.ts` | `PALETTE`, `COMPANION`, `GLYPHS` rows (§4) |
| `frontend/components/retro/PixelArt.vue` … `RetroToast.vue` (12 files) | the kit (§4, §5) |
| `frontend/composables/useRetroToast.ts` | one-at-a-time queue (§5) |
| `frontend/components/ui/StateBlock.vue`, `components/learn/CountdownTimer.vue` | restyled in place (§5) |
| `frontend/tests/unit/*.test.ts` | the table in design §6 |
| `harness/CODEMAP.md` | `shell` bullet: the kit, aliases, what plan 6 deletes |

## Tasks

### Task 1: Tokens and fonts — design §1, §2

**Files:** `tailwind.config.ts`, `package.json`, `nuxt.config.ts`, `tests/unit/tokens.test.ts`, `tests/unit/fonts.test.ts`.

- [ ] **Step 1 (tests first):** rewrite `tokens.test.ts`: every name in design §1 has its hex; `streak === torch`, `alert === ember`, `mute === ink-2`; `paper`, `paper-dark`, `ink`, `borderRadius.card/btn` still exist; a 15-line relative-luminance helper asserts the §6 pairs (`ink-0/1/2`, `line-lit`, `growth`, `torch`, `ember` on `ground-1`; `ground-0` on `growth`/`torch`/`ember`; `ink-0` on `ember-deep`) ≥ 4.5. New `fonts.test.ts`: `package.json` has `@fontsource/vt323` and `@fontsource/nunito` and neither `fraunces` nor `source-sans`; `nuxt.config.ts` source contains `vt323/vietnamese-400.css`, `nunito/vietnamese-400.css`, `nunito/vietnamese-700.css`; `tailwind.config.ts` `fontFamily.display[0] === 'VT323'`, `body[0] === 'Nunito'`.
- [ ] **Step 2:** `tailwind.config.ts` per design §1 (add the new names, `growth: '#3DE1B0'`, aliases as `streak: tokens.torch`-style references computed after the base object, `fontFamily` swap; radii untouched).
- [ ] **Step 3:** `npm uninstall @fontsource-variable/fraunces @fontsource/source-sans-3 && npm install @fontsource/vt323 @fontsource/nunito`; `ls node_modules/@fontsource/vt323 node_modules/@fontsource/nunito | grep -E 'vietnamese|latin'` — confirm the per-subset files (fallback per design §2 if absent, recorded in Notes). `nuxt.config.ts` `css:` → the nine per-subset files.
- [ ] **Step 4:** `npm run build && ls .output/public/_nuxt | grep -c '\.woff2$'` → 9; `grep -c 'woff2' .output/public/sw.js` ≥ 9. `npm run test:unit`. Commit: `web: retro kit tokens (v2 hex, v1 aliases) and VT323/Nunito with Vietnamese subsets`.

### Task 2: `retro.css`, `PixelArt`, `utils/pixelArt.ts`, `CompanionSprite` — design §3, §4

**Files:** `assets/css/retro.css`, `utils/pixelArt.ts`, `components/retro/PixelArt.vue`, `components/retro/CompanionSprite.vue`, `tests/unit/pixelArt.test.ts`, `tests/unit/CompanionSprite.test.ts`.

- [ ] **Step 1 (tests first):** `pixelArt.test.ts` — every `COMPANION[stage]` (six) and `GLYPHS[name]` (the 13 in §4) is square (N rows × N chars), uses only `PALETTE` chars; `COMPANION.sprout` equals the §4 sketch (paste the 32 rows into the test as the expected value); rows 22–31 identical across the six stages; `PixelArt` with `sprout` renders between 40 and 400 `rect`s and none for `.`. `CompanionSprite.test.ts` — the six stages set `data-stage`; unknown → `data-stage="sprout"` with `--px-g` = `ink-2` hex; `aria-label` `"Mầm, giai đoạn sprout, 80 HP"` for `name="Mầm"`; `react="down"` → `transform` contains `rotate(90`; `reduced` → no element with a class matching `/^retro-(breath|hop|shake|flash)/`; `size=50` → `width="32"`; `crop="face"` → `viewBox="8 16 16 16"`; `health=25` → `--px-g` = `ember`.
- [ ] **Step 2:** `retro.css` per design §3 (keyframes, `.retro-pixel`, the one reduced-motion block, `--px` unit); import after `main.css` in `nuxt.config.ts`.
- [ ] **Step 3:** `utils/pixelArt.ts`: `PALETTE` (char → token name), `COMPANION` (six 32-row sets per the §4 table; sprout verbatim from the sketch), `GLYPHS` (16-row glyphs + 8-row `cursor`). `PixelArt.vue`: props `rows`, `palette`, `size`, `label?`; run-length `<rect>`s; `var(--px-<char>)` fills with the variables set on the `<svg>`; `role="img"` + `aria-label` or `aria-hidden`.
- [ ] **Step 4:** `CompanionSprite.vue` per design §4 (props, `normalizeStage` reuse from `utils/plant.ts`, health tint via `healthTone()`, reactions table, `reacted` emit on `animationend`, `data-stage`/`data-react`/`data-frame`).
- [ ] **Step 5:** `npm run test:unit -- pixelArt CompanionSprite`. Commit: `web: PixelArt renderer, companion sprite data for six stages, stepped retro keyframes`.

### Task 3: Panels, buttons, bars — design §5 (`RetroPanel`, `RetroButton`, `HpBar`, `DayBar`)

**Files:** the four components, `tests/unit/RetroPanel.test.ts`, `RetroButton.test.ts`, `HpBar.test.ts`, `DayBar.test.ts`.

- [ ] **Step 1 (tests first):** `RetroPanel` — `speaker` renders the `<h2>` tab + `aria-labelledby` + `aria-live="polite"`; `tone=ember` puts `#B3261E`/`ember` in the outer ring style; `portrait` slot renders left. `RetroButton` — `h-12`; primary/secondary/danger classes; `loading` keeps the label node (`opacity-0`) and sets `aria-busy`, `disabled`; `disabled` → `aria-disabled` + still in DOM; `block` → `w-full`. `HpBar` — for every value 0..100 the fill `width` px is a multiple of 4 and ≤ `cells*4`; tone `ember` at 29, `torch` at 30 and 59, `growth` at 60; `role="meter"` with `aria-valuenow`; `reduced` → no `transition`. `DayBar` — port `SegmentedProgress.test.ts` (three segments, `20/30`, met caption "Phòng hôm nay đã xong", one 15-minute segment via `segments=1 segmentSeconds=900`).
- [ ] **Step 2:** Build the four per design §5 (DOM, classes, shadows, `steps()` transition, copy from §7).
- [ ] **Step 3:** `npm run test:unit -- RetroPanel RetroButton HpBar DayBar`. Commit: `web: RetroPanel, RetroButton, HpBar, DayBar`.

### Task 4: Nodes, chest, badge, speech, toast — design §5 (`QuestNode`, `MapNode`, `Chest`, `Badge`, `SpeechBox`, `RetroToast` + `useRetroToast`)

**Files:** the six components, `composables/useRetroToast.ts`, one test file each.

- [ ] **Step 1 (tests first):** `QuestNode` — one `it` per state (`done`/`current`/`open`/`locked`: border class, glyph, action word, `aria-current`/`aria-disabled`); `enter` emitted with `task.id` on click for `open`/`current`, **not** for `locked`/`done`; `connector` renders `bg-growth`/`bg-line-dim`/nothing. `MapNode` — the five states; `aria-label` "Ngày 9: {title}, hôm nay"; `select(day)` not emitted when `locked`; `expanded` → `aria-expanded`. `Chest` — closed frame, then open frame + list after 200 ms (fake timers), `opened` emitted, at once when `reduced`, `aria-live="polite"` on the list. `Badge` — `earned` palette vs outline-only; count text `x7`; `aria-label` "chuỗi 7 ngày". `SpeechBox` — typing reveals `visibleChars` over time (fake timers), click reveals all and emits `settled`, `reduced` shows all at once, the full line is in a visually-hidden `aria-live` span, `line` change restarts. `RetroToast` — `show('a'); show('b')` renders `a` only; after 2000 ms `b`; `role="status"`; `tone` on the panel.
- [ ] **Step 2:** Build per design §5; `useRetroToast.ts` = module-level `ref` queue with `show(line, tone?)`, `dismiss()`, `queue`.
- [ ] **Step 3:** `npm run test:unit`. Commit: `web: QuestNode, MapNode, Chest, Badge, SpeechBox, RetroToast`.

### Task 5: `StateBlock` and `CountdownTimer` restyled in place; the radius guard — design §5 (last two rows), §6

**Files:** `components/ui/StateBlock.vue`, `components/learn/CountdownTimer.vue`, `tests/unit/retroRadius.test.ts`.

- [ ] **Step 1 (test first):** `retroRadius.test.ts` reads every `components/retro/*.vue` (`fs.readdirSync`) and fails on `/rounded-(?!none|sm)/`, `blur`, `bg-gradient`, `scale-`, `ease-`, `dark:`.
- [ ] **Step 2:** `StateBlock`: props and `role="status"` unchanged; `loading` → three cells with `retro-dots`, `aria-busy`, `aria-label="Đang tải"`; `empty`/`error` → `font-body` `<p>` (+ `cross` glyph 16 px `ember` for error) + `RetroButton` (`secondary` for error, `primary` for empty); no `text-alert`. `CountdownTimer`: `font-display text-xl tabular-nums text-ink-0`, caption "Thời gian:" in `font-body text-sm text-ink-1`, `text-ember` at 0, `aria-live="off"`, no emoji; prop `remainingSeconds` unchanged.
- [ ] **Step 3:** `npm run test:unit` — including `revivePage.test.ts`, `onboardingPage.test.ts` unchanged. Commit: `web: StateBlock and CountdownTimer in the retro kit; radius guard test`.

### Task 6: Screenshots for the reviewer, CODEMAP

- [ ] **Step 1:** Create the throwaway `pages/_kit.vue` mounting every component in every state on a `bg-ground-0` column; `npm run dev`; screenshot `/_kit` (mobile 375 px and desktop) — save the PNGs outside the repo and reference them in this plan's **Notes** (paths + a one-line description each); `rm pages/_kit.vue`; `git status --short` shows no `_kit`.
- [ ] **Step 2:** CODEMAP `shell` bullet: the kit exists under `components/retro/` (list), `utils/pixelArt.ts`, `useRetroToast`, `retro.css`; v1 aliases and what plan 6 removes; fonts VT323/Nunito with Vietnamese subsets precached.
- [ ] **Step 3:** Commit: `codemap: retro kit`. Push.

## Verification
```
cd frontend && npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build
ls .output/public/_nuxt | grep -c '\.woff2$'                         # 9
grep -o '[A-Za-z0-9_.-]*\.woff2' .output/public/sw.js | sort -u | wc -l   # 9
git ls-files | grep -c '_kit'                                          # 0
ls components/retro | wc -l                                            # 12 .vue files
git push -u origin harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n   # CI: frontend + docker-images green
```
Design acceptance (`harness/designs/retro-kit.md` "Acceptance"):
- [ ] `tests/unit/tokens.test.ts` proves every text pair in §6 ≥ 4.5:1 with the v2 hex; `streak`/`alert`/`mute` equal `torch`/`ember`/`ink-2`; `paper`, `paper-dark`, `ink`, `card`, `btn` still exist.
- [ ] No file under `components/retro/` uses a radius above 2 px, a blur, a gradient, a scale tween, an easing curve or a `dark:` variant (`retroRadius.test.ts`).
- [ ] `HpBar` fill width is a multiple of 4 px for every value 0–100 and the tone changes at 30 and 60; `DayBar` renders three segments and the revive single segment.
- [ ] Every `RetroButton` variant is 48 px tall and ≥ 48 px wide; `QuestNode` tiles are 56 px; `MapNode` tiles are 40 px with a ≥ 44 px hit area.
- [ ] `package.json` lists `@fontsource/vt323` and `@fontsource/nunito` only; `nuxt.config.ts` imports the `vietnamese` (and `latin`, `latin-ext`) subsets for both; after `npm run build` the nine `woff2` files are in `.output/public/_nuxt/` and in the service-worker precache manifest.
- [ ] `CompanionSprite` renders all six stages, `down`, and an unknown stage (sprout in `ink-2`); the portrait crop at 48 px is a whole ×3; `aria-label` carries name, stage and HP.
- [ ] With `reduced` (and under `prefers-reduced-motion` in the browser) no `retro-*` animation runs and every reaction still shows its static frame; typing text, chest and toast appear at once.
- [ ] No v1 component is deleted or renamed; every page still builds; `revivePage`, `onboardingPage` and every other existing unit test pass unchanged.
- [ ] `pages/_kit.vue` (or any preview page) is not in `git ls-files`; the plan's Notes carry the reviewer's screenshots.
- [ ] `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` green, and CI green on the pushed branch.
