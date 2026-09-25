# UI kit — the rules every screen follows

Canonical for the designer, executor and reviewer. Values are what ships in `frontend/` today; the frontend spec §6 is the origin, `harness/designs/frontend-shell.md` the first application. Change this file only through a plan that also changes the code it describes.

## Tokens (`frontend/tailwind.config.ts`)
| Token | Value | Use |
|---|---|---|
| `growth` | `#10B981` emerald | plant growth, success, completed, primary action |
| `streak` | `#F59E0B` amber | streak, rewards, milestones — never for errors |
| `alert` | `#EF4444` coral | health depletion, wilted, destructive, errors |
| `ink` | `#1E293B` slate | text on light; dark-mode surfaces |
| `paper` / `paper-dark` | `#F8FAFC` / `#0F172A` | page ground light / dark |
| `mute` | `#64748B` | secondary text, hints, disabled |
Type: `font-display` = Fraunces Variable (serif; headings, the plant's voice, big numbers) over the system sans for body. Radius: `rounded-card` 16 px for cards, `rounded-btn` 12 px for buttons. Spacing on the Tailwind 4-px scale; a screen is one column, `max-w-md`, 16 px side gutter, cards stacked with `mb-4`.

## Components (`frontend/components/`)
`ui/AppCard` (title slot, the only container with a border), `ui/AppButton` (primary = growth, secondary = outline, destructive = alert; full-width on mobile), `ui/StateBlock` (`loading | error | empty`, message + one action — every async region uses it), `ui/HealthBar`, `ui/SegmentedProgress` (the 30-minute day in three segments), `AppHeader` (name + streak), `plant/PlantSvg` (stage + health), `plant/SpeechBubble` (the plant talks), `quest/QuestRow` (`done | next | locked | open`), `learn/ContentViewer`, `learn/CountdownTimer`, `roadmap/RoadmapNode`, `onboarding/GoalCard`. Reuse these; a new component is a kit addition proposed in the design doc.

## Rules
- **Mobile-first, thumb-first.** Primary action at the bottom, ≥ 44 px targets, one primary action per screen.
- **Every async region has three states** through `StateBlock`; never a blank card, never raw JSON. Offline shows cached content plus a quiet banner, never an error wall.
- **Semantic color is not decoration.** Growth = progress made, streak = reward, alert = danger. No other saturated colors.
- **Copy** is Vietnamese, sentence case, short; the plant speaks in the first person ("tớ"), the app never exclaims. English only inside learning content.
- **Motion is spent in one place per screen** (the plant, a correct answer, a completed task), 150–300 ms, and is removed under `prefers-reduced-motion`. No page transitions, no loaders that bounce.
- **Feedback within 100 ms** of any tap: pressed state, then the real result. A learner is never left wondering whether a tap landed.
- **Complete by doing.** A task ends when its last item is done, not when a timer runs out; time is measured, never used as a gate.
- **Dark mode** ships with every screen (`dark:` variants on ground, ink and borders).

## Flow (spec §7)
Login → Onboarding (goal card → placement → waiting → roadmap ready) → **Hub** (`/`: plant, day progress, three quests as a path) ↔ **Learning room** (`/learn/:id`: one item at a time, progress "Câu n / N", feedback per answer, done → back to the hub with the growth moment) · Roadmap tree (`/roadmap`) · Revive (`/revive`, health 0) · Settings (`/settings`). A design names where the learner enters and every exit.
