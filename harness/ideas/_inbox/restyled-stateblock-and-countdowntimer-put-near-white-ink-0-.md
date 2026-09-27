---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
blocks: harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md
---
# Restyled StateBlock and CountdownTimer put near-white ink-0 text on v1 white surfaces, so every error, empty and timer line is invisible on light-scheme phones

## Why
**Blocker for the retro kit plan.** Plan 1 restyles two v1 components in place but leaves the v1 surfaces under them, which design §0 Q1 said to keep until plan 6. In the light colour scheme (`html` is `bg-paper` #F8FAFC, `AppCard` is `bg-white`) the new text classes are close to white:
- `StateBlock` empty/error message is `text-ink-0` (#F4F1FF) on a white card. Contrast is about 1.05:1, so it cannot be read. This is live on `/`, `/roadmap`, `/learn/:id`, `/revive` and `/onboarding`.
- `CountdownTimer` shows `text-ink-0` digits and a `text-ink-1` (#B7B3DC) caption on `bg-paper` in the `/learn/:id` header, so the timer is invisible during every task.
- `StateBlock` loading cells are `bg-ground-2` (#1F1D4A), not the design's `line-lit`. On the dark-scheme `bg-ink` (#1E293B) card they almost disappear.

Every learner who uses a light-scheme phone hits this on the happy path: each failed or empty load, and every task's timer. If the branch merges as it stands, production regresses.

## Expected output
- The restyled `StateBlock` and `CountdownTimer` are readable (≥ 4.5:1) on the surfaces they actually sit on today, in both schemes. Two ways to get there: keep v1-scheme-aware text colours until plan 6 flips the ground, or give each one its own `ground-1` backing. The evaluator picks one.
- The loading cells light `line-lit` in turn, per design §5.
- A test or browser check renders `StateBlock` error/empty and `CountdownTimer` under `prefers-color-scheme: light` on the v1 card and asserts the contrast. Neither component has a dedicated unit test today; they are only mounted inside page tests.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md` (branch `harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` @ `03cb8b6`).
- `frontend/components/ui/StateBlock.vue:28` (`bg-ground-2` loader), `:33` (`text-ink-0`). `frontend/components/learn/CountdownTimer.vue:11-12`. `frontend/assets/css/main.css` (`html { @apply bg-paper … text-ink }`). `frontend/components/ui/AppCard.vue:2` (`bg-white`).
- Browser (`npm run dev`, Playwright, light scheme, no backend): the hub's three error blocks computed `color: rgb(244, 241, 255)` on `background: rgb(255, 255, 255)`. Screenshots: `harness/reviews/retro-kit-screens/hub-light-375.png` and `learn-light-375.png`. The timer finding comes from reading the code, because `/learn/:id` needs a backend to mount the timer.
