---
idea: harness/ideas/_inbox/retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md
status: approved
priority: high
merged: false
amends: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
design: harness/designs/retro-kit.md
branch: harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n
---
# Retro kit amend 2: pin the v1 aliases, ground-0 ink on growth/alert fills, scheme-split v1 green text, and SpeechBox mid-line reduced flip — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Amends:** `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`. This is the second amend; the first is `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md` (done). Failed re-review: `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n-2.md`.
- **Lands on the same branch:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n`, head `2e0125c`.
- Every file below already exists on that branch.
- Check the existing branch out (no `-b`) in your own `.worktrees/<slug>`. The recorded worktrees `.worktrees/retro-adventure-…` and `.worktrees/retro-amend` belong to other sessions.

**Estimate:** about 2 h.

**Design:** `harness/designs/retro-kit.md`, **Addendum 2 (A8–A12)** and "Acceptance (amend 2)".
- Where Addendum 2 disagrees with §1, §0 Q1, or amend acceptance 10 ("no page file changes"), Addendum 2 wins, for the A9 page lines only.
- Every task cites the section it implements.
- The work stays inside `harness/UI-KIT.md` v2.

**Idea (blocker):** `harness/ideas/_inbox/retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md`.

**Folded in (medium, same branch):** `harness/ideas/_inbox/retro-amend-tests-miss-live-reduced-motion-flips-and-the-sta.md` (design A11).

**Not in scope:**
- The low `fontsource-per-subset-css-has-no-unicode-range…` bug. It is selected and planned separately.
- The four earlier low review bugs.
- The pre-existing sub-AA v1 pairs A8 lists as "= main". Plans 2–6 replace those surfaces.
- The v1 non-text graphics A8 accepts: progress fills, selected-option `border-growth`, `PlantSvg`.
- Do not fix any of these here, even where you touch the same file.

**Goal:** no v1 text pair is worse than on `main`, every pair this amend touches reaches AA in both colour schemes, and `tokens.test.ts` pins the v1 pairs so a later re-hue cannot regress them silently. Kit hues and kit pairs are unchanged. SpeechBox reveals the line once on a mid-line OS reduced-motion flip.

**Root cause (evaluator, read at `2e0125c`):**
- `frontend/tailwind.config.ts:37-42` sets `streak: v2.torch`, `alert: v2.ember` and `mute: v2['ink-2']`, and re-hues `growth` to `#3DE1B0`.
- v1 components draw `text-white` on `bg-growth` and `bg-alert`, `text-mute` on paper and white, and `text-growth` on paper and white.
- `tests/unit/tokens.test.ts:41-45` asserts only alias equality plus kit pairs on `ground-*`.
- `SpeechBox.vue:51` watches only `props.line`.
- `StateBlock.test.ts:34` asserts only `style.length > 0`.

## Global Constraints
- Run npm from `frontend/`. `rg` and `timeout` are not installed, so use `grep -n`.
- **Tokens.**
  - Only `streak`, `alert` and `mute` change hex: `#F59E0B`, `#EF4444`, `#64748B`.
  - Every v2 hex is unchanged, including `growth #3DE1B0`.
  - No new token.
- **Pages.** The only page lines that change are the four in A9: `pages/index.vue:32`, `pages/revive.vue:48`, `pages/revive.vue:74` and `pages/learn/[id].vue:90`. They are class-only.
- **Copy and layout.** No copy or layout change anywhere. §7 strings are unchanged.
- **Kit.**
  - `components/retro/` does not change except `SpeechBox.vue`.
  - No `dark:` in `components/retro/`.
  - `dark:` variants are allowed only in the v1 files A9 lists.
- **Existing tests.**
  - `revivePage.test.ts` and `onboardingPage.test.ts` pass unchanged.
  - Every kit pair in `tokens.test.ts` stays asserted as-is.
- **Throwaway files.** `pages/_kit.vue` and any Playwright script are throwaway and are **never committed**: `git ls-files | grep -c _kit` is 0.
- **Commits.** Write tests first in each task. `npm run lint && npm run typecheck && npm run test:unit` must be green before every commit. One commit per task.

## File structure

