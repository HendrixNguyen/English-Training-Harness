---
plan: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/retrobutton-keeps-its-4-px-depth-shadow-when-disabled-or-loa.md, harness/ideas/_inbox/companionsprite-motion-is-off-the-pixel-grid-at-every-size-b.md, harness/ideas/_inbox/retro-kit-small-contract-gaps-hpbar-shows-unclamped-values-b.md, harness/ideas/_inbox/retro-kit-tests-miss-boundaries-and-one-asserts-nothing-ches.md, harness/ideas/_inbox/fontsource-per-subset-css-has-no-unicode-range-so-all-nine-v.md, harness/ideas/_inbox/v1contrast-source-guard-only-scans-class-attributes-so-scrip.md, harness/ideas/_inbox/retro-amend-2-test-gaps-dark-focus-ring-cascade-unguarded-sp.md]
---
# Review — Retro kit (plan 1 of 6): v2 tokens, VT323 + Nunito, `components/retro/*`, the companion sprite — no page changes

**Plan:** `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Branch/worktree:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` / `.worktrees/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n`
**Diff:** `git diff main...harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --stat`

## Context
- A fresh whole-branch review at head **`2d9cab9`**: the original plan plus amend 1 (`2e0125c`) and amend 2 (`2d9cab9`).
- Prior reviews:
  - review 1 (`…-restyle-with-a-n.md`): **fail**, 1 blocker, 5 mediums, 4 lows;
  - review 2 (`…-n-2.md`): **fail**, a new blocker, 1 medium, 1 low;
  - review 3 (`…-n-3.md`): pass-with-bugs, 2 lows.
- The review was done in a fresh detached worktree `.worktrees/rv-retro`, removed afterwards.
- **Already on `main`:** PR #50 squash-merged the branch as `a0ec6ed`, and `main`'s `frontend/` equals `2d9cab9`. The plan's `merged: false` is stale.

## Plan vs idea
Plan 1 of 6 delivers the kit (v2 tokens, VT323 + Nunito with Vietnamese, `components/retro/*`, the companion sprite) and not the screens. That is still the right cut for the idea, which plans 2–6 complete.

## Code vs plan
Reviewer-run at `2d9cab9`:
```
npm ci / lint / typecheck / build → all exit 0
npm run test:unit -- --run → Test Files 35 passed (35) · Tests 263 passed (263)
.output/public/_nuxt/*.woff2 → 9 · woff2 refs in sw.js → 9
grep -rn 'dark:' components/retro → 0 · git ls-files | grep -c _kit → 0
gh run list --branch <branch> --limit 1 → completed success, run 36302673120 @ 2d9cab9
python3 tools/harness/cli.py blockers --plan <this plan> → rc 0
git merge-tree --write-tree origin/main origin/<branch> → clean (rc 0)
```

### Every finding behind the prior fail verdicts, re-checked in the browser
The walk used Playwright against `nuxi dev :3106` with a dead API, at 375×812, in light and dark, with a throwaway `/_kit` that was deleted afterwards.

| Finding (review) | Now |
|---|---|
| R1 **blocker**: StateBlock/CountdownTimer ink on v1 surfaces, 1.05:1 | Error/empty text is **15.97** and "Thử lại" 14.16. The timer caption is 8.88, the digits 15.97, and the digits at 0 are 5.78. Same in both schemes. **Fixed** (amend 1) |
| R1 medium: kit components inherit v1 ink / toast `band` | Every surface paints `ground-1` + `ink-*` (tests and browser). **Fixed** |
| R1 medium: palette chars missing, so black icon rects | 0 black and 0 undefined-var rects of 1774. **Fixed** |
| R1 medium: sprite never emits `reacted` under reduced motion | `[data-log]` = `hit` after 450 ms. **Fixed** |
| R1 medium: OS reduced motion doesn't reach the bars, SpeechBox or Chest | `transitionDuration` 0s everywhere, 0 running animations, line and chest shown at once. **Fixed** |
| R1 medium: tiles lack torch focus / pressed state / 44 px | torch `2px` outline with offset 2 and ring spread 0; MapNode 44×44 around a 40×40 tile. **Fixed** |
| R2 **blocker**: re-hued v1 aliases (white on growth 1.67, light mute 3.38–3.53, white on alert 3.08) | White text nodes: **0** of 150 on 5 live pages × 2 schemes × 2 data states. The growth fill is 11.69, alert 5.18, light mute 4.55/4.76. **Fixed** (amend 2) |
| R2 medium: live OS reduced-motion flips untested; SpeechBox mid-line flip | `watch(isReduced)` is in place, and six live-flip tests pass. **Fixed** |

The only sub-AA text left on live pages is design A8's accepted "= main" list: dark `mute` at 3.75/3.07 and the light streak pill at 1.84. The light `[✓]` glyph is 4.31, under the design's 3:1 glyph floor. The per-pair numbers are in `harness/reviews/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md`.

## Quality
Unchanged from review 3. The branch is clean and within the kit rules (no `dark:`, radius, or v1 aliases in `components/retro/`). The remaining open items are all low: test hardening and polish.

## Bugs filed
No new bugs. These open lows from earlier reviews are carried forward so the verdict lists them:
- `harness/ideas/_inbox/retrobutton-keeps-its-4-px-depth-shadow-when-disabled-or-loa.md` (selected)
- `harness/ideas/_inbox/companionsprite-motion-is-off-the-pixel-grid-at-every-size-b.md` (selected)
- `harness/ideas/_inbox/retro-kit-small-contract-gaps-hpbar-shows-unclamped-values-b.md` (selected)
- `harness/ideas/_inbox/retro-kit-tests-miss-boundaries-and-one-asserts-nothing-ches.md` (selected)
- `harness/ideas/_inbox/fontsource-per-subset-css-has-no-unicode-range-so-all-nine-v.md` (selected)
- `harness/ideas/_inbox/v1contrast-source-guard-only-scans-class-attributes-so-scrip.md` (proposed)
- `harness/ideas/_inbox/retro-amend-2-test-gaps-dark-focus-ring-cascade-unguarded-sp.md` (proposed)

## Verdict
**pass-with-bugs.** Every finding behind review 1's and review 2's fail verdicts is fixed and reproduces at `2d9cab9` in both schemes, and `cli.py blockers --plan` exits 0. Only low bugs remain open. The branch is already merged to `main` (PR #50, `a0ec6ed`), so it must not be merged again into a daily PR. The pending frontend branches still need design A8's per-file edits when they land.
