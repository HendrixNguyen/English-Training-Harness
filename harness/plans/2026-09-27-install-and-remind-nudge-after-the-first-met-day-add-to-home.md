---
idea: harness/ideas/2026-09-26-run-01/install-and-remind-nudge-after-the-first-met-day-add-to-home.md
status: approved
priority: medium
merged: false
order: 5
design: harness/designs/install-and-remind-nudge-after-the-first-met-day-add-to-home.md
---
# Install-and-remind nudge — the companion asks to call you tomorrow after your first met day — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Design:** `harness/designs/install-and-remind-nudge-after-the-first-met-day-add-to-home.md` — every task cites its sections; `## Verification` repeats its Acceptance list verbatim.
**Idea:** `harness/ideas/2026-09-26-run-01/install-and-remind-nudge-after-the-first-met-day-add-to-home.md`
**Goal:** After a learner's first target-met day, the hub shows (at most on two days) one companion card that gets the PWA installed where that is required (iOS) or offered (Chromium) and then turns on the daily Web Push reminder through the existing settings flow — so the reminder system actually reaches learners.

**Scope:** frontend only. No backend change: `POST /api/v1/settings/notifications` body is unchanged (backend spec §6.4). No new dependency. **Estimate:** half a day to one day. **Branch:** `harness/2026-09-27-medium-install-and-remind-nudge-after-the-first-met-day-add-to-home`.

**Depends on (must be on origin/main before execution):** the 2026-09-26 daily code PR — specifically
`harness/2026-09-25-medium-settings-screen-wires-web-push-reminders-and-google-calendar` (`utils/push.ts`, `stores/settings.ts`, `composables/useReminders.ts`, `pages/settings.vue`),
`harness/2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target` (`composables/useGrowthMoment.ts` `active`),
`harness/2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n` (`components/retro/*`, `utils/pixelArt.ts`, `pages/index.vue` retro hub),
`harness/2026-09-26-medium-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o` (`pet.status.plant_name`).
```
for b in 2026-09-25-medium-settings-screen-wires-web-push-reminders-and-google-calendar 2026-09-26-medium-growth-moment-after-every-task-health-gain-streak-and-target 2026-09-26-high-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n 2026-09-26-medium-name-your-plant-at-onboarding-and-see-it-greet-you-by-name-o; do git merge-base --is-ancestor "origin/harness/$b" origin/main && echo "ok $b" || echo "MISSING $b"; done
test -f frontend/composables/useReminders.ts && test -f frontend/composables/useGrowthMoment.ts && test -f frontend/components/retro/RetroPanel.vue && echo files-ok
```
Settings, growth-moment or retro missing → stop and report `blocked`. Plant-name missing alone is tolerable: the tab falls back to "Cây xanh" (design §7) — say so in the `done` summary.

## Global constraints (design §8 "Executor traps" are review blockers)

- `Notification.requestPermission` is reached only through `useReminders().enable()` called synchronously from the step-2 click handler — no `await` before it in the handler.
- iOS detection runs **before** `pushSupport` (design §2 "Platform").
- Never post `DEFAULT_REMINDER_TIME` while `timeKnown` is false unless the learner saw the in-card time field.
- Ineligible → the component renders nothing (`v-if` at the root), never a hidden panel.
- Wait for the growth moment's `active === false`, never a timer. `prompt()` is called at most once per captured event.
- Copy exactly design §7; the denied/error/offline strings live only in `utils/reminderCopy.ts`.
- Each task: `cd frontend && npm run lint && npm run typecheck && npm run test:unit` green; one commit per task.

## File structure

| Path | Change |
| --- | --- |
| `frontend/utils/nudge.ts` (new) + `tests/unit/nudge.test.ts` | showing/dismiss/done state and the twice-max rule (Task 1) |
| `frontend/composables/useInstallPrompt.ts` (new), `frontend/plugins/installPrompt.client.ts` (new) + `tests/unit/useInstallPrompt.test.ts` | capture, `detectPlatform`, `prompt()` (Task 2) |
| `frontend/utils/reminderCopy.ts` (new), `frontend/pages/settings.vue`, `frontend/stores/settings.ts` + tests | shared copy, `timeKnown` (Task 3) |
| `frontend/utils/pixelArt.ts` + `tests/unit/pixelArt.test.ts` | `share`, `addHome` glyphs (Task 4) |
| `frontend/components/hub/InstallNudge.vue` (new) + `tests/unit/InstallNudge.test.ts` | the card (Task 5) |
| `frontend/pages/index.vue` + hub page test | mount under the companion panel (Task 6) |
| `harness/UI-KIT.md`, `harness/CODEMAP.md` | glyph list, `shell` paragraph (Task 7) |

