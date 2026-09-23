---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# The /revive missed-days line renders "bỏ học2 ngày" with no space before the day count

## Why
Pre-existing on `main`, not introduced by the revive error-state fix (the wilted block moved byte-for-byte). The wilted screen's headline copy is:

```
"Bạn đã bỏ học<template v-if="missedDays !== null"> {{ missedDays }} ngày liên tiếp</template>. Hãy hoàn thành…"
```

Vue's default `whitespace: 'condense'` strips the leading space of the inner `<template>`'s first text node, so the browser renders `"Bạn đã bỏ học2 ngày liên tiếp."`. It is the most emotionally loaded sentence in the app — the one shown to a user whose plant has withered — and it reads as a typo.

## Expected output
`/revive` on real wilted data with a known `last_practiced_at` renders `Bạn đã bỏ học 2 ngày liên tiếp.` with the space, and still renders `Bạn đã bỏ học.` cleanly when `missedDays` is `null`. Fix by moving the space outside the conditional (e.g. `bỏ học<template v-if="missedDays !== null">&nbsp;{{ missedDays }} ngày liên tiếp</template>`) or by computing the whole clause in `<script setup>`. Worth a rendered-text assertion in `tests/unit/revivePage.test.ts` so it cannot come back.

## Evidence
- Found while reviewing `harness/plans/2026-09-23-revive-shows-a-false-your-plant-is-dead-alarm-whenever-get-p.md`.
- `frontend/pages/revive.vue:83` (branch HEAD) — identical to `main`'s line 88 (`git show main:frontend/pages/revive.vue | grep -n "bỏ học"`), so this predates the fix.
- Observed in a real browser against the branch's production build (Nuxt preview on 3103, stub API on 3104 returning `health_points: 0, stage: wilted, last_practiced_at: 2026-09-20T13:00:00Z`). Accessibility snapshot: `paragraph: "Bạn đã bỏ học2 ngày liên tiếp. Hãy hoàn thành Bài kiểm tra Cứu Cây 15 phút để hồi sinh!"`.
