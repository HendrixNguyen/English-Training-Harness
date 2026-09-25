# Design: streak shield — two slots under the health bar

**Idea:** `harness/ideas/2026-09-22-run-01/pet-streak-shield-earned-by-target-days.md`
**Inherits:** `harness/designs/frontend-shell.md` (palette, type, card shapes, Vietnamese register, the §2.3 plant hub, the §3 plant). This document adds one row to the hub card and two lines of plant speech; nothing else on the screen moves.
**Wire contract:** backend spec §6.3 `GET /api/v1/pet/status` gains two additive fields — `shields` (int, 0–2) and `last_shield_used_on` (`YYYY-MM-DD` or `null`). The client learns about a shield only from this read; the `POST /quests/progress` response is untouched.

**Subject and job.** The learner who has kept the 30-minute target for seven days straight, and — the moment that matters — the same learner the morning after a day they missed. The row's single job is to make a held shield visible before it is needed, so that a missed day reads as "the shield took it" instead of "the plant got hurt". It is read in a glance, every day, under the health bar; it must never compete with the plant.

**Register.** Vietnamese, sentence case, the plant's voice in the bubble and the app's voice in the caption. "Khiên" (shield) is the one noun, used the same way everywhere: earned, held, spent ("đỡ" — it took the hit). No "freeze", no "streak freeze" anglicism, no marketing.

---

## 1. The signature: two slots that are always drawn

The row draws **two shield outlines whether or not the learner holds any**. An empty outline in `mute/30` is a promise ("there is something here to earn"); a held shield fills solid `streak` amber, the same colour as the streak pill in the header, because a shield is the streak's own reward; a slot whose shield was spent in the last seven days is drawn in `mute` with a small tick so the learner sees *where* the hit landed. The structure encodes the true fact of the mechanic — a rack of exactly two — without a sentence of explanation, and the state of the rack is legible in colour and shape, never colour alone.

Why it earns its place over an icon-with-a-count ("🛡️ ×1"): the count says nothing to a learner on day one, and the rack says "two, you have none yet". It is also the only new element on the hub; the plant, the bar and the bubble stay exactly where wireframe 7.2 put them.

---

## 2. Layout (mobile, inside the existing hub `AppCard`)

```
┌──────────────────────────────────────┐
│           (Cây ảo SVG 160px)         │
│  Máu cây: [████████░░]          80%  │  ← HealthBar, unchanged
│  Khiên:   [🛡][🛡]                    │  ← ShieldRow, new (mt-2), same left column as "Máu cây:"
│  Khiên đã đỡ cho ngày 24/09.         │  ← caption, only within 7 days of last_shield_used_on
│  💬 "Tròn 7 ngày liên tiếp! Bạn có   │  ← SpeechBubble, existing; new line on the earn day
│      khiên bảo vệ streak rồi 🛡️"     │
└──────────────────────────────────────┘
```

The row mirrors `HealthBar`'s grid: the label "Khiên:" in `text-sm text-mute` sits in the same column as "Máu cây:", the two slots follow at 20 × 20 px with an 8 px gap, and nothing sits at the right edge (the bar's percentage is the only number in that column; a second number would compete with it). The caption, when present, is one line directly under the row, left-aligned with the label. The speech bubble keeps its place under everything, as today.

---

## 3. Components

