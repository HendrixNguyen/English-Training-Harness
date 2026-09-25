# Design: the plant has a name

**Idea:** `harness/ideas/2026-09-25-run-01/name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o.md`
**Inherits:** `harness/designs/frontend-shell.md` (palette, type, card/field shapes, Vietnamese UI register, §2.2 onboarding layout, §2.3 hub, §2.6 revive, §3 speech bubble). This document only adds the one field and the places the name is spoken.
**Wire contract:** backend spec §6.1 — `POST /api/v1/onboarding/assessment` gains optional `plant_name` (trimmed, 1–30 characters; blank or absent → the server default "Mầm Non"); the 201/200 body's `pet_state.plant_name` and §6.3 `GET /pet/status` `plant_name` already carry it.

**Subject and job.** The learner, ninety seconds into the product, about to take a placement quiz they did not ask for. The field's job is to make the plant *theirs* before it has done anything for them, in one tap or none: a name they type, or the suggested one if they skip. It must never block the quiz.

**Register.** Vietnamese, sentence case. The plant keeps speaking in first person ("tớ"); its name appears where a third person speaks about it — the hub caption, the wilted alarm, the result line — and in the bubble only where "tớ" is replaced by the name.

---

## 1. The field (goal step, §2.2)

Placed between the goal cards and the reminder time — the goal is *why*, the name is *who*, the time is *when*.

```
┌──────────────────────────────────────┐
│ Mục tiêu học của bạn là gì?  (title) │
│ [🎓 IELTS 7.0] [💼 Business English] │
│                                      │
│ Đặt tên cho cây của bạn (không bắt   │  ← label, text-sm text-mute
│ buộc)                                │
│ ┌──────────────────────────────────┐ │
│ │ Mầm Non                          │ │  ← placeholder, never a value
│ └──────────────────────────────────┘ │
│ Tên cây dài 1–30 ký tự, chỉ gồm chữ, │  ← role="note", text-alert, only when invalid
│ số và dấu cách.                      │
│                                      │
│ Chọn giờ nhắc học hằng ngày          │
│ [ 20:00 ]                            │
│ [ Bắt đầu bài kiểm tra đầu vào ]     │
└──────────────────────────────────────┘
```

- Native `<input type="text">`, same field shape as the time input (`rounded-btn border border-ink/15 bg-transparent px-3 py-2 dark:border-paper/15`, 48 px tall), `maxlength="30"`, `autocomplete="off"`, `enterkeyhint="done"`.
- Placeholder "Mầm Non" is the suggestion: leaving the field empty sends no `plant_name`, and the server names the plant "Mầm Non". The placeholder is not pre-filled as a value, so a learner who skips it is not tricked into "editing" a name.
- Validation, inline, before submit: trim; empty is valid; otherwise 1–30 characters matching `^[\p{L}\p{N} ]+$` (any script's letters and digits, spaces). Invalid → `aria-invalid`, the note above in `alert`, and the start button disabled — the same pattern the reminder-time field already uses. Whitespace-only is treated as empty.
- The label carries "(không bắt buộc)" so the skip is explicit.

## 2. Where the name is spoken

| Place | Before | After |
| --- | --- | --- |
| Onboarding result step | "My Green Buddy đã nảy mầm. Tưới cây bằng 30 phút học mỗi ngày." | "**Mầm Non** đã nảy mầm. Tưới cây bằng 30 phút học mỗi ngày." — from `pet_state.plant_name` in the response, never from the field |
| Hub plant card (§2.3) | plant → health bar → bubble | **name caption** above the plant: `font-display text-lg text-center`, one line, from `GET /pet/status`; then plant → health bar → bubble unchanged |
| Hub wilted banner (§2.3) | "⚠️ Cây xanh đang bị héo rũ!" | "⚠️ **Mầm Non** đang bị héo rũ!" |
| `/revive` alert band (§2.6) | "⚠️ Cây xanh đang bị héo rũ!" | "⚠️ **Mầm Non** đang bị héo rũ!" (uppercase tracking stays; the name is rendered as typed — CSS `uppercase` is not applied to it, so a name like "Bé Lá" is not shouted) |
| Speech bubble (§3) — target met | "Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿" | "Cảm ơn bạn, hôm nay **Mầm Non** đủ nước rồi 🌿" |
| Speech bubble (§3) — health 1–29 | "Tớ sắp héo mất! Học một chút nhé?" | "**Mầm Non** sắp héo mất! Học một chút nhé?" |

The two remaining bubble lines ("Tưới cho tớ 10 phút học đi!", "Tớ hơi khát rồi… 10 phút thôi?") keep "tớ": the name in every line would read as the plant referring to itself in the third person all day. `speechLine` takes the name as an optional input whose default is "Tớ"/"tớ" so every existing line is byte-identical when no name is passed.

## 3. States

| State | Field | Start button |
| --- | --- | --- |
| empty (default) | placeholder "Mầm Non" | enabled (given a goal and a valid time) |
| valid | value | enabled |
| too long (> 30 after trim; reachable by paste despite `maxlength`) | `aria-invalid`, note | disabled |
| disallowed characters (punctuation, emoji) | `aria-invalid`, note | disabled |
| whitespace-only | treated as empty; no note | enabled |
| server `400 invalid_request` | the existing alert copy gains "tên cây": "Máy chủ không nhận thông tin đã gửi. Kiểm tra lại mục tiêu, tên cây và giờ nhắc học rồi thử lại." | quiz answers kept, as today |

Hub/revive when `plant_name` is missing or empty in a response (should not happen — the DDL default and the server default both fill it): the caption line is not rendered and the banners fall back to "Cây xanh". No skeleton change: the caption appears with the rest of the pet card.

## 4. Components

| Component | Status | Role |
| --- | --- | --- |
| native `<input type="text">` | — | the name field, styled like the existing time field; no new component |
| `AppCard`, `AppButton`, `PlantSvg`, `SpeechBubble`, `HealthBar` | existing | unchanged |
| `GoalCard` | existing | unchanged |

Nothing new is registered. No icon, no avatar picker, no rename UI (a rename belongs on `/settings`, approved 2026-09-24).

## 5. Tokens (all existing)

- Colour: `mute` for the label, `alert` for the note and the banners (unchanged), `ink`/`paper` field border per shell.
- Type: label `text-sm text-mute`; field body 16/24; note `text-sm text-alert`; hub caption `font-display text-lg` (Fraunces at a third, small size is a deliberate exception to the shell's two-size rule: the caption is the plant's name, i.e. bubble-register text, and it sits directly above the plant it names).
- Shape: field 12 px radius, 48 px tall on touch, focus ring `ring-2 ring-growth ring-offset-2`.
- Motion: none.

## 6. Accessibility

- The field has a visible `<label>`; the note is `role="note"` and referenced by `aria-describedby` when shown; `aria-invalid` mirrors validity.
- `PlantSvg`'s `aria-label` is unchanged (stage and health); the name is adjacent visible text, so screen readers hear it in reading order.
- The name is rendered as text, never interpolated into HTML.

## 7. Self-critique

Removed: a random-name "🎲" button (a second control on a step that must not slow the quiz), a live preview of the plant "saying" the name (the bubble already does that on the hub two screens later), and a character counter (the note plus `maxlength` cover the one failure mode). The one risk taken is the third Fraunces size for the caption; it is defensible because the name is the plant's own voice register, not UI chrome.
