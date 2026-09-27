---
plan: harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md
verdict: pass
bugs: []
---
# Review — Retro kit amend 2: pin the v1 aliases, ground-0 ink on growth/alert fills, scheme-split v1 green text, and SpeechBox mid-line reduced flip

**Plan:** `harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md`
**Branch/worktree:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` / `.worktrees/retro-amend`
**Diff:** `git diff main...harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --stat`

## Context
- Reviewed at branch head **`2d9cab9`**, the last commit of this amend, in a fresh detached worktree `.worktrees/rv-retro`, removed afterwards.
- **The branch is already on `main`.** PR #50 squash-merged it as `a0ec6ed`, and `main`'s `frontend/` is byte-identical to `2d9cab9`, so this review is post-merge confirmation. The plan's `merged: false` is stale.
- The parent's review 3 (`…-restyle-with-a-n-3.md`) already judged this amend; this is its own review.

## Plan vs idea
The blocker idea (`retro-v1-aliases-re-hue-…`) expected no v1 text pair to be worse than on `main` before the kit, white on `growth`/`alert` gone, and light `mute` back to AA. Delivered: the login/primary/avatar/"Học" pair is 11.69, the alert fills are 5.18, and light `mute` is 4.55/4.76. The folded medium (`retro-amend-tests-miss-live-reduced-motion-flips…`) is also delivered: the SpeechBox `watch(isReduced)` is in place, and there are live-flip tests for all six consumers plus the StateBlock `tokens.ember` assertion.

## Code vs plan
Reviewer-run, `frontend/` @ `2d9cab9`:
```
npm ci / lint / typecheck / build → 0 · test:unit → 35 files, 263 tests passed
grep -rnE 'bg-(growth|alert)[^/].*text-white' components pages → no hits
grep -rn 'text-white' components pages → no hits
grep -rn 'dark:' components/retro | wc -l → 0
grep -nE "streak: '#F59E0B'|alert: '#EF4444'|mute: '#64748B'|growth: '#3DE1B0'" tailwind.config.ts | wc -l → 4
git diff --stat 03cb8b6 -- pages/ → index.vue, learn/[id].vue, revive.vue only (class-only, 4 lines)
git ls-files | grep -c _kit → 0 · a2-*.png → 6 committed
gh run list --branch <branch> --limit 1 → success (36302673120 @ 2d9cab9)
```
Tasks 1–4 were followed. The `main.css` dark focus override was placed in a second `@media` block after the base rule, not inside the `:9` block. That deviation is justified, commented in the file, and the browser confirms it (below).

**Independent text walk** (Playwright, `nuxi dev :3106`, dead API, 375×812): `/login` unauthenticated, then `/`, `/revive`, `/learn/t1`, `/learn/x` authenticated, each in light and dark. `pet/status` was stubbed wilted (hp 0) or healthy, and `quests/daily` was stubbed with three open tasks or target-met with all done. Every visible element with its own text was checked (150 nodes); alpha backgrounds were composited.
- White text on any background: **0 nodes**.
- Focus ring, real `Tab` on `/login`: light `rgb(23,138,105)` 4px with a white 2px offset; dark `rgb(61,225,176)`.
- Sub-AA nodes, the complete list:

| Pair | Scheme | Ratio | Status |
|---|---|---|---|
| `text-mute` on `paper-dark` (login captions, "Lộ trình", "‹ Quay lại") | dark | 3.75 | A8 "= main" |
| `text-mute` on `ink` card (hub labels, `(10m)`, `[ ]`, locked rows, "Cây héo - 0%", "Mục tiêu…") | dark | 3.07 | A8 "= main" |
| `bg-streak/15 text-streak` pill "🔥 Streak: 2 ngày" | light | 1.84 | A8 "= main" |
| QuestRow done glyph `[✓]` `text-growth-deep`, 14 px | light | 4.31 | touched by this amend; design A9/A10 classes it as a glyph (3:1 floor), and "Xong" (`text-ink`, 14.63) carries the state |

Every one of these is on A8's accepted list or is the glyph floor. None is new.

## Quality
- **Minimal and correct:** 32 code lines. The token pins remove the regression at its source, and `tokens.test.ts` pins literal hex plus live-token floors, so a future re-hue fails loudly.
- **Goal wording vs design (observation):** the plan's Goal says "every pair this amend touches reaches AA", but the `[✓]` glyph it touched sits at 4.31:1 at 14 px. The design deliberately applies the 3:1 glyph floor here (A9 note "no kit token gives a green ≥ 4.5:1 on white"), and the glyph is `aria-hidden` next to an AA label, so no bug is filed. It is recorded so the owner knows the "AA" claim is scoped by the glyph exception.
- **Test honesty:** the two lows filed by review 3 still stand: the `v1Contrast` source guard only scans class attributes, and the dark focus-ring cascade plus the SpeechBox flip-back emit count are unguarded. No new gap was found.
- **CODEMAP:** no path changed.

## Bugs filed
None (review 3's two lows already cover the test gaps).

## Verdict
**pass.** The blocker and the folded medium are delivered and reproduce at `2d9cab9` in both schemes. The residual sub-AA pairs are exactly the design's accepted "= main" list plus the glyph-floor `[✓]`. CI is green, and the code is on `main` via PR #50.
