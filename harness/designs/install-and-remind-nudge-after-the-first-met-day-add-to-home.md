# Design: Install-and-remind nudge — "keep me alive tomorrow" card on the hub

**Idea:** `harness/ideas/2026-09-26-run-01/install-and-remind-nudge-after-the-first-met-day-add-to-home.md`
**Inherits:** `frontend-shell.md` → `retro-hub.md` (the screen it sits on), `growth-moment.md` (its `done` is the trigger), `plant-name.md` (the speaker), `settings.md` §4.1 + `retro-settings.md` (reminder state machine and copy — reused, not restated).
**Kit:** `harness/UI-KIT.md` v2 (retro kit, `frontend/components/retro/` on the retro branch).
**Spec wireframe:** frontend spec §7.2 (hub) has no install surface; this card is additive. Departure from the idea: the card sits **under the companion panel**, not under the quests (reason in §0).
**Code it reuses (settings branch, not yet on `main`):** `utils/push.ts` (`pushSupport`), `stores/settings.ts` (`notificationTime`, `remindersOn`, `DEFAULT_REMINDER_TIME`), `composables/useReminders.ts` (`init`/`enable`, states `off|requesting|on|denied|unsupported|no-key|error|invalid`). The card drives `useReminders().enable(time)`; it never calls `Notification.requestPermission` or `pushManager.subscribe` itself.

## 0. Research
- **Learner's job:** keep the companion alive tomorrow without having to remember — one or two taps, right after they proved they can do a day.
- **The moment that earns the next minute:** the growth moment has just shown HP rising and the streak chip; the companion then asks, in its own voice, to be allowed to call them tomorrow at the time *they* chose at onboarding. The ask is a favour to the plant they just watered, not a system permission.
- **What today does wrong:** nothing in `frontend/` handles `beforeinstallprompt`, detects standalone, or asks for permission outside `/settings` (only `nuxt.config.ts` sets `display: standalone`). An iPhone learner in a Safari tab can never receive a push (WebKit: Web Push only for Home Screen apps, iOS/iPadOS ≥ 16.4, permission from a user tap). A lesser design would (1) fire the browser permission prompt on page load — instantly denied, permanently; (2) show a generic "Install app" banner on every visit — banner blindness, then resentment; (3) tell an iPhone user "enable notifications" in a tab where it cannot work.
- **Open questions, answered from evidence:**
  - *Where does the card go?* Under the companion panel. On a met day the path is done and the card is the only next step; under the quests it falls below the fold on a 667-px phone (status 48 + panel ~220 + day bar 50 + path ~216 + bottom bar 80). It is shown at most twice, so pushing the day bar down twice is cheap.
  - *What is "the first met day"?* `pet.status.last_practiced_at != null` (set only when a target is met — `growth-moment.md` §3) **or** `quest.daily.is_target_met`. Using the server's `last_practiced_at` matters on iOS: a Home Screen app has **its own storage**, separate from Safari, so the standalone app cannot see anything the tab remembered — but it can see that the learner has met a day.
  - *Where does the time come from?* `settings.notificationTime`, which onboarding writes via `rememberTime()` and `/settings` via `saveReminder()`. There is no GET for it and `AuthUser` has no `notification_time`. When this device never stored a time (fresh iOS standalone storage, second device), the card must **not** silently post `20:00` over the learner's real choice — it shows a time field prefilled with `DEFAULT_REMINDER_TIME` instead. This needs one flag on the settings store (`timeKnown`, §6).
  - *Does Android need step 1?* No — Chrome delivers push to a tab. Install is offered because a Home Screen icon brings people back, but declining the install still leads to step 2.
  - *Which denied copy?* `settings.md` §4.1's. `retro-settings.md`'s denied line ("giờ vẫn được lưu…") is false here: `useReminders.enable` returns on `denied` before saving anything.

## 1. The signature
**The companion asks for your phone number.** After the growth moment settles, a dialogue box with the plant's name on its tab appears under the companion and says, in "tớ", that it wants to call you tomorrow at 20:00 — and for an iPhone it walks you through Share → Add to Home Screen in three pixel-glyph lines, then picks up the conversation in the installed app. No banner, no system-looking modal, no prompt the learner did not tap for.

