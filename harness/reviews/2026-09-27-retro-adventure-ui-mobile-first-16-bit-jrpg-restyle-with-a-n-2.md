---
plan: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
verdict: fail
bugs: [harness/ideas/_inbox/retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md, harness/ideas/_inbox/retro-amend-tests-miss-live-reduced-motion-flips-and-the-sta.md, harness/ideas/_inbox/fontsource-per-subset-css-has-no-unicode-range-so-all-nine-v.md]
---
# Re-review (after amend) — Retro kit (plan 1 of 6): v2 tokens, VT323 + Nunito, `components/retro/*`, the companion sprite — no page changes

**Plan:** `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Branch/worktree:** `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `2e0125c`. Reviewed in the clean worktree `.worktrees/retro-amend` (== `origin/<branch>`).
**Amend reviewed:** `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md` (`done`, amends this plan). **Previous review:** `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (fail).
**Design:** `harness/designs/retro-kit.md`, including Addendum A1–A7 and "Acceptance (amend)". **Kit:** `harness/UI-KIT.md` v2.
**Diff:** `git diff main...harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n --stat`

## Plan vs idea
Unchanged from the first review. Plan 1 of 6 delivers the kit, not the screens. The amend fixes everything the first review found, and each fix reproduces in the browser. A new learner-visible regression remains, though. It was found by widening the contrast walk from the kit components to the v1 components that inherit the re-hued tokens: the v1 `growth`/`alert`/`mute` pairs get worse on every live page (new blocker below). The first review's blocker covered text that the kit itself draws on v1 surfaces. This one covers v1 text drawn in the kit's new hues. Same class, different surface.

## Code vs plan
Run by the reviewer in `.worktrees/retro-amend` @ `2e0125c`:
```
npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build && npx nuxi generate   → exit 0
  Test Files 34 passed (34) · Tests 230 passed (230)      (executor reported 230, reproduces)
