---
type: feature
status: planned
source: human
run: 2026-09-25-run-01
priority: high
plan: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
---
# Retro adventure UI: mobile-first 16-bit JRPG restyle with a new UI kit, night-dungeon palette and the plant as party companion

## Why
Owner decision (2026-09-25) after the first real session: the current "gamified minimalist" UI does not hold attention for the 30-minute daily target (see idea "A session a learner wants to finish…" and the inbox bug on raw-JSON tasks). The owner wants the product to *feel like a retro adventure game* so learning English reads as play, not homework — and asked for the UI to be remade in that style with a proper UI kit, mobile-first, using the designer role. Decisions taken with the owner:
- **Flavor: 16-bit JRPG** (SNES era): pixel-grid type and borders, dialogue boxes with a portrait, a world-map roadmap, chests/XP for rewards. Rich enough for passages and quizzes to stay legible.
- **The plant stays the retention mechanic, restyled as the party companion**: same health / streak / stage maths (CODEMAP `pet`), drawn as a pixel creature that levels up with the learner and speaks in dialogue boxes ("tớ").
- **Palette: night dungeon** — deep indigo grounds, torchlight amber, teal accents; dark-mode native. Semantic mapping kept: growth (progress) → teal/green glow, streak/reward → torchlight amber, danger → ember red.
- **Mobile-first**, thumb-first; desktop is the same column, centred, on a dungeon backdrop.

## Expected output
Design first (designer role, harness-design skill), then plans per screen:
1. **`harness/UI-KIT.md` v2 — the retro kit**: token table (grounds, ink, three semantic glows, torchlight, panel borders), type (a pixel/bitmap display face for headings and numbers + a highly legible body face for passages — reading must not be pixel), spacing on an 8-px grid, the panel/dialogue-box/button/progress-bar/chest/badge component set with states, motion budget (sprite-sheet steps, 150–300 ms, reduced-motion fallback), sound-free by default, copy register (Vietnamese, the companion speaks as "tớ", quest vocabulary: "nhiệm vụ", "kho báu", "hồi sinh"), accessibility floor (contrast ≥ 4.5:1 for body text on indigo, ≥ 44 px targets, no information by color alone). A companion visual board lives in the Claude Design artifact "Retro Quest UI Kit".
2. **Design docs, one per screen, inheriting the kit** (`harness/designs/retro-*.md`): hub (companion, day progress as an HP-style bar, three quests as a path on a map tile), learning room (one item at a time in a dialogue box; answer → immediate hit/miss feedback + XP), onboarding (goal card → placement as the "first battle" → roadmap ready), roadmap (world map with 28 nodes across 4 regions), revive (companion down; 15-minute recovery quest), settings (inn/menu), login (title screen). Each carries an acceptance list for the reviewer.
3. **Plans**: kit tokens/components first (one plan: tailwind tokens, fonts, base components, sprite for the companion's stages), then screens in the order hub → learning room → onboarding → roadmap → revive/settings/login. Every plan depends on the typed-content contract bug being fixed so the learning room has real items to render.
4. Out of scope: sound, new gameplay systems (inventory, battles), backend changes beyond naming in copy.

## Evidence
- Owner conversation 2026-09-25 (choices recorded above); frontend spec §6.1 "Modern Gamified Minimalist" is superseded for visual style by this decision — the spec's §7 flows and states still apply.
- Current kit: `harness/UI-KIT.md` (tokens `growth/streak/alert/ink/paper/mute`, Fraunces display, 16/12 px radii), `frontend/tailwind.config.ts`, `frontend/components/*`.
- Retention thesis: 1st-thinking §1 (30 min/day), §5.2 daily loop, backend spec §8 plant maths (unchanged).

## Evaluation
_Evaluator, 2026-09-26 — daily decide (two-cap rule: ≤5 bug plans + ≤5 feature plans; this is a feature verdict)._

**Select — high. Planned today: plan 1 of 6 (the kit).**

*Is the Why real?* Yes — owner decision after the first live session, and the design work is already done: `harness/UI-KIT.md` v2 and the six screen docs under `harness/designs/retro-*.md` were merged in PR #32. `harness/designs/retro-README.md` fixes the build order (kit → hub → learning room → onboarding → roadmap → revive/settings) and the dependencies.

*Achievable in one plan?* Not as a whole (six screens); the kit alone is one plan of ~4 h: tokens in `frontend/tailwind.config.ts`, two self-hosted faces (VT323 + Nunito, Vietnamese subsets), the `components/retro/` set with the states the kit table lists, the companion sprite, restyled `StateBlock`/`CountdownTimer`, and the Vitest checks the README names. No page changes in plan 1, so nothing a learner sees changes until plan 2 — that is by design: the kit is what every later screen plan builds on and it can run in parallel with the typed-content bug's backend branch. *Dependencies:* none for the kit. Plan 3 (learning room) also needs the typed-content branch on `main`; it is written tomorrow.

*Priority.* High — every retro screen and the 59-of-84 rendering fix's frontend half wait on it, and the owner asked for it explicitly. Auto-approved as a `priority: high` feature.