| Path | Change | Design |
| --- | --- | --- |
| `frontend/tailwind.config.ts:37-42` | pin `streak`/`alert`/`mute` to v1 hex, move them into `v1Only`, update the comment | A8, A9 |
| `frontend/tests/unit/tokens.test.ts:41-45` | replace the alias `it` with the three A10 blocks | A10 |
| `frontend/tests/unit/retroRadius.test.ts` | guard: no `streak`/`alert`/`mute` class under `components/retro/` | A9, A10 |
| `frontend/assets/css/main.css:14-15` (+ dark block `:9`) | focus ring `ring-growth-deep`, dark `ring-growth` | A9 |
| `frontend/components/ui/AppButton.vue:16-17`, `components/AppHeader.vue:21`, `components/quest/QuestRow.vue:13,18,21`, `components/roadmap/RoadmapNode.vue:11`, `components/ui/SegmentedProgress.vue:24` | `text-ground-0` on fills; scheme-split green text | A9 |
| `frontend/pages/index.vue:32`, `pages/revive.vue:48,74`, `pages/learn/[id].vue:90` | same, class-only | A9 |
| `frontend/tests/unit/v1Contrast.test.ts` (new) | class contracts for the A9 v1 components | A9 |
| `frontend/components/retro/SpeechBox.vue:51` | `watch(isReduced, r => { if (r) revealAll() })` | A11 |
| `frontend/tests/unit/{Chest,DayBar,HpBar,QuestNode,SpeechBox,CompanionSprite,StateBlock}.test.ts` | live OS-flip tests, CompanionSprite OS-default block, ember assertion | A11 |
| `harness/reviews/retro-kit-screens/a2-*.png` | A12 screenshots, committed | A12 |

## Tasks

### Task 1: Pin the v1 aliases and pin the v1 pairs in tests (design A8, A9 row 1, A10)
**Files:** `frontend/tailwind.config.ts`, `frontend/tests/unit/tokens.test.ts`, `frontend/tests/unit/retroRadius.test.ts`.

- [ ] **Step 1 (tests first).**
  - In `tokens.test.ts`, delete the `it('keeps the v1 aliases pointing at their v2 targets')` block (lines 41-45).
  - Add the three A10 blocks, copied as written, using the existing `contrastRatio` helper:
    - `it('pins the v1-only aliases to their v1 hex (A8)')`: `alert` is `#EF4444`, `streak` is `#F59E0B` and `mute` is `#64748B`; each differs from `ember`/`torch`/`ink-2`.
    - `it('v1 text pairs meet the A8 floor')`: a `[fg, bg, min, label][]` table with every A10 row. White is the literal `'#FFFFFF'`. Put the label in the `expect` message so a failure names the pair.
    - `it('v1 pre-existing sub-AA pairs do not regress below main (A8)')`:
      - `mute` on `paper-dark` ≥ 3.7;
      - `mute` on `ink` ≥ 3.0;
      - `streak` on `#FFFFFF` ≥ 2.0;
      - `alert` on `#FFFFFF` ≥ 3.7.
  - In `retroRadius.test.ts`, add an `it.each(files)` that fails on `/\b(text|bg|border|fill|ring|outline)-(streak|alert|mute)\b/` in any `components/retro/` file (A9, last row).
  - Run `npx vitest run tests/unit/tokens.test.ts tests/unit/retroRadius.test.ts`. The pin test and the `ground-0`/`alert`, `mute`/`paper`, `mute`/white rows must fail, and the kit pairs must still pass.
- [ ] **Step 2.** In `tailwind.config.ts`:
  - Remove `streak`/`alert`/`mute` from the alias block and add them to `v1Only` as `streak: '#F59E0B'`, `alert: '#EF4444'`, `mute: '#64748B'`.
  - Replace the alias comment with: `// v1-only, pinned to v1 hex (design A8); the kit never uses them; deleted in plan 6.`
  - Keep `tokens` a flat record.
- [ ] **Step 3.** `npm run lint && npm run typecheck && npm run test:unit`. Commit: `web: pin v1 streak/alert/mute to v1 hex; tokens.test pins v1 pairs (retro amend 2)`.

### Task 2: v1 text on fills is ground-0; v1 green text and ring follow the scheme (design A9 rows 2-13)
**Files:**
- `frontend/assets/css/main.css`
- `frontend/components/ui/AppButton.vue`
- `frontend/components/AppHeader.vue`
- `frontend/components/quest/QuestRow.vue`
- `frontend/components/roadmap/RoadmapNode.vue`
- `frontend/components/ui/SegmentedProgress.vue`
- `frontend/pages/index.vue`
- `frontend/pages/revive.vue`
- `frontend/pages/learn/[id].vue`
- new `frontend/tests/unit/v1Contrast.test.ts`

