---
plan: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/v1contrast-source-guard-only-scans-class-attributes-so-scrip.md, harness/ideas/_inbox/retro-amend-2-test-gaps-dark-focus-ring-cascade-unguarded-sp.md]
---
# Re-review 3 (after amend 2) — Retro kit (plan 1 of 6): v2 tokens, VT323 + Nunito, `components/retro/*`, the companion sprite

**Plan:** `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Amends reviewed:** `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md` (done; passed review 2) and `harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md` (done; this review).
**Branch/worktree:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `2d9cab9`. Reviewed in `.worktrees/retro-amend`, which was clean and equal to `origin/<branch>` before and after.
**Design:** `harness/designs/retro-kit.md` Addendum 2 (A8–A12) and "Acceptance (amend 2)". **Kit:** `harness/UI-KIT.md` v2.
**Previous reviews:** `…-restyle-with-a-n.md` (fail), `…-restyle-with-a-n-2.md` (fail).
**Diff:** `git diff main...harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --stat`. Amend 2 alone is `2e0125c..2d9cab9`: 21 files, +360/−27, of which 32 lines are code and the rest tests.

## Plan vs idea
The blocker's expected output is delivered:
- No v1 text pair is worse than on `main`, measured on every text node of the four live pages, in both schemes and three data states.
- Every pair this amend touches reaches AA.
- `tokens.test.ts` now pins the v1 aliases and pairs, so a later re-hue fails a test instead of shipping silently.
- The folded medium bug is delivered: SpeechBox reveals the line on a mid-line OS flip, and the six consumers have live-flip tests.

Plan 1 still delivers the kit, not the screens (unchanged from review 1). The remaining sub-AA v1 pairs are exactly A8's "= main" list and leave with plans 2–6.

## Code vs plan
The reviewer ran this in `.worktrees/retro-amend/frontend` @ `2d9cab9`, with `NUXT_PUBLIC_API_BASE` unset:
```
npm ci && npm run lint (exit 0) && npm run typecheck (exit 0) && npm run test:unit && npm run build && npx nuxi generate   → all exit 0
  Test Files 35 passed (35) · Tests 263 passed (263)      (executor reported 263: reproduces)