| Component | Status | Role |
| --- | --- | --- |
| `HealthBar`, `SpeechBubble`, `AppCard`, `PlantSvg` | existing | unchanged |
| `ShieldRow` (`components/plant/ShieldRow.vue`) | **new** | props `shields: number` (0–2, clamped), `lastUsedOn: string \| null`, `today?: string` (`YYYY-MM-DD`; defaults to the device's local date). Renders the label, two slots and the optional caption. Each slot carries `data-shield="held" \| "spent" \| "empty"`; the row is `role="img"` with `aria-label="Khiên: 1 trên 2"` (plus ", một chiếc vừa đỡ cho ngày 24/09" when the caption shows) so a screen reader gets one sentence, not three glyphs |
| `speechLine` (`utils/plant.ts`) | extended | gains `streak` and `shields` inputs; the earn line outranks the target-met line |
| `shieldSpentLine` (`utils/plant.ts`) | **new** | `(lastUsedOn, today) → string \| null`: the caption text when the spend is 0–7 calendar days old, else `null`. Pure calendar arithmetic on `YYYY-MM-DD` (no `Date.parse` of a bare date — it would land on UTC midnight and be a day off east of Greenwich) |

The shield glyph is an inline SVG path inside `ShieldRow` (a rounded shield: `M10 2 L17 5 V10 C17 14.5 13.5 17.5 10 18.5 C6.5 17.5 3 14.5 3 10 V5 Z`), `currentColor` fill/stroke — no icon library, no emoji in the row (the emoji is reserved for the bubble, where the shell already uses them).

---

## 4. States

### 4.1 The rack

| `shields` | `last_shield_used_on` | Slots (left → right) | Caption |
| --- | --- | --- | --- |
| 0 | null / older than 7 days | empty, empty | — |
| 0 | within 7 days | **spent**, empty | "Khiên đã đỡ cho ngày 24/09." |
| 1 | null / older than 7 days | held, empty | — |
| 1 | within 7 days | held, **spent** | "Khiên đã đỡ cho ngày 24/09." |
| 2 | any | held, held | — (a full rack has no spent slot to show; the caption is still shown if within 7 days — the learner should still learn that the hit was taken) |

"Within 7 days" is inclusive of today (0–7 calendar days between `last_shield_used_on` and today). The spent marker occupies the first empty slot: it is the slot that most recently emptied. The date in the caption is `dd/mm` — the learner's own calendar, the same order the header's streak copy and the roadmap use; no year (a spend is at most a week old when shown).

| Slot | Fill | Stroke | Extra |
| --- | --- | --- | --- |
| empty | none | `mute/30`, 1.5 px | — |
| held | `streak` | `streak` | — |
| spent | `mute/25` | `mute` | a 2-px `mute` tick centred in the shield (✓ — it did its job), never a cross (a cross reads as failure, and the shield succeeded) |

### 4.2 The bubble

`speechLine` order, first match wins:

1. wilted (`stage === 'wilted' || health <= 0`) → `…` (unchanged; a shielded miss never wilts, so this and the shield never coincide on the same day).
2. **earned** (`streak > 0 && streak % 7 === 0 && shields > 0`) → "Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️". The copy states the milestone and that a shield is *held* — both true on day 14 (second shield) and on day 21 with a full rack (no new shield, cap of 2), since the status body cannot tell those apart. It is never "tớ tặng bạn một chiếc khiên", which would be false at the cap.
3. target met → unchanged line.
4. health thresholds → unchanged lines.

The earn line shows for the rest of that local day: the hub re-reads `GET /pet/status` on mount after `/learn/:id` navigates back, so the learner sees it right after the session that earned it, and again on any reopen that day. Tomorrow the streak is 8 (or 15, 22) and the ordinary lines return.

### 4.3 What does *not* change

- The header's "🔥 Streak: 7 ngày" pill: untouched (it already shows the streak surviving the miss, which is the shield's effect).
- The wilted banner and `/revive`: untouched; a shielded miss never reaches health 0.
- Loading / error / no-roadmap: the row renders only inside the `pet.status` branch, like `HealthBar`, so the hub's existing state convention covers it.
- Offline: the row shows last-known values from the NetworkFirst cache, like the bar.

---

## 5. Tokens (all existing; nothing new is registered)

- Colour: `streak` (held shield — amber is "streak counts, rewards, and milestone badges" in Frontend spec §6.1, and a shield is exactly a milestone badge), `mute` at `/30`, `/25` and solid (empty outline, spent fill, spent stroke and tick, label, caption).
- Type: label and caption `text-sm text-mute` (the `HealthBar` label class); bubble line in the existing `SpeechBubble` Fraunces.
- Shape: 20 × 20 px slots on the 8-pt grid, 8 px gap; the glyph's own rounded corners — no container, no pill.
- Motion: a slot that changes from empty to held fades its fill in over 200 ms (`transition-colors duration-200 motion-reduce:transition-none`, the `HealthBar` width transition's pattern). No pulse, no bounce, no confetti — the bubble carries the celebration in words. `prefers-reduced-motion` → instant.
- Contrast: `streak` `#F59E0B` on `paper` and on `ink` both clear 3:1 for a 20-px glyph; the spent slot never relies on colour — it has the tick, and the empty slot has no fill.

## 6. Self-critique

Removed before shipping: a shield count in the header pill ("🔥 7 · 🛡 1" — two rewards in one pill make neither legible), a toast on earn (the bubble is already the hub's one announcing voice, `aria-live="polite"`), a "days until next shield" progress ring (a third meter under the plant; and the streak pill already counts), and animating the spent slot on the morning after (the learner should find the hit already taken, quietly — the caption is the whole message). The one risk taken is drawing empty slots for a learner who has never earned anything: two grey outlines under the bar on day one. It is defensible because the outline is the only thing that teaches the mechanic without a sentence, and it is quiet (`mute/30`, 20 px). A forthcoming growth-moment celebration (`harness/designs/growth-moment.md`, planned the same day) may later add "Shield earned" to its own moment; this design does not depend on it and must not wait for it.