## 2. Flow
```
growth moment done (active=false)  ─┐
hub open on a later day, loaded   ──┼─> eligible? ──no──> nothing
                                    │       │yes
                                    │   platform
            ┌───────────────────────┼───────┼──────────────────────┬──────────────┐
            standalone          chromium+bip          ios tab (≥16.4)      other w/ push ok
            │                   step 1 "Cài vào máy"  step 1 recipe        │
            │                   │accepted/dismissed   │(learner leaves,    │
            │                   ▼                     │ opens Home icon)   │
            └────────────> step 2 "Bật nhắc lúc t" <──┘  (standalone app)  ┘
                                │ enable(time)
            on ──> toast "Tớ sẽ nhắc cậu lúc t." + card gone for good
            denied ──> denied line + "Đóng" ──> gone for good
            error/invalid ──> ember line, retry in place
            permission 'default' (prompt closed) ──> step 2 unchanged
"Để sau" ──> card hidden for today; reappears on a later day only if shownCount < 2
```
**Eligible** = on `/` (never `/learn`, `/revive`, onboarding) · pet and quest regions loaded without error · not the empty (`no_active_roadmap`) state · `health_points > 0` (the revive band owns a wilted hub) · growth moment not `active` · `everMet` · `nudge.done` false · `nudge.shownCount < 2` or `nudge.lastShownOn == today` · `settings.remindersOn` false · `Notification.permission !== 'denied'` (when `Notification` exists) · `localStorage` available · platform ≠ `unsupported`.
**Platform** (`useInstallPrompt().platform`, resolved when the card first renders on a day):
1. `standalone` — `matchMedia('(display-mode: standalone)').matches || navigator.standalone === true` → step 2 only, and only if `pushSupport(vapid) === 'ok'`, else `unsupported`.
2. `ios` — iPhone/iPad UA (incl. iPadOS "Macintosh" with `maxTouchPoints > 1`), not standalone, iOS ≥ 16.4 when the UA carries a version (no version → eligible) → step 1 recipe. Checked **before** `pushSupport`, which is `unsupported` in an iOS tab by design.
3. `chromium` — a captured `beforeinstallprompt` exists and push is `ok` → step 1 install.
4. `other` — push `ok`, no event (desktop Firefox, Chromium after the event was spent) → step 2 only.
5. `unsupported` — anything else, or `pushSupport` is `no-key` → the card never renders.

Returning screens: none change. `/settings` reads the same store, so after step 2 succeeds it shows "Đang nhắc lúc t".

## 3. Layout (mobile-first, `max-w-md`; desktop: same column)
```
│ [av] Hendrix        [🔥 x1]  [🛡][🛡]  │ status bar (retro-hub 1)
│ ┌ Mầm Non ────────────────────────┐    │ companion panel (retro-hub 2)
│ └──────────────────────────────────┘    │
│ ┌ Mầm Non ────────────────────────┐    │ ← InstallNudge: RetroPanel speaker=plantName, tone=plain
│ │ BƯỚC 1/2                         │    │   eyebrow VT323 16 ink-1 (only when step 1 exists)
│ │ Cài Học 30 phút vào màn hình     │    │   line font-body 17 ink-0
│ │ chính để tớ nhắc cậu lúc 20:00   │    │   time in VT323 20 torch, inline
│ │ mỗi ngày nhé.                    │    │
│ │ [        CÀI VÀO MÁY        ]    │    │   RetroButton primary block
│ │            Để sau                │    │   text button 44 px, VT323 20 ink-1, centred
│ └──────────────────────────────────┘    │
│ PHÒNG HÔM NAY               30/30      │ day bar + path (retro-hub 3–4), pushed down
```
- **iOS recipe (step 1):** eyebrow "Bước 1/2"; line; then an ordered list, one row per step, 16-px gap: `PixelArt` glyph 24 px (`share`, `addHome`, the companion via `CompanionSprite size=32`) + `font-body 16 ink-0` text; then only "Để sau" (secondary `RetroButton`, not a text button — it is the only control).
- **Step 2:** eyebrow "Bước 2/2" only if step 1 rendered in this browsing context; line; primary button. **Time unknown** variant: line "Chọn giờ tớ nhắc cậu mỗi ngày:", then the native `<input type="time">` in the same `RetroPanel frame=input` treatment `/settings` uses (VT323 20 digits, 48 px), value `DEFAULT_REMINDER_TIME`, then the button "Bật nhắc học".
- **Status line** (error / offline / denied): `font-body 16`, directly above the button; error in an ember strip (`tone=ember` inner band as `/settings`), offline and denied in `ink-1`.
- Data: `pet.status.plant_name`, `pet.status.health_points`, `pet.status.last_practiced_at` (`GET /pet/status`); `quest.daily.is_target_met` (`GET /quests/daily`); `settings.notificationTime`, `settings.timeKnown`, `settings.remindersOn`; `useReminders().state`; `useInstallPrompt()`; `runtimeConfig.public.vapidPublicKey`.