- [ ] **Step 1 (tests first).** Write `v1Contrast.test.ts` (`@vue/test-utils`, happy-dom). Each case asserts the classes on the rendered element. Reuse the props that existing tests already use for these components (look at `tests/unit/` for fixtures).
  - `AppButton`:
    - `variant="primary"` has `bg-growth` and `text-ground-0` and no `text-white`;
    - `variant="danger"` has `bg-alert` and `text-ground-0` and no `text-white`.
  - `AppHeader`: the avatar element has `text-ground-0`.
  - `QuestRow`:
    - the next-task "Học" link or button has `text-ground-0`;
    - the done glyph has `text-growth-deep` and `dark:text-growth`;
    - the done label has `text-ink` and `dark:text-growth`.
  - `RoadmapNode` in the today state: `bg-growth` and `text-ground-0`.
  - `SegmentedProgress` when met: `text-growth-deep` and `dark:text-growth`.
  - Source guard: read `components/**/*.vue` and `pages/**/*.vue` with `node:fs` (as `retroRadius.test.ts` does). No class string may contain both `bg-growth` or `bg-alert` (not followed by `/`) and `text-white`. This mirrors Acceptance (amend 2) item 3.
  - Run it and see it fail.
- [ ] **Step 2.** Apply A9 rows 2-13 exactly as the design table writes them:
  - `main.css:14-15`: `:focus-visible { @apply outline-none ring-2 ring-growth-deep ring-offset-2 }`. Inside the existing dark `@media` block, add `:focus-visible { @apply ring-growth }`.
  - `AppButton.vue:16-17`:
    - `primary: 'bg-growth text-ground-0 hover:bg-growth/90'`;
    - `danger: 'bg-alert text-ground-0 hover:bg-alert/90'`.
  - `AppHeader.vue:21`: `text-white` → `text-ground-0`.
  - `QuestRow.vue`:
    - `:13` `'text-growth'` → `'text-growth-deep dark:text-growth'`;
    - `:18` `text-white` → `text-ground-0`;
    - `:21` `'text-growth'` → `'text-ink dark:text-growth'`.
  - `RoadmapNode.vue:11`: `today: 'border-growth bg-growth text-ground-0 scale-105'`.
  - `SegmentedProgress.vue:24`: `met ? 'text-growth-deep dark:text-growth' : ''`.
  - `pages/index.vue:32`: `text-white` → `text-ground-0`.
  - `pages/revive.vue`:
    - `:48` `text-growth` → `text-growth-deep dark:text-growth`;
    - `:74` `text-white` → `text-ground-0`.
  - `pages/learn/[id].vue:90`: `text-growth` → `text-ink dark:text-growth`.
  - If a line number has drifted, match on the quoted class string. If a quoted string is not found, stop and log it rather than guess.
- [ ] **Step 3.** Check the test and the diff:
  - `npm run lint && npm run typecheck && npm run test:unit`. `revivePage.test.ts` and `onboardingPage.test.ts` pass unchanged.
  - `git diff --stat 2e0125c -- pages/` lists only `index.vue`, `revive.vue` and `learn/[id].vue`.
  - `git diff 2e0125c -- pages/` shows only class-attribute changes.
  - Commit: `web: v1 text on growth/alert fills is ground-0; v1 green text and focus ring follow the scheme (retro amend 2)`.

### Task 3: SpeechBox mid-line reduced flip, and live OS-flip tests (design A11; the folded medium bug)
**Files:**
- `frontend/components/retro/SpeechBox.vue`
- `frontend/tests/unit/SpeechBox.test.ts`
- `Chest.test.ts`
- `DayBar.test.ts`
- `HpBar.test.ts`
- `QuestNode.test.ts`
- `CompanionSprite.test.ts`
- `StateBlock.test.ts`

