---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Settings page misses the design hero time and breaks UI kit v2 rules (rounded blurred switch, 3 to 1 status text, invisible time-picker icon)

## Why
The branch was built from the v1 design two days before kit v2 (`harness/UI-KIT.md`) landed on `main`; the retro restyle of this screen is `retro-settings.md` (build-order plan 6), so the v1 look itself is expected for now. What is not covered by that transition:
- **Design miss:** `harness/designs/settings.md` §1/§5 make the reminder time the hero (`font-display text-[40px] leading-[44px] tabular-nums`). The page renders the time only inside `SpeechBubble` body text and in a `text-2xl` input — no hero number.
- **New component against kit v2:** `components/ui/AppSwitch.vue` is a new v1-style component added after v2 (kit rule "Reuse before inventing"; `retro-settings.md` draws no switch at all): `rounded-full` (kit: radius ≤ 2 px, "nothing rounder, ever"), knob `shadow-sm` (kit: no blurred shadow), eased 200 ms `transition` (kit: stepped motion only).
- **Contrast:** status lines are `text-sm text-mute`; `mute` is pinned to v1 `#64748B`, which on the dark card (`~#1E293B`) is about 3:1 — below the kit's 4.5:1 floor for body text.
- **Native time input:** its picker icon renders near-black on the dark card (no `color-scheme: dark` / `[color-scheme:dark]` on the input), visible in the screenshot.

## Expected output
Either the retro-settings plan (6) absorbs these explicitly (drop `AppSwitch`, use `ink-1` for status text, `[color-scheme:dark]` on the time field), or a small fix now: hero time per settings.md §5, status text in a ≥ 4.5:1 token, `[color-scheme:dark]` on the input.

## Evidence
- Plan `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md`; `harness/designs/settings.md` §1, §5; `harness/UI-KIT.md` (Grid/shape/depth, Motion budget, Accessibility floor, Rules "Reuse before inventing"); `harness/designs/retro-README.md` row 6.
- `frontend/components/ui/AppSwitch.vue` classes; `frontend/pages/settings.vue` template (`text-mute` status `<p>`s, time `<input>`); `frontend/tailwind.config.ts` `v1Only.mute = '#64748B'`.
- Browser (branch merged into origin/main, `nuxi dev`, /settings): screenshot shows no hero time and a barely visible clock icon in the time field.
