---
plan: harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md
verdict: pass
bugs: []
---
# Review — Retro kit amend: the kit paints its own dark ground (StateBlock/CountdownTimer readable in light scheme), palette fallback, OS reduced motion, static sprite reactions, torch focus and the 44 px MapNode

**Plan:** `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`
**Branch/worktree:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` / `.worktrees/retro-amend`
**Diff:** `git diff main...harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --stat`

## Context
- Reviewed at branch head **`2d9cab9`** (amend 1 ends at `2e0125c`; amend 2 sits on top). The reviewer used a fresh detached worktree `.worktrees/rv-retro` (== `origin/<branch>`), removed afterwards. The recorded `.worktrees/retro-amend` was not touched.
- **The branch is already on `main`.** PR #50 ("2026-09-27 [high] Retro UI kit … (+2 review amends)") squash-merged it as `a0ec6ed`. `git diff --stat origin/main origin/<branch> -- frontend` is empty. This review is therefore post-merge confirmation, and the plan's `merged: false` flag is stale.
- This amend was previously judged inside the parent's review 2 (`…-restyle-with-a-n-2.md`); this is its own review.

## Plan vs idea
The blocker idea (`restyled-stateblock-and-countdowntimer-put-near-white-ink-0-`) expected StateBlock/CountdownTimer text ≥ 4.5:1 on v1 surfaces in both schemes. Delivered. The five folded medium bugs (kit components inheriting v1 ink, palette fallback, sprite `reacted` under reduced motion, OS reduced motion reaching the kit, tile focus/pressed/44 px) are each delivered and reproduce in the browser (below).

## Code vs plan
Reviewer-run, `frontend/` @ `2d9cab9`, `NUXT_PUBLIC_API_BASE` unset:
```
npm ci → 0 · npm run lint → 0 · npm run typecheck → 0
npm run test:unit -- --run → Test Files 35 passed (35) · Tests 263 passed (263)
npm run build → 0 · woff2 in .output/public/_nuxt → 9 · woff2 refs in sw.js → 9
git diff --stat 03cb8b6 -- pages/ → only index.vue, learn/[id].vue, revive.vue (4 lines; these are amend 2's A9 lines, not this amend's)
git diff 03cb8b6 -- tests/unit/revivePage.test.ts tests/unit/onboardingPage.test.ts → empty
grep -rn 'function hexPalette' components → 0 · grep -rn 'dark:' components/retro → 0
grep -n "reduced: undefined" components/retro/{HpBar,DayBar,SpeechBox,Chest,QuestNode,CompanionSprite}.vue | wc -l → 6
git ls-files | grep -c _kit → 0 · amend-*.png in harness/reviews/retro-kit-screens → 6
gh run list --branch <branch> --limit 1 → completed success (run 36302673120, head 2d9cab9)
```
The executor's evidence reproduces, so there is no gate failure. Tasks 1–6 were followed. The one logged process deviation (a second worktree via `--force`) is harmless.

**Browser (Playwright, `nuxi dev :3106`, dead API, 375×812, `aelp.auth` seeded, throwaway `pages/_kit.vue` deleted after, `git status` clean):**

| Check (design "Acceptance (amend)") | light | dark |
|---|---|---|
| 1. StateBlock error text in a v1 `AppCard` | `rgb(244,241,255)` on `rgb(21,20,52)` **15.97** | 15.97 |
| 1. "Thử lại" button | 14.16 | 14.16 |
| 1. StateBlock empty | 15.97 | 15.97 |
| 1. CountdownTimer caption / digits / digits at 0 on bare `html` | 8.88 / 15.97 / 5.78 (`ember`) | same |
| 2. Loader cells | 3 × `rgb(201,196,244)` (`line-lit`) | same |
| 5. `svg rect` with black fill / with an undefined `--px-*` var | 0 / 0 of 1774 (all QuestNode types × states incl. an unknown type, 7 sprite stages incl. a bogus one) | 0 / 0 |
| 7. `reducedMotion:'reduce'`: `[data-fill]`/`[data-segment]` `transitionDuration` | `0s` (all); 0 running animations; SpeechBox whole line at load; Chest open | same |
| 8. Sprite `hit` click → `[data-log]` after 450 ms | `hit` | `hit` |
| 9. Real `Tab` focus on MapNode / QuestNode | `solid 2px rgb(242,168,59)` offset 2px, ring spread `0px` | same |
| 9. MapNode button / tile | 44×44 / 40×40 (all 5 states) | same |

Live pages `/`, `/learn/t1`, `/learn/x` in both schemes: every StateBlock/CountdownTimer text node is ≥ 5.78:1.

## Quality
- **Design and boundaries:** it matches A1–A5. `useReducedMotion` is a single module-level ref with one listener, which is appropriate given `ssr: false`. `CompanionSprite`'s hold timer is cleared on every `[react, isReduced]` change and on unmount.
- **Edge (observation, not filed):** if a caller never resets `react` to `idle` after `reacted`, an OS flip into reduced re-enters the hold and emits a second `reacted` for the same reaction. The component contract says the caller resets, and every current caller does, so no bug is filed.
- **Test honesty:** the reduced-motion tests use `vi.doMock` + `vi.resetModules` isolation, and `pixelPalette.test.ts` walks every kit state. Both are real assertions.
- **CODEMAP:** accurate for the paths touched.
- The earlier open lows (`retrobutton-keeps-its-4-px…`, `companionsprite-motion-is-off-the-pixel-grid…`, `retro-kit-small-contract-gaps…`, `retro-kit-tests-miss-boundaries…`) are out of this plan's scope, as the plan says, and remain queued.

## Bugs filed
None.

## Verdict
**pass.** The blocker and all five folded mediums are delivered and reproduce at `2d9cab9` in both colour schemes. CI is green, and the code is already on `main` via PR #50.
