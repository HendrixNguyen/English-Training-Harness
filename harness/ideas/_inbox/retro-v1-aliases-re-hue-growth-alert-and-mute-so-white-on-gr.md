---
type: bug
status: planned
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
plan: harness/plans/2026-09-27-retro-v1-aliases-re-hue-growth-alert-and-mute-so-white-on-gr.md
---
# Retro v1 aliases re-hue growth, alert and mute, so white-on-growth buttons drop to 1.67:1 and text-mute to 3.4:1 on live v1 pages

## Why
The retro kit changes the hex of `growth` (#10B981 → #3DE1B0). It also points the v1 aliases `alert` → `ember` (#EF4444 → #FF5A4E) and `mute` → `ink-2` (#64748B → #8783B5), so v1 components "pick up the v2 hue" (design §1 alias table). Nobody checked the v1 text pairs those components draw with these colours. Merging this branch lowers them on every live v1 page, in both schemes:

| pair (v1 component) | `main` | this branch |
|---|---|---|
| `text-white` on `bg-growth` (login "Đăng nhập bằng Google", `AppButton` primary everywhere, `QuestRow` "Học", `AppHeader` avatar, `RoadmapNode` today) | 2.54:1 | **1.67:1** (light and dark) |
| `text-white` on `bg-alert` (`AppButton` danger, `pages/index.vue` error banner, `pages/revive.vue` alert) | 3.76:1 | 3.08:1 |
| `text-mute` on `paper`/white (19 uses: hub "Lộ trình", section headings, "‹ Quay lại" on `/learn`, captions) | 4.76:1 (passes AA) | **3.38–3.53:1** (fails AA) in light scheme; 4.14–5.05:1 in dark |

The sign-in button is the first control a new learner has to read. The kit itself already knows `growth` needs dark text: `RetroButton` primary is `bg-growth text-ground-0` (design §5, 11.69:1). This is the same class of problem as the blocker the amend just fixed: a v1 surface that becomes harder to read the moment this branch merges. Pending frontend branches (growth moment, roadmap tree, name-your-plant, streak shield) add more `text-growth`, `bg-growth … text-white` and `text-mute` uses, and they would inherit the drop.

## Expected output
- No v1 text pair gets worse than it is on `main`. Every pair that passes AA (≥ 4.5:1) on `main` still passes after the merge, in both `prefers-color-scheme` values, at 375 px, with no backend.
- Concretely: text on `bg-growth`/`bg-alert` in v1 components uses `text-ground-0` (the kit's own pair), and `text-mute` on v1 light surfaces is ≥ 4.5:1. Two ways to get there: keep `mute` at its v1 hex until plan 6 swaps the pages, or switch those v1 uses to an ink that passes. The choice is the evaluator's and designer's, and the design §1 alias table should be updated to match.
- `tokens.test.ts` asserts the v1 pairs that v1 components actually draw (white or ground-0 on growth/alert, mute on paper/white/paper-dark), so a later re-hue cannot silently regress them.
- No page layout or copy change.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (amend `harness/plans/2026-09-27-restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`), branch @ `2e0125c`.
- `frontend/tailwind.config.ts` (the `growth` hex, and `streak`/`alert`/`mute` aliases to `torch`/`ember`/`ink-2`); design `harness/designs/retro-kit.md:29`.
- Uses: `frontend/components/ui/AppButton.vue:16-17`, `components/AppHeader.vue:21`, `components/quest/QuestRow.vue:18`, `components/roadmap/RoadmapNode.vue:11`, `pages/index.vue:32`, `pages/revive.vue:74`. `grep -rnoE 'text-mute' pages components` gives 19.
- Browser (Playwright, `nuxi dev`, no backend, 375×812): `/login` button computed `rgb(255,255,255)` on `rgb(61,225,176)` = 1.67:1 in both schemes. The hub avatar is the same. Hub `text-mute` is `rgb(135,131,181)` on `rgb(248,250,252)` = 3.38:1 and on white = 3.53:1 (light). Screenshots: `harness/reviews/retro-kit-screens/rr-login-light-375.png`, `rr-hub-light-375.png`.
- Review: `harness/reviews/2026-09-27-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n-2.md` (re-review).

## Evaluation
**Verdict: select, priority high (blocker).** It fails the review on plan 1, and a learner hits it on the first control they see (the `/login` button at 1.67:1).

**Root cause (evaluator, read at `2e0125c`).** `frontend/tailwind.config.ts:37-42` defines the v1 aliases as `streak: v2.torch`, `alert: v2.ember` and `mute: v2['ink-2']`, and re-hues `growth` to #3DE1B0. Design §1 accepted this without listing the v1 text pairs drawn in those hues. `tests/unit/tokens.test.ts:41-45` asserts only that the aliases equal their v2 targets and checks kit pairs on `ground-*`, so nothing measures the v1 pairs on `paper`, white, `paper-dark` or `ink`.

**Fix.** Designer rule, `harness/designs/retro-kit.md` Addendum 2, A8–A12:
- Pin `streak`, `alert` and `mute` back to their `main` hex. They are v1-only names the kit never draws.
- Keep `growth` at v2.
- v1 text on a growth or alert fill becomes `text-ground-0`.
- Small v1 green text becomes `text-ink dark:text-growth`; large text and glyphs become `text-growth-deep dark:text-growth`.
- The v1 focus ring gets the same split.

This needs 4 page lines. A8 overrides the plan-1 "no pages" rule for those lines only.

**Floor.** No v1 pair is worse than on `main`, and every pair this amend touches reaches AA. Full AA on every v1 pair in both schemes cannot be done with one hex per token (proof in A8). The pairs that were already below AA stay at their `main` value until plans 2–6 replace those surfaces.

**Dependencies.** None beyond the branch.

**Plan.** A second amend, `amends:` plan 1, on the same branch. The medium bug `retro-amend-tests-miss-live-reduced-motion-flips-and-the-sta.md` is folded in.