- [ ] **Step 1 (tests first).** In each consumer test, add a `describe('live OS flip')`. It mocks `~/composables/useReducedMotion` to return one shared `ref(false)` the test can reach. Use the same `vi.doMock` + `vi.resetModules` pattern the first amend used.
  - For each of Chest, DayBar, HpBar, QuestNode, SpeechBox and CompanionSprite: mount **without** `reduced`, set the ref to `true`, `await nextTick()`, then assert the reduced render:
    - HpBar/DayBar: no inline `transition` on `[data-fill]`/`[data-segment]`.
    - Chest: open, and `opened` emitted.
    - QuestNode `current`: the cursor has no `retro-blink`.
    - CompanionSprite: under fake timers, `react='hit'` holds `data-frame="1"`, then emits `reacted` `['hit']` at 300 ms.
  - SpeechBox, under fake timers:
    - Advance 2 characters (60 ms) and flip. The whole line is shown and `settled` has been emitted exactly once.
    - In a second case, let the line settle, then flip. `settled` is still emitted exactly once.
    - In a third case, flip `true` → `false` after the reveal. Nothing re-types, and a new `line` types normally.
  - CompanionSprite OS-default block:
    - with the OS set to `true` and no prop: hold, then `reacted`;
    - with an explicit `reduced=false` and the OS set to `true`: the non-reduced animation class is applied (`retro-hop` for `hit`).
  - `StateBlock.test.ts:26-38`: the error state's `RetroPanel` root `style` contains `tokens.ember` (import `tokens` from `~/tailwind.config`), and the empty state's does not.
  - Run the tests. Only the SpeechBox mid-line case should fail. If any other consumer fails, stop and record it in Notes; do not widen the scope.
- [ ] **Step 2.** In `SpeechBox.vue`, directly after the `watch(() => props.line, startTyping, { immediate: true })` line (`:51`), add `watch(isReduced, (r) => { if (r) revealAll() })`. Do **not** add `isReduced` to the `line` watch.
- [ ] **Step 3.** `npm run lint && npm run typecheck && npm run test:unit`. Commit: `web: SpeechBox reveals the line on a mid-line reduced-motion flip; live OS-flip tests (retro amend 2)`.

### Task 4: Browser reproduction and screenshots (design A12; the review's reproduction steps)
**Files:** `harness/reviews/retro-kit-screens/a2-{login,hub,revive}-{light,dark}-375.png`, the plan's `## Notes`, and `harness/CODEMAP.md` only if a path it lists changed (none is expected).
- [ ] Run `npx nuxi dev --port <free port>` in the worktree with no backend and `NUXT_PUBLIC_API_BASE` pointed at a dead port. A throwaway Playwright script (not committed) runs every step in the `## Verification` "Review reproduction, re-run" list in both colour schemes and saves the six screenshots.
- [ ] Record each measured `color` / `background-color` / ratio in `## Notes`.
- [ ] Kill the dev server and confirm the port is free.
- [ ] Commit the screenshots with the plan update: `harness: retro amend 2 browser reproduction`.

## Verification
```
cd frontend && npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build
npx vitest run tests/unit/tokens.test.ts tests/unit/retroRadius.test.ts tests/unit/v1Contrast.test.ts tests/unit/SpeechBox.test.ts tests/unit/StateBlock.test.ts   # all pass
npx vitest run tests/unit/revivePage.test.ts tests/unit/onboardingPage.test.ts                       # pass, files unchanged:
git diff 2e0125c -- tests/unit/revivePage.test.ts tests/unit/onboardingPage.test.ts | wc -l         # 0
git diff --stat 2e0125c -- pages/                                                                    # only index.vue, revive.vue, learn/[id].vue
git diff 2e0125c -- components/retro/ | grep '^[-+][^-+]' | grep -v SpeechBox                        # only SpeechBox.vue lines
grep -rnE 'bg-(growth|alert)[^/].*text-white' components pages                                       # no hits
grep -rnE '\b(text|bg|border|fill|ring|outline)-(streak|alert|mute)\b' components/retro/             # no hits
grep -rn 'dark:' components/retro/                                                                   # no hits
grep -nE "streak: '#F59E0B'|alert: '#EF4444'|mute: '#64748B'|growth: '#3DE1B0'" tailwind.config.ts | wc -l   # 4
git ls-files | grep -c '_kit'                                                                        # 0
ls ../harness/reviews/retro-kit-screens/a2-*.png | wc -l                                             # 6
gh run list --branch harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --limit 1   # completed success
```

**Review reproduction, re-run.** These come from `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n-2.md` and the blocker's Evidence, which this amend must now pass. The setup follows design A12:
- Playwright against `nuxi dev`, with no backend, at 375×812, and `aelp.auth` seeded.
- `page.route` stubs:
  - `/api/v1/pet/status` → wilted (`health_points: 0`);
  - `/api/v1/quests/daily` → one `next` task;
  - everything else aborted.
- Run each check under `emulateMedia({ colorScheme: 'light' })` and again under `'dark'`.
- For each check, read the element's computed `color` and the first non-transparent ancestor `background-color`, then compute the WCAG ratio.