ls .output/public/_nuxt | grep -c '\.woff2$' → 9 · unique woff2 in sw.js → 9
gh run list --branch <branch> → CI 36298745809 ok on 2e0125c
git merge-tree --write-tree origin/main origin/<branch> → ef4412d (clean, rc 0)
```
Executor evidence reproduces, so this is not a gate failure. The amend's tasks 1–6 were followed with no unlogged deviations. The diff (`03cb8b6..2e0125c`) matches A1–A5: palette fallback in `PixelArt`; `hexPalette` removed ×4; StateBlock/CountdownTimer/RetroPanel paint `bg-ground-1` + `text-ink-*`; the toast no longer uses `band`; MapNode `p-0.5` wrapper around a 40×40 `[data-tile]`; `useReducedMotion` with `reduced: undefined` ×6; the CompanionSprite 300 ms hold.

### First review's findings, re-checked (Playwright, `nuxi dev`, no backend, 375×812, `aelp.auth` seeded; throwaway `/_kit`, deleted after)
| Finding | Before (review 1) | After (@ `2e0125c`) |
|---|---|---|
| BLOCKER StateBlock/CountdownTimer ink on v1 surfaces | `rgb(244,241,255)` on white, **1.05:1** | Own `ground-1` plate: error/empty text **15.97:1**; "Thử lại" 14.16:1; `/learn/x` empty 15.97:1, its button 11.69:1; CountdownTimer caption 8.88:1, digits 15.97:1. The same in **light and dark** on `/`, `/learn/x` and `/_kit`. **Resolved** |
| Loader cells | `bg-ground-2`, near-invisible | cells `rgb(201,196,244)` (`line-lit`) on a `rgb(21,20,52)` plate, both schemes. **Resolved** |
| Kit text inheriting v1 colour; toast `band` | panel/MapNode text inherited | Panel `text-ink-0` on its own ground; MapNode digits `ink-0`/`ink-2`, `z-10`, the partial band `h-2` clear of them; toast unbanded. **Resolved** |
| Black quest-icon pixels | 48 `rgb(0,0,0)` rects | **0** (all rects on `/_kit`, every glyph/state/stage incl. unknown stage). **Resolved** |
| OS reduced motion on the bars | `transitionDuration` 0.3s | **0s** on every `[data-fill]`/`[data-segment]` (0.3s without the emulation). No running animations. SpeechBox shows the whole line at once; the chest reveals within 50 ms. **Resolved** |
| Sprite `reacted` under reduced motion | never fired, `data-react="hit"` stuck | `[data-log]` = `hit` after **312 ms** (the non-reduced path emits in 265 ms). **Resolved** |
| Torch focus ring (real `Tab` navigation) | missing on tiles, v1 growth ring | QuestNode ×3 and MapNode ×5: `outline: solid 2px rgb(242,168,59)`, offset 2px, v1 ring spread `0px`. RetroButton the same. **Resolved** |
| MapNode hit area | 40×40 | button **44×44**, tile 40×40, all five states. **Resolved** |
| Fonts / Vietnamese | pass | `document.fonts.check` true for `ặỆữ` in VT323 and Nunito; 9 woff2 built and precached. Pass (but see the low bug: no `unicode-range`) |

### Design "Acceptance (amend)" 1–10
1 pass (light/dark contrast above). 2 pass (loader). 3 pass. 4 pass. 5 pass (0 black rects; locked icon in `ink-2`, visible in `rr-kit-light-375.png`). 6 pass (unit). 7 pass. 8 pass (browser 312 ms; unit fake timers incl. unmount mid-hold). 9 pass (focus/44×44; the pressed inset is in the class list and unit-pinned, not browser-measured). 10 pass (no `pages/`/`tailwind.config.ts` change since `03cb8b6`, no `_kit` tracked, CI green).

### Original design acceptance, re-walked for regressions
1 tokens/contrast **pass** (unit) — but see the blocker: the kit pairs pass, and the v1 pairs the re-hue creates were never listed. 2 radius guard pass. 3 HpBar cells/tones and DayBar pass (visible in the kit screenshot). 4 **now pass** (48 px buttons, 56 px QuestNode, 44 px MapNode). 5 fonts pass. 6 six stages plus unknown pass. 7 **now pass**. 8 no v1 component deleted; every page builds. 9 no `_kit`; screenshots committed. 10 pass.

### Integration with the other pending frontend branches
`main` has **no frontend change** since this branch's base (`67ad0c0`). The 23 commits are harness records and tooling. Stay-signed-in, the raw-JSON fix, growth moment, name-your-plant, streak shield, roadmap tree and caddyfile are all `done`/`merged: false`, and their code is only on their own branches. So `merge-tree` against `main` is clean, and the real question is today's integration branch. Pairwise, retro merged into each of those branches (scratch worktree, deleted after):
- The **only textual conflict is `harness/CODEMAP.md`**, in all of stay-signed-in, raw-JSON, finish-level, growth moment, name-your-plant, streak shield and roadmap tree. No frontend source file is touched by both sides.
- After a CODEMAP-resolved merge, `eslint`, `nuxi typecheck` and `vitest` are green: stay-signed-in 240, growth moment 266, name-your-plant 233, streak shield 239, roadmap tree 254 tests. (One lint run on stay-signed-in reported red once but was green on re-run with the same tree, so it was a transient node_modules symlink artefact.)
- Semantic clash: those branches add `bg-growth … text-white`, `text-growth`, `text-streak` and `text-mute` UI (growth moment, roadmap tree and name-your-plant most of all). They inherit the v1 contrast drop in the blocker. Separately, name-your-plant, streak shield and roadmap tree conflict with *each other* in `pages/index.vue`, `utils/plant.ts`, `tests/unit/plant.test.ts` and `service-worker/sw.ts`, which is not this branch's concern.

### Screenshots (`harness/reviews/retro-kit-screens/`)
`rr-hub-{light,dark}-375.png`, `rr-learn-{light,dark}-375.png`, `rr-kit-{light,dark}-375.png`, `rr-kit-focus.png`, `rr-login-{light,dark}-375.png` (the white-on-growth sign-in button).

## Quality
- **Design/boundaries:** clean. The `PixelArt` fallback removes the whole palette-gap bug class as the first review suggested. `useReducedMotion` is a single owner with one listener (fine under `ssr: false`). The kit stays Nuxt-free and has no `dark:`.
- **Correctness:** `SpeechBox` does not watch `isReduced` (`SpeechBox.vue:51`), so an OS flip mid-line keeps typing. `CompanionSprite` does watch it. Medium bug.
- **Test honesty (validator pass, read-only; 114/114 on the touched files):** there are no live OS-flip tests for any consumer, and no OS-default test for `CompanionSprite`. `StateBlock.test.ts:34` asserts only `style.length > 0` under an "ember" title, so it would pass if the tone binding were reverted. Medium bug. The `vi.doMock`/`resetModules` isolation is ordered correctly. It relies on Vitest's default `isolate: true`.
- **Performance:** the fontsource per-subset files ship without `unicode-range`, so every face claims `U+0-10FFFF`. The first uncached load may fetch more faces than it needs. Low bug.
- **Tokens:** the design's contrast test covers kit pairs only. The re-hued v1 aliases are unguarded (blocker).
- **Security:** nothing in scope. CODEMAP: the executor's update is accurate, so no correction.

## Bugs filed
- **BLOCKER (high, `blocks:` this plan)**: `harness/ideas/_inbox/retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md`. White on `bg-growth` goes from 2.54 to **1.67:1** (the sign-in button, `AppButton` primary, the avatar, `QuestRow` "Học"). `text-mute` on light surfaces goes from 4.76 to **3.38–3.53:1**, so it now fails AA. White on `bg-alert` goes from 3.76 to 3.08:1. It hits every live v1 page in both schemes.
- medium: `harness/ideas/_inbox/retro-amend-tests-miss-live-reduced-motion-flips-and-the-sta.md`. No live OS-flip tests; SpeechBox ignores a mid-line flip; the StateBlock ember assertion is empty.
- low: `harness/ideas/_inbox/fontsource-per-subset-css-has-no-unicode-range-so-all-nine-v.md`
- The first review's 10 bugs: the blocker and the five mediums are fixed by the amend and verified above. The four lows stay `selected`.

## Verdict
**fail.** The amend delivers: every finding from the first review is resolved in the browser, the amend's acceptance 1–10 passes, and nothing in the original acceptance regressed. Build, 230 tests and CI are green, and `merge-tree` with `main` is clean. The branch still must not merge today. The kit's token re-hue lowers contrast on v1 text across every live page: the sign-in button drops to 1.67:1, and `text-mute` falls below AA in the light scheme. The pending growth-moment/roadmap-tree/name-your-plant UI would inherit it. The fix is a small second amend on this branch (`amends:` this plan), token- or class-level, with `tokens.test.ts` pinning the v1 pairs. When it lands, the only integration friction with today's other frontend branches is a `harness/CODEMAP.md` conflict.