## Tasks

### Task 1: nudge state (design §5 "Showing rule")
- [ ] Tests first: fresh state → eligible; `recordShowing('2026-09-27')` → `shownCount 1`; same day again → still 1 and eligible; next day → 2; a third day → not eligible; `dismissToday(d)` → not eligible on `d`, eligible on `d+1` if count < 2; `markDone()` → never eligible; unparsable JSON is removed and treated as fresh; `storageOrNull()` returning null → never eligible.
- [ ] Implement `utils/nudge.ts` with `localStorage['aelp.nudge'] = {shownCount, lastShownOn, done, dismissedOn}` through the existing `storageOrNull()` guard (reuse, don't copy, the settings store's helper — move it to `utils/storage.ts` if it is not already importable).
- [ ] Commit `feat(frontend): nudge showing state with a two-day cap`.

### Task 2: install prompt capture and platform detection (design §2 "Platform", §5 "Capture", §6 `useInstallPrompt`)
- [ ] Tests: `detectPlatform(win, vapid, event)` matrix — standalone via `matchMedia` → `standalone`; `navigator.standalone === true` → `standalone`; standalone with push not ok → `unsupported`; iPhone Safari 17 UA → `ios`; iPadOS Macintosh UA + `maxTouchPoints 5` → `ios`; iOS 15 UA → `unsupported`; iOS UA without version → `ios`; Chromium with event and push ok → `chromium`; push ok, no event → `other`; `no-key` → `unsupported`. Plugin: dispatching `beforeinstallprompt` calls `preventDefault` and stores the event; `appinstalled` clears it and sets `installed`; `prompt()` resolves `userChoice.outcome` and a second call is a no-op returning `'dismissed'`.
- [ ] Implement module-level state in `useInstallPrompt.ts` and the client plugin that calls `listen()` at boot.
- [ ] Commit `feat(frontend): capture beforeinstallprompt and detect install platform`.

### Task 3: shared reminder copy and `timeKnown` (design §6 `utils/reminderCopy.ts`, `stores/settings.ts`)
- [ ] Tests: settings store `timeKnown` false on empty storage, true after `hydrate()` finds a string time, after `rememberTime()`, after a successful `saveReminder()`; `pages/settings.vue` still renders the same denied/error/offline strings (existing settings page test stays green unchanged).
- [ ] Move the three strings from `pages/settings.vue` into `utils/reminderCopy.ts` (`REMINDER_COPY = {denied, error, offline}`) and import them there; add `timeKnown` to the store.
- [ ] Commit `refactor(frontend): shared reminder copy; settings store knows if a time was chosen`.

### Task 4: kit glyphs (design §6 "Kit additions")
- [ ] Tests: `share` and `addHome` exist in `GLYPHS`, are 16×16, use palette characters only.
- [ ] Draw both in `utils/pixelArt.ts`. Commit `feat(frontend): share and addHome pixel glyphs`.

### Task 5: `InstallNudge.vue` (design §1, §3, §4, §5, §7)
- [ ] Tests with a mocked `useReminders`, settings store and `useInstallPrompt`: every row of design §4; chromium flow (prompt accepted → step 2; dismissed → step 2); ios recipe renders three rows with glyph + text and only "Để sau"; step 2 known time → button "Bật nhắc lúc 20:00" calls `enable('20:00')` synchronously in the click handler; unknown time → field prefilled `20:00`, button "Bật nhắc học" sends the field value; `on` → toast text + `markDone`; `denied` → denied copy + "Đóng" + `markDone`; `error` → retry line; offline → disabled + offline line; "Để sau" → hidden and `dismissToday`; eyebrow "Bước 2/2" only when step 1 rendered; speaker tab falls back to "Cây xanh"; renders nothing when `momentActive` is true, `everMet` is false, `remindersOn` is true or permission is `denied`.
- [ ] Implement the component (props `plantName`, `everMet`, `momentActive`), `recordShowing(today)` on first render of a local date.
- [ ] Commit `feat(frontend): InstallNudge card`.

