# Retro UI — build order and dependencies

The retro restyle (idea `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`) is one kit and six screen designs. The evaluator writes one plan per line below, in this order; each plan's Definition of done includes its design's acceptance list.

| # | Plan | Design | Depends on |
|---|---|---|---|
| 1 | Kit: tokens, fonts, base components, companion sprite | `harness/UI-KIT.md` v2 | — |
| 2 | Hub | `retro-hub.md` | 1 |
| 3 | Learning room | `retro-learning-room.md` | 1, **typed-content bug** (below) |
| 4 | Onboarding + title screen | `retro-onboarding.md` | 1, 3 (`ItemQuestion`); calibration row also on the regenerate endpoint |
| 5 | Roadmap (world map) | `retro-roadmap.md` | 1, `roadmap-tree` contract (`GET /api/v1/roadmap`) |
| 6 | Revive + settings | `retro-revive.md`, `retro-settings.md` | 1, `settings` contracts |

## The kit plan (1) in one paragraph
Register the v2 tokens in `frontend/tailwind.config.ts` (keep `growth`; add `ground-0/1/2`, `line-lit/dim`, `ink-0/1/2`, `growth-deep`, `torch`, `torch-deep`, `ember`, `ember-deep`; alias `streak`→`torch`, `alert`→`ember`, `mute`→`ink-2` until plan 6 lands, then delete them with `paper`/`paper-dark`). Self-host VT323 and Nunito (Google Fonts, both with a `vietnamese` subset — the kit's font rule; precached by the service worker). Build `components/retro/`: `RetroPanel`, `RetroButton`, `HpBar`, `DayBar`, `QuestNode`, `MapNode`, `Chest`, `Badge`, `CompanionSprite` (32×32 grid, six stages + `down`, reactions `idle|hit|miss|levelup`), `SpeechBox`, `RetroToast`; restyle `StateBlock` and `CountdownTimer`. Vitest: contrast table (the kit's hex pairs ≥ 4.5:1 for text), sprite renders every stage and an unknown stage, `HpBar` snaps to 4-px cells, buttons ≥ 48 px. No page changes in plan 1.

## The blocking bug
`harness/ideas/_inbox/59-of-84-roadmap-tasks-render-as-raw-json-and-the-other-25-a.md` — until the roadmap content contract is typed (`words[{term, definition, example}]`, `questions[{id, prompt, options, answer, explanation}]`) and enforced by `ParseRoadmap`, the learning room has nothing to render as items. Plan 3 must not start before that bug's backend plan is `done`; plans 1 and 2 can run in parallel with it.

## Rules the executors carry from the kit
Radius ≤ 2 px; no blur, no gradients, no scale tweens; motion stepped and 150–300 ms; body text in Nunito never below 14 px; VT323 only between 14 and 34 px; both faces must ship a Vietnamese subset; every state has a glyph and a Vietnamese string; the companion speaks as "tớ" to "cậu".

## Out of scope
Sound, a daylight theme, XP persistence (client-side only in the room), inventory or battle systems, backend changes beyond the ones named above.
