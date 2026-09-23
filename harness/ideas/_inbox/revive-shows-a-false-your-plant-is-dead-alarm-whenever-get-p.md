---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: high
---
# /revive shows a false "your plant is dead" alarm whenever GET /pet/status fails

## Why
The pet is the retention mechanism; telling a user with a healthy plant that it has withered is the single most damaging false message this app can show. `pages/revive.vue` has loading, revived and healthy branches but **no error branch** — `pet.error` is never rendered — so the final `v-else` catches "the request failed" together with "the plant really is wilted" and renders the wilted UI unconditionally: a red `⚠️ Cây xanh đang bị héo rũ!` banner, a wilted plant SVG at 0% health, "Bạn đã bỏ học..." and the revival CTA.

This is the offline path the PWA is sold on (Frontend spec §3): if `/pet/status` is unreachable and the service worker has no cached copy, every visit to `/revive` is a false alarm. `/` (`pages/index.vue:40`) and `/roadmap` (`pages/roadmap.vue:30`) both handle the same failure correctly with a `StateBlock state="error"` and a "Thử lại" button, so `/revive` is the outlier, not the house style.

## Expected output
`/revive` renders an error state with a retry action when `pet.error` is set and `pet.status` is null, exactly as `/` and `/roadmap` do. The wilted branch is entered only on real data (`pet.status !== null && pet.isWilted`). Offline with a cached status, the last-known state is shown; offline with no cached status, the user is told the plant's state is unknown — never that it is dead.

## Evidence
- Plan: `harness/plans/2026-09-23-frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md` (Task 11).
- `frontend/pages/revive.vue:74` — `<template v-else>` is reached when `pet.loading === false && passed === false && pet.notWilted === false && pet.status === null`, i.e. the load-failed case; lines 75-93 then render the wilted banner, `<PlantSvg stage="wilted" :health="0" />` and the revive CTA.
- `frontend/stores/pet.ts:50-53` sets `this.error` on failure and leaves `status` null; `grep -n "pet.error" frontend/pages/revive.vue` → no hits (the page's `error` ref is only the `revive()` action's own message).
- Reproduction (run in the plan's worktree, `frontend/`):
  1. `npm run build`
  2. `PORT=3101 HOST=127.0.0.1 NUXT_PUBLIC_API_BASE=http://127.0.0.1:3198 node .output/server/index.mjs` (nothing listens on 3198)
  3. In a browser at `http://127.0.0.1:3101/login`, seed a session:
     `localStorage.setItem('aelp.auth', JSON.stringify({accessToken:'t', expiresAt: Date.now()+86400000, user:{id:'u1',email:'a@b.c',full_name:'Review User',cefr_current:'B1'}}))`
  4. Navigate to `/revive`.
  Observed (accessibility snapshot): `alert: ⚠️ Cây xanh đang bị héo rũ!`, `img "Cây đang ở giai đoạn wilted, máu 0%"`, `paragraph: Cây héo - 0%`, `button "🚨 Cứu cây ngay (Quiz 15 phút)"`. The same run at `/` correctly shows three "Không tải được…" error blocks with "Thử lại" buttons, and `/roadmap` shows "Không tải được lộ trình."