ls .output/public/_nuxt | grep -c '\.woff2$' → 9 · woff2 refs in sw.js → 9
gh run view 36301861950 → headSha 2d9cab9, success: backend-unit, frontend, backend-integration, docker-images, harness-tooling
git merge-tree --write-tree origin/main origin/<branch> → ed72e51 (clean, rc 0; origin/main 0392e5d has no frontend change since the base 67ad0c0)
cli.py blockers --plan <parent> → rc 0
```
The executor's evidence reproduces, so this is not a gate failure. Task by task:
- **Task 1: followed.** The pins moved into `v1Only` with the A9 comment, the three A10 blocks were added, and the `retroRadius` guard was added.
- **Task 2: followed.** Every A9 row was applied verbatim. Among pages, `git diff 2e0125c -- pages/` touches only `index.vue:32`, `revive.vue:48,74` and `learn/[id].vue:90`, and each change is class-only. `grep -rn text-white components pages` finds nothing. There is no `dark:` in `components/retro/`, and `retro/` changes only in `SpeechBox.vue`.
- **Task 3: followed.** `SpeechBox.vue:52` has `watch(isReduced, r => { if (r) revealAll() })`, separate from the `line` watch.
- **Task 4: followed** (the six `a2-*.png` screenshots are committed).
- **One deviation, justified:** `main.css` puts the dark `:focus-visible` override in a second `@media` block *after* the base rule, not inside the `html` block at `:9`. Equal specificity means source order decides, so A9's literal placement would have lost in dark. The browser confirms the fix (below). A comment in `main.css` records why.

### A12 pairs, re-measured by the reviewer
Playwright ran against `nuxi dev` at 375×812 with no backend and `aelp.auth` seeded. `page.route` stubbed wilted `pet/status` and one `next` task, plus a "done" state (target met, one done task and two next tasks) and a revive-passed state. Everything else was aborted. The same script also ran against a scratch `origin/main` worktree (deleted after), so the "main" column below was measured, not quoted.

| Pair | Scheme | `main` (measured) | Review 2 @2e0125c | Now @2d9cab9 |
|---|---|---|---|---|
| `/login` button, `AppButton` primary | both | white on `rgb(16,185,129)` **2.54** | 1.67 | `rgb(11,10,31)` on `rgb(61,225,176)` **11.69** |
| `/` avatar, QuestRow "Học" (×2), `/learn` disabled "Đã hoàn thành" | both | 2.54 | 1.67 | **11.69** |
| `/` hub banner + "Cứu cây ngay" | both | white on `rgb(239,68,68)` **3.76** | 3.08 | `rgb(11,10,31)` on `rgb(239,68,68)` **5.18** |
| `/revive` alarm band, danger `AppButton` | both | 3.76 | 3.08 | **5.18** |
| `mute` on `paper`: `/login` captions, "Lộ trình", "‹ Quay lại" | light | 4.55 | 3.38 | **4.55** |
| `mute` on white: AppCard `h2`, "Máu cây:", `(10m)`, `[ ]` | light | 4.76 | 3.53 | **4.76** |
| `mute` on `paper-dark` / on `ink` | dark | 3.75 / 3.07 | 5.05 / 4.14 | 3.75 / 3.07 (= main, per A8) |
| SegmentedProgress met "30 / 30 phút" (24 px); revive "Cây đã hồi sinh!" (24 px) | light | 2.54 | 1.67 | `growth-deep` **4.31** (large, ≥ 3) |
| same | dark | 5.77 | 8.77 | **8.77** |
| QuestRow done glyph `[✓]` | light / dark | 2.54 / 5.77 | 1.67 / 8.77 | **4.31 / 8.77** |
| QuestRow "Xong"; `/learn` "✓ Đã hoàn thành" | light / dark | 2.54 / 5.77 | 1.67 / 8.77 | `ink` **14.63** / `growth` **8.77** |
| Focus ring (`/login` button and all four hub tabbables, real `Tab`) | light | `rgb(16,185,129)` 4px | growth | **`rgb(23,138,105)`** 4px, white 2px offset |
| same | dark | `rgb(16,185,129)` | growth | **`rgb(61,225,176)`**: the `main.css` ordering works |

White text on a `growth` or `alert` background: **0 elements** on `/login`, `/`, `/revive`, `/learn/t1` and `/learn/x`, in both schemes and all states.

### Page walk against `main` (step 3)
Every visible text node was diffed against the same node on `main`: **152 pairs**, across 5 pages, 2 schemes and 3 states. **One** pair is lower than on `main`: the CountdownTimer digits on `/learn` in dark drop from 17.06 to **15.97**. They now sit on the kit's own `ground-1` plate (amend 1, A2), which review 2 already accepted, and they stay far above AAA. The only other text difference is "⏱️ Thời gian:" → "Thời gian:", which design §5 specifies ("no emoji"). Otherwise every pair is equal to or better than `main`. The residual sub-AA pairs are exactly A8's pre-existing list: dark `mute`, light `text-streak` pill 1.84 (= main), and `text-alert` on its tint. No **new** sub-main pair exists.

### First review's findings, re-checked for regressions (throwaway `/_kit`, deleted after; `git status` clean)
| Finding | Now |
|---|---|
| Black icon rects | **0** of 1813 `svg rect` elements have `rgb(0,0,0)` fill (all quest types × 4 states, including an unknown type, and 6 stages plus a bogus stage), in light and dark |
| Reduced motion (context `reducedMotion: 'reduce'`) | 0 `[data-fill]`/`[data-segment]` with a non-zero transition; 0 running animations; sprite `reacted` = `hit` after ~360 ms |
| **Live mid-line flip (A11, new)** | SpeechBox typed 25 of 45 chars, `emulateMedia({reducedMotion:'reduce'})` → **45 of 45** within 60 ms; flipping back leaves it at 45 (no retype) |
| Focus ring on tiles | QuestNode ×16 and MapNode ×5: `outline solid 2px rgb(242,168,59)` (torch), ring spread `0px`, in both schemes; RetroButton: torch outline plus its own `growth-deep` drop shadow |
| MapNode hit area | all 5 states: button **44×44**, tile 40×40 |
| Fonts / Vietnamese | `document.fonts.check` true for `ặỆữ` in VT323 and Nunito; 9 woff2 built and precached |

### Design "Acceptance (amend 2)" 1–8
1 pass (4 hex greps; kit pairs green). 2 pass. 3 pass (grep empty). 4 pass. 5 pass. 6 pass (browser; screenshots `a2-*` from the executor and `r3-*` from the reviewer). 7 pass (the live flip is also shown in the browser). 8 pass.
**Must not regress:** amend-1 acceptance 1–9 and original Verification 1–3, 5, 6 and 8 were re-walked, and none regressed.

### Screenshots (`harness/reviews/retro-kit-screens/`)
`r3-login-{light,dark}-375.png`, `r3-hub-done-{light,dark}-375.png`, `r3-revive-{light,dark}-375.png`, `r3-revive-passed-{light,dark}-375.png`, `r3-learn-done-{light,dark}-375.png`.

## Quality
- **Design and boundaries:** clean and minimal, 32 code lines. Pinning `streak`/`alert`/`mute` back to their v1 hex removes the regression at the token level. `ground-0` on fills reuses the kit's own pair. The kit remains `dark:`-free and alias-free (guarded).
- **Correctness:** the A11 watch is placed correctly, and `revealAll` guards against a double emit. The dark ring override depends on source order. It is commented, but no test enforces it (low bug).
- **Test honesty** (the validator ran read-only, 110/110 on the touched files):
  - The `tokens.test.ts` pins use literal hex, and the floors use live tokens through `contrastRatio`, so both are real tests.
  - The StateBlock ember assertion is now specific.
  - The mock isolation (`doUnmock` → `resetModules`, `useRealTimers`) is sound.
  - The `v1Contrast` "acceptance item 3" source guard only matches `class="…"`/`:class="…"` text. It misses class maps built in `<script>` (the `AppButton`/`RoadmapNode` idiom). The reviewer and the validator both confirmed it stays green with `text-white` restored in `AppButton`. The per-component tests cover today's files (low bug).
  - The SpeechBox flip-back case asserts no emit count (low bug).
  - Three `NuxtLink` resolve warnings appear in `v1Contrast` (folded into the first low bug).
- **Performance and security:** nothing in scope. **CODEMAP:** no path changed, so no correction is needed.
- **Integration note (unchanged, not this branch's defect):** the pending growth-moment, roadmap-tree and name-your-plant branches must apply A8's per-file edits to their new lines when they merge after this one.

## Bugs filed
- low: `harness/ideas/_inbox/v1contrast-source-guard-only-scans-class-attributes-so-scrip.md`
- low: `harness/ideas/_inbox/retro-amend-2-test-gaps-dark-focus-ring-cascade-unguarded-sp.md`
- No blocker. Review 2's blocker (`retro-v1-aliases-…`) and its medium (`retro-amend-tests-miss-live-…`) are fixed and verified above. The low fontsource `unicode-range` bug and review 1's four lows stay open as planned.

## Verdict
**pass-with-bugs.** Amend 2 resolves review 2's blocker in the browser:
- The growth fill goes from 1.67 to 11.69, and the alert fill from 3.08 to 5.18.
- Light `mute` is back at `main`'s 4.55/4.76.
- Green text and the focus ring follow the scheme.
- No text pair on any live page is below `main`, apart from one AAA kit plate that was accepted earlier.

Nothing from review 1 or 2 regressed. Build, 263 tests, generate and CI are green; `merge-tree` with `main` is clean; `cli.py blockers --plan <parent>` exits 0. The two lows are test-hardening only. Merge command for the human: `/harness merge harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, or it goes into the daily PR. When that PR is cut, apply A8 to the pending frontend branches.