## 4. States
| State | What renders |
|---|---|
| Not eligible / loading / hub error / empty / wilted | nothing (no placeholder, no skeleton) |
| Growth moment playing | nothing until `active` turns false (≤ 2 s after landing), then the card in one frame |
| Step 1 — chromium | line + "Cài vào máy" + "Để sau" |
| Step 1 — installing | "Cài vào máy" `loading` while the native prompt is open; "Để sau" disabled |
| Step 1 — ios | recipe list + "Để sau" |
| Step 2 — time known | line with the time + "Bật nhắc lúc {t}" + "Để sau" |
| Step 2 — time unknown | time field + "Bật nhắc học" + "Để sau" |
| Step 2 — requesting (`requesting`) | button `loading`, "Để sau" disabled; the browser's own permission prompt is on top |
| Permission closed without choice (`off`) | step 2 unchanged; the learner can tap again |
| Success (`on`) | card removed; `RetroToast tone=growth` "Tớ sẽ nhắc cậu lúc {t}." once; `nudge.done = true` |
| Denied (`denied`) | line replaced by the denied copy (ink-1), single secondary button "Đóng"; `nudge.done = true` at the moment of denial |
| Subscribe/save failure (`error`/`invalid`) | ember status line "Không lưu được. Thử lại."; button stays and retries `enable(time)` |
| Offline (`navigator.onLine === false`) at step 2 | button disabled, ink-1 line "Cần mạng để lưu cài đặt."; step 1 works offline (install is local) |
| Reduced motion | identical — the card has no motion of its own; the toast uses the kit's one-step fade |

First-time is the whole point: the first eligible hub paint is the first showing. There is no empty state.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| "Cài vào máy" | button drops, `loading` | `deferredPrompt.prompt()`; await `userChoice` → `accepted` or `dismissed` both go to step 2 (the event is spent; `appinstalled` also flips `installed`) | — |
| "Bật nhắc lúc {t}" / "Bật nhắc học" | drop, `loading` | `useReminders().enable(t)` → store `saveReminder` → `POST /api/v1/settings/notifications` `{notification_time: "HH:MM:00", timezone, push_subscription}` (body unchanged, backend spec §6.4) | server + `aelp.settings` (`remindersOn`, `notificationTime`) |
| Time field (unknown variant) | native picker | local value only | sent with enable |
| "Để sau" | inset | card hidden until the next local date | `aelp.nudge.lastShownOn` already today |
| "Đóng" (denied) | drop | card removed | `aelp.nudge.done` |

**Showing rule.** A *showing* is a local date (`YYYY-MM-DD`, device clock) on which the card rendered at least once. On the first render of a date: `shownCount += 1`, `lastShownOn = today`. Leaving and returning to the hub the same day re-shows the card without counting (unless "Để sau" was tapped — then it stays hidden today). After two showings it never renders again; `/settings` remains the path. State lives in `localStorage['aelp.nudge'] = {shownCount, lastShownOn, done, dismissedOn}` through the same `storageOrNull()` guard as the settings store; unparsable JSON is removed and treated as fresh; no storage → never render.
**Capture.** `beforeinstallprompt` is captured app-wide from first boot by a client plugin (`plugins/installPrompt.client.ts`) that calls `e.preventDefault()` and stores the event in `useInstallPrompt`'s module state, so an event fired on `/login` is still usable on `/` later. `appinstalled` clears the event and sets `installed`.
Focus is never moved to the card; `RetroPanel` with `speaker` is `aria-live="polite"`, so it is announced once when it appears.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `components/hub/InstallNudge.vue` | new | props `plantName: string`, `everMet: boolean`, `momentActive: boolean`; no events (owns its store writes) | renders `null` when not eligible; mounted in `pages/index.vue` directly after the companion panel |
| `composables/useInstallPrompt.ts` | new | `platform: Ref<'standalone'\|'ios'\|'chromium'\|'other'\|'unsupported'>`, `canPrompt`, `installed`, `prompt(): Promise<'accepted'\|'dismissed'>`, `listen()` (plugin) | pure detection helpers exported for the matrix test (`detectPlatform(win, vapid, event)`) |
| `utils/nudge.ts` | new | `loadNudge()`, `recordShowing(today)`, `dismissToday(today)`, `markDone()`, `isEligibleByCount(state, today)` | the twice-max rule, unit-tested without Vue |
| `utils/reminderCopy.ts` | new | `REMINDER_COPY = {denied, error, offline}` | the three strings moved **out of** `pages/settings.vue`'s inline `reminderLine` map and imported by both; nothing is duplicated |
| `stores/settings.ts` | existing, +1 field | `timeKnown: boolean` — true after `hydrate()` finds a string `notificationTime`, after `rememberTime()`, after a successful `saveReminder()` | the only store change |
| `useReminders`, `RetroPanel`, `RetroButton`, `RetroToast`, `PixelArt`, `CompanionSprite` | existing | as shipped | — |

