---
plan: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
verdict: fail
bugs: [harness/ideas/_inbox/restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md, harness/ideas/_inbox/retro-kit-components-inherit-the-v1-page-text-colour-and-ret.md, harness/ideas/_inbox/questnode-icon-palettes-miss-chars-their-glyphs-use-so-book-.md, harness/ideas/_inbox/companionsprite-never-emits-reacted-when-animations-are-off-.md, harness/ideas/_inbox/os-prefers-reduced-motion-does-not-reach-hpbar-daybar-speech.md, harness/ideas/_inbox/questnode-and-mapnode-tiles-lack-the-torch-focus-ring-and-pr.md, harness/ideas/_inbox/retrobutton-keeps-its-4-px-depth-shadow-when-disabled-or-loa.md, harness/ideas/_inbox/companionsprite-motion-is-off-the-pixel-grid-at-every-size-b.md, harness/ideas/_inbox/retro-kit-small-contract-gaps-hpbar-shows-unclamped-values-b.md, harness/ideas/_inbox/retro-kit-tests-miss-boundaries-and-one-asserts-nothing-ches.md]
---
# Review — Retro kit (plan 1 of 6): v2 tokens, VT323 + Nunito, `components/retro/*`, the companion sprite — no page changes

**Plan:** `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Branch/worktree:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` / `.worktrees/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n`
**Diff:** `git diff main...harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --stat`

## Plan vs idea
This is plan 1 of 6. The idea's *Expected output* is a retro restyle of every screen, and this plan delivers only the kit that plans 2–6 compose. So the plan is scoped correctly, and the question is whether the kit is a sound base. Mostly it is: the tokens, fonts and component set match `harness/UI-KIT.md` v2 and design §1–§5. There are two problems:
1. The in-place restyle of `StateBlock`/`CountdownTimer` makes a learner-visible regression **today**, on v1 pages.
2. Several kit contracts that plans 2–6 depend on are broken: reduced motion, `reacted`, palette coverage, and text colour.

## Code vs plan
Reviewed on a detached copy of `origin/<branch>` @ `03cb8b6` (`.worktrees/retro-code`), because the recorded worktree belongs to another session.

```
npm ci → ok · npm run lint → clean · npm run typecheck → clean
npm run test:unit → 30 files, 167 tests passed
npm run build → ok · npx nuxi generate → ok (.output/public)
ls .output/public/_nuxt | grep -c '\.woff2$' → 9
grep -o '[A-Za-z0-9_.-]*\.woff2' .output/public/sw.js | sort -u | wc -l → 9
git ls-files | grep -c _kit → 0 · ls components/retro | wc -l → 12
gh run list --branch <branch> → CI 36228032861 ok on 03cb8b6 (earlier 36227541592 red, superseded by the lockfile fix)
git merge-tree --write-tree origin/main origin/<branch> → clean, no conflicts
```
The executor's evidence reproduces. This is not a gate failure.

- **Task 1 (tokens, fonts):** followed. `tokens` is flat, the aliases equal their targets, and v1 names are kept. Nine per-subset CSS imports, and fraunces/source-sans are removed. In the browser the `vt323-vietnamese-400` and `nunito-vietnamese-400` woff2 files load, and `document.fonts.check` passes for `ặ`/`Ệ`.
- **Task 2 (retro.css, PixelArt, sprite):** followed, with the logged deviations 1–4. Unlogged: `retro-flash` is a CSS `filter`, not the palette swap. `--px` is a global 4, not per size. `down` lacks its translate. The sprite's `reacted` never fires without an animation (bug 4, bug 8).
- **Task 3 (panel, button, bars):** followed on the pinned tests. `RetroPanel` sets no text colour. The `RetroButton` disabled state keeps its shadow. The bars ignore OS reduced motion (bugs 2, 5, 7).
- **Task 4 (nodes, chest, badge, speech, toast):** built. The `QuestNode` icon palettes are incomplete, so pixels render black. Both tiles lack the torch focus ring and the pressed inset, and MapNode has no 44 px wrapper. The toast uses `band`, so it has no fill (bugs 2, 3, 6).
- **Task 5 (StateBlock, CountdownTimer, radius guard):** the guard and the page tests pass. The restyle sits on v1 surfaces and cannot be read in the light scheme (bug 1, blocker).
- **Task 6 (screens, CODEMAP):** the CODEMAP `shell` bullet is present and accurate. The executor's screenshots were in an ephemeral scratchpad, so the reviewer's own screenshots are below.

