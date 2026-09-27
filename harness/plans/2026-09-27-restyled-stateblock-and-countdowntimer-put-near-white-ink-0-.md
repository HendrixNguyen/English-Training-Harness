---
idea: harness/ideas/_inbox/restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md
status: done
priority: high
merged: true
amends: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
design: harness/designs/retro-kit.md
branch: harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n
worktree: .worktrees/retro-amend
---
# Retro kit amend: the kit paints its own dark ground (StateBlock/CountdownTimer readable in light scheme), palette fallback, OS reduced motion, static sprite reactions, torch focus and the 44 px MapNode — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Amends:** `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (failed review `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`). **Lands on the same branch** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (@ `03cb8b6`). Every file below already exists on that branch, except the new tests and the one new composable. The recorded worktree `.worktrees/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` belongs to another session, so check the branch out in your own `.worktrees/<slug>` (no `-b`; it is the existing branch).

**Estimate:** ½ day (≈ 4 h). **Design:** `harness/designs/retro-kit.md` **§Addendum 2026-09-27 (A1–A7)**. Where the addendum and §3–§5 disagree, the addendum wins. Every task cites the section it implements. Inside `harness/UI-KIT.md` v2.

**Idea (blocker):** `harness/ideas/_inbox/restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`. **Folded in (medium, same branch):**
- `retro-kit-components-inherit-the-v1-page-text-colour-and-ret.md`
- `questnode-icon-palettes-miss-chars-their-glyphs-use-so-book-.md`
- `companionsprite-never-emits-reacted-when-animations-are-off-.md`
- `os-prefers-reduced-motion-does-not-reach-hpbar-daybar-speech.md`
- `questnode-and-mapnode-tiles-lack-the-torch-focus-ring-and-pr.md`

All of these are in `harness/ideas/_inbox/`. **Not in scope:** the four low review bugs (`retrobutton-keeps-its-4-px…`, `companionsprite-motion-is-off-the-pixel-grid…`, `retro-kit-small-contract-gaps…`, `retro-kit-tests-miss-boundaries…`). They are evaluated separately, so do not fix them here, even where you touch the same file.

**Goal:** every retro surface component, plus the restyled `StateBlock`/`CountdownTimer`, paints its own `ground-1` and `ink-*`, so it reads ≥ 4.5:1 on v1 light and dark surfaces today and on `ground-0` after plan 6. `PixelArt` always defines every `--px-*` variable. The OS reduced-motion setting reaches every kit component through `useReducedMotion()`, and the sprite's static reactions emit `reacted`. The quest and map tiles have the torch focus ring, the pressed inset and a MapNode hit area of at least 44 px. There are **no page, copy or token changes**.