### Task 6: mount on the hub (design §2 "Eligible", §3)
- [ ] Hub test: the card sits directly after the companion panel; absent while loading, on error, on the empty state, when `health_points == 0`, and while the growth moment is active; present after `active` turns false on a met day; `everMet` from `pet.status.last_practiced_at != null || quest.daily.is_target_met`. A test that `/learn/:id` and `/revive` never render it (they don't mount the hub — assert by component search).
- [ ] Mount in `pages/index.vue`. Commit `feat(frontend): hub shows the install-and-remind nudge`.

### Task 7: docs
- [ ] `harness/UI-KIT.md` glyph list gains `share`, `addHome`; CODEMAP `shell` paragraph (plugin, composable, card). Commit `docs: UI kit glyphs and CODEMAP for the install nudge`.

## Verification
```
cd frontend && npm run lint && npm run typecheck && npm run test:unit && npm run build
grep -rn 'requestPermission' frontend --include=*.ts --include=*.vue | grep -v tests   # only utils/push.ts / useReminders
grep -rn 'Trình duyệt đang chặn thông báo' frontend --include=*.ts --include=*.vue | grep -v tests   # only utils/reminderCopy.ts
```
Manual (reviewer, Playwright at 390×844): Chromium — meet a day with a seeded account, the card appears after the growth moment; iOS UA emulation — the recipe renders; reload twice on two faked dates → no third showing.

Design Acceptance (verbatim):
1. With `last_practiced_at` null and `is_target_met` false, the hub never renders the nudge; the first hub paint on which either becomes true renders it — after the growth moment's `active` has turned false when a moment is playing, never during it.
2. The card renders directly under the companion panel as a `RetroPanel` whose tab shows `plant_name`, or "Cây xanh" when `plant_name` is missing or blank, and it never renders on `/learn/*`, `/revive`, onboarding, while the hub is loading, errored, empty (`no_active_roadmap`) or wilted (`health_points == 0`).
3. Platform detection: standalone (`display-mode: standalone` or `navigator.standalone`) with push `ok` shows step 2 only; an iOS ≥ 16.4 tab (incl. iPadOS Macintosh UA with touch) shows the three-row Share → Thêm vào Màn hình chính recipe; Chromium with a captured `beforeinstallprompt` shows "Cài vào máy"; push `ok` without the event shows step 2 only; `unsupported` or `no-key` never renders the card.
4. "Cài vào máy" calls the captured event's `prompt()` once; both `accepted` and `dismissed` switch the same card to step 2; the event is captured from app boot so one fired before the hub mounted is still used.
5. Step 2's button calls `useReminders().enable(t)`, where `t` is `settings.notificationTime` when `timeKnown`, otherwise the value of the in-card time field (prefilled `20:00`); the request is the unchanged `POST /api/v1/settings/notifications` body with `notification_time: "HH:MM:00"`, `timezone` and `push_subscription`.
6. On `on`: the card disappears, one `RetroToast` reads "Tớ sẽ nhắc cậu lúc {t}.", and the card never renders again on that device. On `denied`: the card shows "Trình duyệt đang chặn thông báo. Mở cài đặt trang web, cho phép Thông báo, rồi thử lại." with "Đóng", and never renders again. On `error`/`invalid`: "Không lưu được. Thử lại." shows and the button retries. Closing the browser prompt without a choice leaves step 2 as it was.
7. "Để sau" hides the card for the rest of the local day; the card renders on at most two distinct local dates in total (`aelp.nudge.shownCount ≤ 2`), same-day re-renders do not count, and corrupt or unavailable `localStorage` means it never renders.
8. The card does not render when `settings.remindersOn` is true or `Notification.permission` is `denied` at load.
9. Offline at step 2 disables the button with "Cần mạng để lưu cài đặt."; the denied, error and offline strings are imported from `utils/reminderCopy.ts` by both the nudge and `pages/settings.vue`.
10. The card has no animation of its own; under `prefers-reduced-motion` it is identical, focus is never moved to it, and every glyph in the recipe has visible text beside it.