### Design acceptance, walked against the running app (`npm run dev`, a throwaway `/_kit` in the review copy, deleted after)
| # | Item | Result |
|---|---|---|
| 1 | tokens / contrast pairs / aliases / v1 names | **pass** (unit test; hex checked) |
| 2 | no radius > 2 px, blur, gradient, scale, easing or `dark:` in `components/retro/` | **pass** (`retroRadius.test.ts`). The `retro-flash` `filter` is not caught by the guard; see bug 8 |
| 3 | HpBar 4-px cells, tone at 30/60; DayBar 3 segments and the revive single segment | **pass** |
| 4 | RetroButton 48 px tall, ≥ 48 wide; QuestNode 56 px; MapNode 40 px with ≥ 44 px hit area | **fail**: buttons measure 48 px and QuestNode tiles 56 px, but MapNode is 40×40 with no hit wrapper (bug 6) |
| 5 | fonts: packages, vietnamese/latin/latin-ext imports, 9 woff2 in the build and the SW precache | **pass**. VT323 and Nunito render Vietnamese glyphs in the browser |
| 6 | CompanionSprite six stages, `down`, unknown in `ink-2`, 48-px face crop at ×3, aria-label | **pass**, with the logged fidelity gaps (bug 8) |
| 7 | reduced (prop and media query): no animation, static reaction frames, typing/chest/toast at once | **fail**: under the media query the bars still transition and typing/chest ignore it; reactions have no static frame, and `reacted` never fires (bugs 4, 5) |
| 8 | no v1 component deleted; every page builds; existing tests unchanged | **pass** |
| 9 | no `_kit` in `git ls-files`; screenshots in Notes | **pass / partial**: the executor's PNGs lived outside the repo; this review's are committed |
| 10 | lint, typecheck, test, build, CI green | **pass** |

Kit checks beyond the list: the kit panel text, the toast and the MapNode numbers inherit the v1 text colour (bug 2). 48 black pixels appear in the quest icons (bug 3).

### Screenshots (`harness/reviews/retro-kit-screens/`)
- `kit-mobile-375.png`: every kit component at 375 px on `ground-0`. It shows the black quest-icon pixels, the unreadable panel and map-node text, and the near-invisible loader.
- `hub-light-375.png`: the real hub `/` in the light scheme with no backend. The three error lines are #F4F1FF on white (the blocker).
- `learn-light-375.png`: `/learn/:id` in the light scheme. The "not in today" empty state has the same invisible text.

## Quality
- **Boundaries/conventions:** the kit is Nuxt-free as the design asks. It has no `NuxtLink`/`navigateTo`, and timers are cleared on unmount. The `hexPalette` helper is copy-pasted into four components. Hand-picking the palette per call is what caused bug 3; one shared helper that falls back to `PALETTE` would remove the whole class of bug.
- **Test honesty:** the suite pins the documented contracts, but it misses what this review found in the browser: colour inheritance, palette coverage, and reduced motion under the media query. `CompanionSprite.test.ts:29` claims "shows a static frame" but checks only that a class is absent. `Chest.test.ts:39-44` asserts nothing. `StateBlock`/`CountdownTimer` have no dedicated tests (bug 10, from the validator pass).
- **Performance:** `PixelArt` run-length merges rows (sprout is under 400 rects), so render cost is negligible.
- **Security:** nothing in scope.

## Bugs filed
- **BLOCKER (high, `blocks:` this plan)**: `harness/ideas/_inbox/restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`. StateBlock and CountdownTimer text is invisible on light-scheme v1 surfaces, live on every page.
- medium: `harness/ideas/_inbox/retro-kit-components-inherit-the-v1-page-text-colour-and-ret.md`. Panel, toast and MapNode text colour; the toast has no fill.
- medium: `harness/ideas/_inbox/questnode-icon-palettes-miss-chars-their-glyphs-use-so-book-.md`. Black pixels in the quest icons.
- medium: `harness/ideas/_inbox/companionsprite-never-emits-reacted-when-animations-are-off-.md`. `reacted` never fires, and there are no static reduced frames.
- medium: `harness/ideas/_inbox/os-prefers-reduced-motion-does-not-reach-hpbar-daybar-speech.md`. The media query does not reach the bars, typing or the chest.
- medium: `harness/ideas/_inbox/questnode-and-mapnode-tiles-lack-the-torch-focus-ring-and-pr.md`. Focus ring, pressed inset, and the MapNode 44 px hit area.
- low: `harness/ideas/_inbox/retrobutton-keeps-its-4-px-depth-shadow-when-disabled-or-loa.md`
- low: `harness/ideas/_inbox/companionsprite-motion-is-off-the-pixel-grid-at-every-size-b.md`
- low: `harness/ideas/_inbox/retro-kit-small-contract-gaps-hpbar-shows-unclamped-values-b.md`
- low: `harness/ideas/_inbox/retro-kit-tests-miss-boundaries-and-one-asserts-nothing-ches.md`

## Verdict
**fail.** The build, the tests and CI are green, and the branch merges cleanly with `main`. The branch still must not merge as it stands:
- The blocker makes every error, empty and timer line unreadable for light-scheme learners today.
- Acceptance items 4 (MapNode hit area) and 7 (reduced motion) fail against the running screen.

The fix is to amend this branch with a plan that sets `amends:` this one. It should cover at least the blocker. The medium kit bugs are cheaper to fix now, before plans 2–6 compose these components, than after.