**Root cause (evaluator, read on the branch):**
- `StateBlock.vue:28,33` and `CountdownTimer.vue:11-12` use kit inks (`ink-0` #F4F1FF, `ink-1`) without a kit ground, on top of v1 `html { bg-paper text-ink }` (`assets/css/main.css`) and `AppCard` `bg-white`.
- `RetroPanel.vue:30` sets no `text-*`, and in `band` mode drops the fill and border. `RetroToast.vue:18` uses `band`.
- `MapNode.vue:50,58,68` sets no ink, and the `h-1/2` partial fill paints over the number.
- `PixelArt.vue:29-33` defines only the passed chars. `QuestNode.vue:58` passes `k,l` to glyphs that use `i`/`d`/`T`, and the undefined `var()` resolves to black.
- `CompanionSprite.vue:50-54,64-66` emits only on `animationend`, and the reduced path applies no class.
- The `retro.css:77-84` reduced block matches only `retro-` classes. The bars' inline transitions and the `SpeechBox`/`Chest` JS timers read only the `reduced` prop.
- The tile buttons (`QuestNode.vue:50-57`, `MapNode.vue:48-51`) have no focus, active or hit-wrapper classes.

## Global Constraints
- Run npm from `frontend/`. `rg`/`timeout` are not installed, so use `grep -n`.
- No change under `frontend/pages/`, to `tailwind.config.ts`, or to any §7 copy string.
- No `dark:`, `rounded-*` above `sm`, `blur`, `bg-gradient`, `scale-` or `ease-` in `components/retro/` (`retroRadius.test.ts` unchanged and green).
- `StateBlock` keeps its props, `role="status"` on the root, and its button inside that root. `CountdownTimer` keeps `remainingSeconds` and `aria-live="off"`. `revivePage.test.ts` and `onboardingPage.test.ts` pass **unchanged**.
- Every new timer (the sprite's 300 ms hold) is cleared when `react` changes and on unmount.
- Every `reduced` prop is declared `reduced?: boolean` with `withDefaults({ reduced: undefined })`. Without the explicit `undefined` default, Vue casts an absent Boolean prop to `false` and the OS value is never read (A4 trap).
- `pages/_kit.vue` is throwaway and is **never committed** (`git ls-files | grep -c _kit` is 0).
- Tests first in each task. `npm run lint && npm run typecheck && npm run test:unit` green before every commit. One commit per task.

## File structure

| Path | Change | Design |
| --- | --- | --- |
| `frontend/components/retro/PixelArt.vue` | `palette` optional; define every `PALETTE` char first, then overlay | A3 |
| `frontend/components/retro/{QuestNode,MapNode,Chest}.vue`, `components/ui/StateBlock.vue` | delete the local `hexPalette`; pass overrides only | A3 |
| `frontend/components/ui/StateBlock.vue`, `components/learn/CountdownTimer.vue` | own `ground-1` plate / `RetroPanel`; `line-lit` loader cells | A1, A2 |
| `frontend/components/retro/RetroPanel.vue`, `RetroToast.vue` | always `border-2 border-line-lit bg-ground-1 text-ink-0` + ring; `band` = `p-2 w-full`; the toast drops `band` | A2 |
| `frontend/components/retro/QuestNode.vue` | tile ink; locked icon `ink-2`; torch focus; `active:` inset; static cursor under reduced | A2, A3, A4, A5 |
| `frontend/components/retro/MapNode.vue` | `<button class="group relative p-0.5">` wrapping a 40×40 `<span data-tile>`; ink; `z-10` number; `h-2` partial band; focus/`group-active:` | A2, A5 |
| `frontend/composables/useReducedMotion.ts` (new) | module-level `matchMedia` ref | A4 |
| `frontend/components/retro/{HpBar,DayBar,SpeechBox,Chest,CompanionSprite}.vue` | `isReduced = props.reduced ?? os`; bar fills `retro-anim`; sprite static frames + 300 ms emit | A4 |
| `frontend/tests/unit/` | new `pixelPalette`, `useReducedMotion`, `StateBlock`, `CountdownTimer` tests; extended kit tests | A6 |
| `harness/reviews/retro-kit-screens/amend-*.png` | the A7 screenshots, committed so the reviewer can see them | A7 |

## Tasks

### Task 1: `PixelArt` palette fallback: design A3 (fixes `questnode-icon-palettes-miss-chars…`)
**Files:** `components/retro/PixelArt.vue`, `QuestNode.vue`, `MapNode.vue`, `Chest.vue`, `components/ui/StateBlock.vue`, new `tests/unit/pixelPalette.test.ts`, `tests/unit/QuestNode.test.ts`.
- [ ] **Step 1 (tests first):** write `pixelPalette.test.ts` (A6). It mounts every `components/retro/*.vue` and `StateBlock` in every state: QuestNode ×4, MapNode ×5, Chest closed/open, Badge earned/unearned, CompanionSprite ×6 stages + `down` + unknown stage, StateBlock ×3. For every `rect[fill^="var(--px-"]`, the variable must be present and non-empty in its nearest `<svg>` inline style. In `QuestNode.test.ts`, the `locked` icon's `--px-l`/`--px-i`/`--px-d`/`--px-T` equal the `ink-2` hex and `--px-k` stays `ground-0`. Run the tests and see them fail (book/scroll/sword).
- [ ] **Step 2:** `PixelArt.vue`: `palette?: Partial<Record<string,string>>` default `{}`. `cssVars` first maps every `PALETTE` char to `tokens[PALETTE[c]]`, then overlays `palette`. Delete `hexPalette` from `QuestNode`, `MapNode`, `Chest`, `StateBlock`, and drop the `palette` props that only restated `PALETTE`. The locked QuestNode icon gets `{ every non-k char in PALETTE: tokens['ink-2'] }`. `CompanionSprite`/`Badge` overrides are unchanged.
- [ ] **Step 3:** `npm run test:unit`. Commit: `web: PixelArt defines every palette var; locked quest icon dims (retro amend)`.

### Task 2: Surfaces paint their own ground and ink: design A1, A2 (fixes the **blocker** and `retro-kit-components-inherit…`)
**Files:** `components/ui/StateBlock.vue`, `components/learn/CountdownTimer.vue`, `components/retro/RetroPanel.vue`, `RetroToast.vue`, `QuestNode.vue`, new `tests/unit/StateBlock.test.ts`, `tests/unit/CountdownTimer.test.ts`, `tests/unit/RetroPanel.test.ts`, `RetroToast.test.ts`, `QuestNode.test.ts`.
- [ ] **Step 1 (tests first), per A6 "Colour contracts" and "New StateBlock/CountdownTimer tests":**
  - `StateBlock` `loading`: the root has `bg-ground-1` and `aria-busy`/`aria-label="Đang tải"`, there are 3 cells each with `bg-line-lit`, and there is no `bg-ground-2`.
  - `StateBlock` `error`/`empty`: `[role="status"]` contains a `RetroPanel` whose ring colour is `ember`/`line-dim`, the `<p>` has `text-ink-0`, and `[role="status"] button` exists.
  - `CountdownTimer`: the root has `bg-ground-1 border-2 border-line-dim`, the caption has `text-ink-1`, and the digits have `text-ink-0`, or `text-ember` at 0.
  - `RetroPanel` plain/tone/`band`: the root has `bg-ground-1 text-ink-0 border-2 border-line-lit` and a ring `box-shadow`, and `band` has `p-2 w-full`.
  - `RetroToast`: the panel has `bg-ground-1` and a ring `box-shadow`.
  - The `QuestNode` button has `text-ink-0`, or `text-ink-2` when locked.
- [ ] **Step 2:** implement A2 exactly:
  - `StateBlock` loading: `inline-flex items-center gap-2 bg-ground-1 p-2` + `h-2 w-2 bg-line-lit retro-dots` cells at 0/150/300 ms.
  - `StateBlock` error/empty: `<div role="status">` → `<RetroPanel :tone="state === 'error' ? 'ember' : 'plain'">` → the unchanged `<p>` + `RetroButton`.
  - `CountdownTimer` root: `inline-flex items-baseline gap-1 border-2 border-line-dim bg-ground-1 px-2`.
  - `RetroPanel` `<section>`: always `border-2 border-line-lit bg-ground-1 text-ink-0`, with `band ? 'p-2 w-full' : 'p-4'`.
  - `RetroToast`: `<RetroPanel :tone>` without `band`.
  - `QuestNode` button: `text-ink-0` / `text-ink-2`.
- [ ] **Step 3:** `npm run test:unit`. `revivePage.test.ts` and `onboardingPage.test.ts` must pass untouched. Commit: `web: StateBlock, CountdownTimer and RetroPanel paint their own kit ground (retro amend, blocker)`.

### Task 3: Tiles: MapNode 44 px wrapper, ink, partial band; torch focus and pressed inset on both: design A2 (MapNode), A5 (fixes `questnode-and-mapnode-tiles-lack…` and the MapNode half of `retro-kit-components-inherit…`)
**Files:** `components/retro/MapNode.vue`, `QuestNode.vue`, `tests/unit/MapNode.test.ts`, `QuestNode.test.ts`.
- [ ] **Step 1 (tests first):**
  - Both tile buttons carry all five focus classes: `focus-visible:outline`, `focus-visible:outline-2`, `focus-visible:outline-offset-2`, `focus-visible:outline-torch`, `focus-visible:ring-0`, plus `focus-visible:ring-offset-0`.
  - QuestNode `open`/`current` has `active:translate-y-[2px] active:border-line-dim`, and `locked`/`done` do not.
  - The MapNode `<button>` has `group relative p-0.5`, and `[data-tile]` has `h-10 w-10 bg-ground-1` with `text-ink-0` (`text-ink-2` when locked) and `group-active:translate-y-[2px] group-active:border-line-dim` except when locked.
  - The MapNode number has `relative z-10`, and `[data-partial-fill]` has `h-2`, not `h-1/2`.
  - The existing aria-label, `aria-current`, `aria-expanded` and the "no select when locked" tests still pass.
- [ ] **Step 2:** implement A5 and the A2 MapNode bullet. Move the state border classes, number, glyphs, partial band, fog and `sprite` slot into `<span data-tile>`. The `<button>` keeps `type`, `aria-*`, `@click` and the focus classes.
- [ ] **Step 3:** `npm run test:unit`. Commit: `web: quest/map tiles get torch focus, pressed inset, 44px MapNode hit area (retro amend)`.

### Task 4: `useReducedMotion()` as the default for every `reduced` prop: design A4 (fixes `os-prefers-reduced-motion-does-not-reach…`)
**Files:** new `composables/useReducedMotion.ts`, `tests/unit/useReducedMotion.test.ts`; `components/retro/HpBar.vue`, `DayBar.vue`, `SpeechBox.vue`, `Chest.vue`, `QuestNode.vue`, `CompanionSprite.vue` (the prop default only; its static frames are Task 5); their tests.
- [ ] **Step 1 (tests first), per A6:**
  - `useReducedMotion.test.ts`: stub `window.matchMedia` with `matches: true` → `true`; dispatching `change` flips the ref; a missing `matchMedia` → `false`.
  - In each component test, add a `vi.mock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))` case with **no** `reduced` prop:
    - the `HpBar` `[data-fill]` and `DayBar` `[data-segment]` have no inline `transition` and carry `retro-anim`;
    - `SpeechBox` shows the whole line at mount;
    - `Chest` with `open` shows `chestOpen` and emits `opened` synchronously;
    - `QuestNode` `current` renders the cursor without `retro-blink`.
  - Explicit `reduced=false` restores the animated path. Put these cases in their own file, or use `vi.doMock`/`vi.resetModules`, so the mock does not leak into the existing non-reduced cases.
- [ ] **Step 2:** write the composable per A4: a module-level `ref`, initialised on first call, with one `change` listener, returning `Readonly<Ref<boolean>>`. In each of the six components:
  - declare `withDefaults({ reduced: undefined })`, set `const isReduced = computed(() => props.reduced ?? os.value)`, and replace every `props.reduced` read with it;
  - add `retro-anim` to the bar fills;
  - in QuestNode, always render the cursor for `current` and drop only `retro-blink` when reduced;
  - in SpeechBox, pass `:reduced="isReduced"` to the portrait sprite.
- [ ] **Step 3:** `npm run test:unit`. Commit: `web: useReducedMotion drives every retro component's reduced default (retro amend)`.

### Task 5: `CompanionSprite` static reactions and the `reacted` emit under reduced motion: design A4, the reaction table (fixes `companionsprite-never-emits-reacted…`)
**Files:** `components/retro/CompanionSprite.vue`, `tests/unit/CompanionSprite.test.ts`.
- [ ] **Step 1 (tests first, fake timers, `reduced`):**
  - `hit`: `data-frame="1"` and a `translateY(calc(var(--px) * -2px))` style during the hold; no emit at 299 ms; at 300 ms `reacted` is `[['hit']]` and `data-frame="0"`.
  - `levelup`: during the hold, the sprite svg's `--px-g` equals the `torch` hex; it emits at 300 ms.
  - `miss`: `data-frame="0"`; it emits at 300 ms.
  - Changing `react` mid-hold restarts the timer and emits only for the new value.
  - Unmounting mid-hold emits nothing, and `vi.getTimerCount()` is 0.
  - Replace the test at `:29` ("shows a static frame") with these assertions.
  - The non-reduced `animationend` path still emits.
- [ ] **Step 2:** implement per the A4 table: a `watch` on `react` (plus `isReduced`) starts one `setTimeout(300)`, and a `holding` ref drives the frame. On fire, return to base and emit `reacted(react)`. Clear the timer on the next change and in `onBeforeUnmount`. `idle` under reduced is the static base, and `down` is unchanged. The sprite never resets `react` itself.
- [ ] **Step 3:** `npm run test:unit`. Commit: `web: companion sprite holds a static reaction frame and emits reacted under reduced motion (retro amend)`.

### Task 6: Browser reproduction and screenshots: design A7
- [ ] **Step 1:** create the throwaway `pages/_kit.vue` per the original plan's Task 6 plus A7's **"v1 surfaces"** section, with **no** `bg-ground-0` wrapper: `StateBlock` loading/empty/error inside an `AppCard`, and `CountdownTimer` at 125 s and 0 on the bare `html` ground. Add reaction controls that write the emits into `<pre data-log>`. Run `npm run dev` with no backend, and use the faked `aelp.auth` localStorage session from the original plan's Notes to pass the route guard.
- [ ] **Step 2:** run A7 steps 1–5 with Playwright at 375×812, and record each measured value in this plan's **Execution summary**:
  - contrast ratios per scheme;
  - the loader cell colour;
  - the black-rect count;
  - `transitionDuration` and the `[data-log]` contents under `reducedMotion: 'reduce'`;
  - the outline colour and the MapNode 44×44 rect.
- [ ] **Step 3:** save `hub-light-375.png`, `hub-dark-375.png`, `learn-light-375.png`, `learn-dark-375.png`, `kit-v1-light-375.png` and `kit-v1-dark-375.png` as `harness/reviews/retro-kit-screens/amend-<name>` and **commit them**. The original run's screenshots were lost in a scratchpad. Reference them in Notes. `rm pages/_kit.vue`; `git ls-files | grep -c _kit` → 0.
- [ ] **Step 4:** run the full Verification below. Commit: `web: retro amend screenshots and evidence`. Push the branch and read CI with `gh run list --branch harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n`.

## Verification
```
cd frontend && npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build
npx vitest run tests/unit/pixelPalette.test.ts tests/unit/useReducedMotion.test.ts tests/unit/StateBlock.test.ts tests/unit/CountdownTimer.test.ts   # all pass
npx vitest run tests/unit/revivePage.test.ts tests/unit/onboardingPage.test.ts                                                                # pass, files unchanged:
git diff 03cb8b6 -- tests/unit/revivePage.test.ts tests/unit/onboardingPage.test.ts pages/ tailwind.config.ts | wc -l                          # 0
grep -n 'bg-ground-2' components/ui/StateBlock.vue                     # no hits
grep -rn 'function hexPalette' components/                             # no hits
grep -rn 'dark:' components/retro/                                     # no hits
grep -n "reduced: undefined" components/retro/{HpBar,DayBar,SpeechBox,Chest,QuestNode,CompanionSprite}.vue | wc -l   # 6
git ls-files | grep -c '_kit'                                          # 0
ls ../harness/reviews/retro-kit-screens/amend-*.png | wc -l            # 6
gh run list --branch harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --limit 1   # completed success
```

**Review reproduction, re-run** (from `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, which the amend must now pass). Use Playwright against `npm run dev` with no backend, at 375×812:
- [ ] `emulateMedia({ colorScheme: 'light' })` on the hub `/`: each `[role="status"]` error text's computed `color` is no longer `rgb(244, 241, 255)` on `rgb(255, 255, 255)`, and its ratio against the nearest painted ancestor is ≥ 4.5:1. Screenshot `amend-hub-light-375.png`.
- [ ] `emulateMedia({ colorScheme: 'light' })` on `/learn/x`: the "not in today" empty-state text is ≥ 4.5:1. Screenshot `amend-learn-light-375.png`.
- [ ] Repeat both under `colorScheme: 'dark'`: ≥ 4.5:1 for each.
- [ ] Browser context with `reducedMotion: 'reduce'` (OS reduced motion, **no** `reduced` prop on the kit page):
  - `[data-fill]` computed `transitionDuration` is `0s`, where the review measured `0.3s`;
  - the sprite "hit" click logs `hit` in `[data-log]` within 400 ms, where the review found it empty with `data-react="hit"` stuck.
- [ ] `svg.retro-pixel rect` with computed `fill: rgb(0, 0, 0)` that are not `k` rects: 0, where the review found 48.
- [ ] MapNode `getBoundingClientRect()` is 44×44, where the review measured 40×40. On keyboard focus the QuestNode/MapNode `outline-color` is `rgb(242, 168, 59)`, with no growth ring.

Design acceptance (`harness/designs/retro-kit.md` "Acceptance (amend)", verbatim):
- [ ] 1. Under `prefers-color-scheme: light` **and** `dark`, at 375 px with no backend, every `StateBlock` empty/error text on `/` and `/learn/:id` (inside the v1 `AppCard`) and the `CountdownTimer` digits and caption on `/_kit`'s bare v1 ground have a computed contrast ≥ 4.5:1 against their nearest painted background. Screenshots `hub-light-375.png`, `hub-dark-375.png`, `learn-light-375.png`, `learn-dark-375.png`, `kit-v1-light-375.png` and `kit-v1-dark-375.png` are in the plan's Notes.
- [ ] 2. `StateBlock` loading renders three cells whose computed background is `line-lit` (`rgb(201, 196, 244)`) on a `ground-1` plate; no `bg-ground-2` cell remains.
- [ ] 3. `StateBlock`, `CountdownTimer`, `RetroPanel` (plain, toned, band), `RetroToast`, and the `QuestNode`/`MapNode` tiles each set their own `bg-ground-1` (or state ground) and `text-ink-*` on their root. The toast has a `ground-1` fill, a 2 px `line-lit` border and the 2+2 px ring, and does not use `band`.
- [ ] 4. `MapNode` digits are `ink-0` (`ink-2` when locked), `z-10`, and clear of the `h-2` partial band.
- [ ] 5. No `<rect>` in any kit component or state references an undefined `--px-*` variable (`pixelPalette.test.ts`), and the browser shows zero non-outline `rgb(0, 0, 0)` rects. The locked `QuestNode` icon is drawn in `ink-2`.
- [ ] 6. `useReducedMotion()` exists. With it mocked `true` and no `reduced` prop: `HpBar`/`DayBar` fills have no transition; `SpeechBox` shows the whole line; `Chest` opens at once; the `QuestNode` cursor is static; explicit `reduced=false` overrides it.
- [ ] 7. Under emulated `reducedMotion: 'reduce'`, `[data-fill]`/`[data-segment]` computed `transitionDuration` is `0s`, and the SpeechBox and Chest appear at once.
- [ ] 8. Under reduced motion, `CompanionSprite` holds the static frame for 300 ms and then emits `reacted` (`hit`: plant raised 2 units, `data-frame="1"`; `levelup`: torch palette; `miss`: base). This holds both in fake-timer tests and in the browser `[data-log]` after 400 ms. No emit follows an unmount mid-hold.
- [ ] 9. The `QuestNode` and `MapNode` buttons show a 2 px `torch` outline with a 2 px offset on keyboard focus and no v1 growth ring. The actionable tiles drop 2 px with a `line-dim` border on `:active`, with no transition. The MapNode button measures 44×44 around a 40×40 tile.
- [ ] 10. No page file, copy string or token changes (`git diff` of `pages/`, `tailwind.config.ts` and §7's strings is empty); `retroRadius.test.ts` still passes with no `dark:` in `components/retro/`. `pages/_kit.vue` is not in `git ls-files`. `npm run lint`, `typecheck`, `test:unit` and `build` are green, and CI is green on the pushed branch.

Original plan's acceptance items that this amend must not regress: items 1–3, 5, 6 and 8 of `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` `## Verification` (tokens/contrast, radius guard, HpBar cells, fonts ×9 woff2 in `.output/public/_nuxt` and `sw.js`, the six sprite stages, no v1 component deleted).

## Notes
- Design decision (A1): **the kit forces its own dark ground; there are no light variants.** The kit is dark-native (UI-KIT "Dark/light"). Light variants would need `dark:` or a second palette, which the kit forbids, and plan 6 deletes `paper` anyway. Until plans 2–6 land, a light-scheme learner sees a small dark dialogue box inside a white v1 card. That is readable, and accepted.
- `useReducedMotion` is the one kit addition (A4). It supersedes §3's "reduced motion is CSS-only".

## Execution summary

Executed on the existing branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n`, in a fresh worktree `.worktrees/retro-amend` (git-force-added alongside the recorded worktree, which belongs to another session and was left untouched, per the orchestrator's instruction). Head commit **`2e0125c`**.

**Tasks (one commit each, tests-first):**
1. `41b0699` — `PixelArt` palette fallback (A3): every `--px-*` is defined from `PALETTE` before a caller's override; deleted `hexPalette()` in `QuestNode`/`MapNode`/`Chest`/`StateBlock`; the locked `QuestNode` icon dims every non-`k` char to `ink-2`. New `tests/unit/pixelPalette.test.ts` (25 cases across every kit component/state).
2. `196489b` — **Blocker fix** (A1/A2): `StateBlock`, `CountdownTimer`, `RetroPanel` (and by extension `RetroToast`, which stopped passing `band`) now paint their own `bg-ground-1`/`text-ink-*`/ring on their own root, so they read on the v1 white card / light-scheme `bg-paper`, not just the kit's own ground. New `tests/unit/StateBlock.test.ts`, `tests/unit/CountdownTimer.test.ts`; extended `RetroPanel.test.ts`, `RetroToast.test.ts`, `QuestNode.test.ts`.
3. `1ea70b5` — Tiles (A2 MapNode, A5): `MapNode`'s `<button>` is now a 44×44 `group relative p-0.5` hit-area wrapping a 40×40 `[data-tile]`; both tiles carry the five torch focus-visible classes and a pressed inset bound to their actionable states; the partial band is `h-2` (was `h-1/2`, which painted over the digits). Extended `MapNode.test.ts`, `QuestNode.test.ts`.
4. `2dffed4` — `useReducedMotion()` (A4): new module-level composable reading `matchMedia('(prefers-reduced-motion: reduce)')`; `HpBar`, `DayBar`, `SpeechBox`, `Chest`, `QuestNode`, `CompanionSprite` default `reduced` to the OS setting (`withDefaults({ reduced: undefined })` — the explicit `undefined` avoids Vue's absent-Boolean-prop-casts-to-`false` trap). New `tests/unit/useReducedMotion.test.ts`; extended `HpBar.test.ts`, `DayBar.test.ts`, `SpeechBox.test.ts`, `Chest.test.ts`, `QuestNode.test.ts` with `vi.doMock` + `vi.resetModules` per-test isolation.
5. `fc74546` — `CompanionSprite` static reactions (A4 table): under reduced motion, `hit`/`miss`/`levelup` hold one static frame for 300 ms (`watch` on `[react, isReduced]` driving one `setTimeout`, cleared on every change and on unmount) and then emit `reacted`, mirroring the non-reduced `animationend` path. Extended `CompanionSprite.test.ts` with fake-timer cases (hit/levelup/miss, restart-on-change, unmount-mid-hold).
6. `2e0125c` — Browser reproduction (A7) and `harness/CODEMAP.md` update (screenshots below).

**Deviations from the plan, logged:**
- None from the design's build contracts. One process deviation: per the amend plan's own note, the parent plan's recorded worktree "belongs to another session" and was still checked out (clean, on-branch) at `zen-burnell-b29ff5/.worktrees/retro-adventure-…`; rather than working inside a worktree another session might be using concurrently, I added a second worktree for the same branch with `git worktree add --force` at `.worktrees/retro-amend`, per this task's orchestrator instructions. Both worktrees now point at the same branch; the other session's worktree was not touched.
- `origin/main` has moved ~23 commits ahead of this branch's base (`03cb8b6`). Per the git rule ("only ever merge `main`/`origin/main`"), I did not merge it in: nothing in the plan or its Verification requires this branch to be current with `main`, and CI (which runs the workflow file at the pushed commit) is green without it. Flagging so the reviewer/owner can decide whether the eventual `main` merge needs a rebase.

**Runtime proof (step 8 of the executor role):**
- **Builds:** `npm run build` — clean production build; PWA precache manifest generated; no errors.
- **Whole suite, clean shell:** `npm run test:unit -- --run` → **34 files, 230 tests, all passing** (baseline before this plan: 30 files / 167 tests).
- **Boots and answers a real path:** `npx nuxi dev --port 3355` (no backend, `NUXT_PUBLIC_API_BASE=http://127.0.0.1:19999`) served `/`, `/learn/x` and the throwaway `/_kit`; a Playwright script drove real page loads, real `matchMedia`/`prefers-reduced-motion`, keyboard `Tab` focus and click reactions (see below). Dev server killed afterward; `pgrep -fl "nuxi dev --port 3355"` empty, port free.
- **Documented commands:** `npm ci`, `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` — all green, exactly as the plan's Verification lists them.
- **Bounded:** dev server polled with a 30 s cap via `curl --max-time 2` per attempt; `gh run watch --exit-status` (self-bounded).

**Plan's `## Verification` block, run and passing:**
```
npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build   # green (230 tests)
npx vitest run tests/unit/pixelPalette.test.ts tests/unit/useReducedMotion.test.ts tests/unit/StateBlock.test.ts tests/unit/CountdownTimer.test.ts   # 34/34 pass
npx vitest run tests/unit/revivePage.test.ts tests/unit/onboardingPage.test.ts   # 15/15 pass, files unchanged
git diff 03cb8b6 -- tests/unit/revivePage.test.ts tests/unit/onboardingPage.test.ts pages/ tailwind.config.ts | wc -l   # 0
grep -n 'bg-ground-2' components/ui/StateBlock.vue   # no hits
grep -rn 'function hexPalette' components/   # no hits
grep -rn 'dark:' components/retro/   # no hits
grep -n "reduced: undefined" components/retro/{HpBar,DayBar,SpeechBox,Chest,QuestNode,CompanionSprite}.vue | wc -l   # 6
git ls-files | grep -c '_kit'   # 0
ls ../harness/reviews/retro-kit-screens/amend-*.png | wc -l   # 6 (copied into ROOT, not this worktree/branch)
gh run list --branch harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --limit 1   # completed success, run 36298745809
```

**Review reproduction, re-run (Playwright, `npm run dev`, no backend, 375×812, `aelp.auth` seeded in `localStorage` before navigation), before → after:**
- Light-scheme hub `/` error text contrast: **1.05:1 → 15.97:1** (computed `rgb(244, 241, 255)` on `rgb(21, 20, 52)`, its own `ground-1`, not the v1 card's white). Same on `/learn/x`'s empty state and `/_kit`'s bare-`html` `CountdownTimer`. Dark scheme: identical ratio (the kit paints the same ground either way, design A1). Screenshots: `amend-hub-light-375.png`, `amend-hub-dark-375.png`, `amend-learn-light-375.png`, `amend-learn-dark-375.png`, `amend-kit-v1-light-375.png`, `amend-kit-v1-dark-375.png` — committed under `harness/reviews/retro-kit-screens/` **in ROOT** (per this task's routing; the branch only carries the `CODEMAP.md` update).
- Loader cell colour: `bg-ground-2` (`rgb(31, 29, 74)`) → `bg-line-lit` **`rgb(201, 196, 244)`**.
- `svg.retro-pixel rect` computed `fill: rgb(0, 0, 0)` (non-`k`): **48 → 0**.
- Reduced motion `[data-fill]`/`[data-segment]` `transitionDuration`: **0.3s → 0s**. Sprite `hit` click: `[data-log]` was empty/stuck → now shows `hit` within 400 ms.
- Keyboard focus `outline-color` on `QuestNode`/`MapNode`: → **`rgb(242, 168, 59)`** (torch), verified via real `Tab` key navigation (a programmatic `.focus()` does not trigger Chromium's `:focus-visible` after a prior mouse interaction — a script-methodology trap I hit and corrected). `box-shadow` at focus shows the v1 growth ring's colour channel present but its width/spread collapsed to `0px` by `ring-0`/`ring-offset-0` — no visible ring.
- `MapNode.getBoundingClientRect()`: **40×40 → 44×44** (width and height both measured 44).

**Design acceptance (amend), items 1–10:** all satisfied by the above (unit tests pin items 2–8 and 10; the browser run above pins items 1, 5's "zero non-outline black rects" and 9's focus/press/hit-area numbers).
**Original plan's acceptance items 1–3, 5, 6, 8 (not regressed):** covered by the unchanged `tokens.test.ts`, `retroRadius.test.ts`, `HpBar.test.ts`, `fonts.test.ts` (9 woff2 confirmed in `.output/public/_nuxt` and the build's precache manifest) and `CompanionSprite.test.ts`'s six-stage cases, all still green.

**CI:** green on the pushed branch — https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/36298745809 (`backend-unit`, `backend-integration`, `frontend`, `docker-images`, `harness-tooling` all ✓), head sha `2e0125c`.