- [ ] `/login` button (the review measured `rgb(255,255,255)` on `rgb(61,225,176)` = 1.67:1): now `rgb(11, 10, 31)` on `rgb(61, 225, 176)`, ≥ 11.6:1, in both schemes.
- [ ] `/login` caption `p.text-mute`:
  - light: `rgb(100, 116, 139)` on `rgb(248, 250, 252)`, ≥ 4.5:1. The review measured `rgb(135,131,181)`, 3.38:1.
  - dark: `rgb(100, 116, 139)` on `rgb(15, 23, 42)`, 3.75:1 (= main).
- [ ] `/` avatar and QuestRow "Học": the same pair as the login button, ≥ 11.6:1.
- [ ] `/` hub banner: `rgb(11, 10, 31)` on `rgb(239, 68, 68)`, ≥ 5.1:1.
- [ ] `/` AppCard `h2.text-mute`:
  - light: on `rgb(255, 255, 255)`, ≥ 4.5:1. The review measured 3.53:1.
  - dark: on `rgb(30, 41, 59)`, 3.07:1 (= main).
- [ ] `/revive`: the alarm band and the danger `AppButton` each read `rgb(11, 10, 31)` on `rgb(239, 68, 68)`, ≥ 5.1:1. The review measured white at 3.08:1.
- [ ] `/learn/<stubbed id>`: "‹ Quay lại" is `rgb(100, 116, 139)` on `rgb(248, 250, 252)` in light.
- [ ] Tab onto the `/login` button. The computed `box-shadow` contains `rgb(23, 138, 105)` in light and `rgb(61, 225, 176)` in dark.
- [ ] No element with a computed `color` of `rgb(255, 255, 255)` has a `growth` or `alert` painted background on any of the four pages.
- [ ] The six `a2-*.png` screenshots are committed.

Design acceptance (`harness/designs/retro-kit.md` "Acceptance (amend 2)", verbatim):
- [ ] 1. `tailwind.config.ts` has `streak #F59E0B`, `alert #EF4444`, `mute #64748B`; every v2 hex, including `growth #3DE1B0`, is unchanged, and all kit pairs in `tokens.test.ts` still pass.
- [ ] 2. Every A10 pair passes in `tokens.test.ts`, and the non-regression block passes.
- [ ] 3. No `text-white` remains on a `bg-growth`/`bg-alert` element under `components/` or `pages/` (`grep -rnE 'bg-(growth|alert)[^/].*text-white' frontend/components frontend/pages` prints nothing).
- [ ] 4. Every A9 row is applied, and no other page line changes (`git diff --stat` touches only `pages/index.vue`, `pages/revive.vue` and `pages/learn/[id].vue` among pages, and only the listed lines). No copy or layout changes.
- [ ] 5. `retroRadius.test.ts` also fails on any `streak`/`alert`/`mute` class under `components/retro/`, and passes.
- [ ] 6. A12 steps 1–5 give the listed computed colours and ratios in both schemes, and the screenshots are in Notes.
- [ ] 7. SpeechBox: a mid-line flip to reduced reveals the line and emits `settled` once, with no emit after settling (A11). Each of the six consumers has a live-flip test; `CompanionSprite` has the OS-default block; the `StateBlock` error test asserts `tokens.ember`.
- [ ] 8. `npm run lint`, `typecheck`, `test:unit` and `build` are green, and CI is green on the pushed branch.

**Must not regress:**
- the first amend's "Acceptance (amend)" items 1–9 (kit surfaces, palette fallback, reduced motion, focus/44 px);
- items 1–3, 5, 6 and 8 of the original plan's `## Verification` (the fonts: 9 woff2 in `.output/public/_nuxt` and `sw.js`).

## Notes
- **Design decision (A8).** The v1 alias names go back to v1 hex because the kit never draws them. `growth` stays v2, so v1 text on it uses the kit's own `ground-0` pair.
- **Floor.** "No v1 pair worse than `main`, and every touched pair reaches AA." Full AA on every v1 pair in both schemes is not reachable with one hex per token. The remaining pre-existing sub-AA pairs (dark `mute`, light `text-streak`, `text-alert` on its `/10` tint) are at their `main` value and leave with plans 2–6.
- **Known hover gap.** The danger button's hover over the dark `ink` card is 4.47:1. It is hover-only, and `main` measures 4.36:1 there.
- **Pending frontend branches.** Growth moment, roadmap tree, name-your-plant and streak shield: whichever of each pair merges second applies A8 to its new lines. The per-file edits are listed in A8 under "Pending branches". The daily integration branch needs that step. It is not part of this plan.