**Kit additions** (land through the plan, in `utils/pixelArt.ts` `GLYPHS`, 16-unit grid, `line-lit` + `ink-0`): `share` (square tray with an up arrow — the iOS Share icon read in pixels) and `addHome` (square with a plus). Reason: the recipe must show the icon the learner has to find; words alone ("Chia sẻ") fail when Safari shows only the glyph. No new token, no new component.

## 7. Copy
Speaker tab: `{plant_name}`; when missing/blank → "Cây xanh" (plant-name.md fallback). The name appears only in the tab; lines say "tớ", so no sentence breaks when the name is absent.
- Eyebrows: "Bước 1/2" · "Bước 2/2"
- Step 1 chromium line: "Cài Học 30 phút vào màn hình chính để tớ nhắc cậu lúc {t} mỗi ngày nhé." · button "Cài vào máy" (not "Cài đặt" — that is the settings screen's name)
- Step 1 ios line: "Trên iPhone, tớ chỉ nhắc được khi Học 30 phút nằm trên màn hình chính:" · rows: "Chạm [share] Chia sẻ trên thanh trình duyệt." · "Chọn [addHome] Thêm vào Màn hình chính." · "Mở Học 30 phút từ màn hình chính — tớ đợi cậu ở đó." (on iPad the line reads "Trên iPad, …")
- Step 1 when the time is unknown: "{t}" is replaced by the clause-free line "Cài Học 30 phút vào màn hình chính để tớ nhắc cậu học mỗi ngày nhé."
- Step 2 known: "Bật thông báo để tớ nhắc cậu lúc {t} mỗi ngày nhé." · button "Bật nhắc lúc {t}"
- Step 2 unknown: "Chọn giờ tớ nhắc cậu mỗi ngày:" · field label (visually hidden) "Giờ nhắc" · button "Bật nhắc học"
- Dismiss: "Để sau" · denied close: "Đóng"
- Success toast: "Tớ sẽ nhắc cậu lúc {t}."
- Reused verbatim from `REMINDER_COPY` (origin `settings.md` §4.1 / `retro-settings.md`): denied "Trình duyệt đang chặn thông báo. Mở cài đặt trang web, cho phép Thông báo, rồi thử lại." · error "Không lưu được. Thử lại." · offline "Cần mạng để lưu cài đặt."
`{t}` is `HH:MM` (24 h, as stored), never seconds.

## 8. Self-critique
- **Traded away:** a persistent "install" entry point on the hub. After two showings the only path is `/settings`; a learner who dismissed twice and later wants reminders must find the inn. Accepted — a third ask is nagging, and `/settings` already explains the iOS recipe.
- **iOS gap:** Safari gives no signal that the learner followed the recipe; the tab simply keeps its card until "Để sau" or the second showing. In the standalone app the count restarts (separate storage), so an iPhone learner can see up to two showings in each context. Intentional: the standalone showing is the one that works.
- **Existing learners:** `everMet` from `last_practiced_at` means anyone who met a day before this ships sees the card on their next hub open, without a fresh growth moment. Acceptable and arguably wanted.
- **Per device, not per user:** `aelp.nudge` survives sign-out; a second account on the same phone inherits the count. Low stakes.
- **Executor traps:** (1) never call `requestPermission` outside the button's click handler — a call after an `await` of anything but the prompt loses the user-gesture on Safari; `enable()` calls it first, keep it that way. (2) check iOS before `pushSupport`. (3) never post `DEFAULT_REMINDER_TIME` when `timeKnown` is false without the learner seeing the field. (4) render nothing — not a hidden panel — when ineligible, so the hub layout does not shift. (5) wait for `active === false`, not a timer. (6) `prompt()` may be called once per event.
- **Review checks:** the matrix below; that `pages/settings.vue` now imports `REMINDER_COPY`; no `rounded-*` > 2 px; the card is absent on `/learn/*` and `/revive`.

## Acceptance
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
