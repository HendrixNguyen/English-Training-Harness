# Design: `/settings` — daily reminder and Google sync

**Idea:** `harness/ideas/2026-09-24-run-01/settings-screen-wires-web-push-reminders-and-google-calendar.md`
**Inherits:** `harness/designs/frontend-shell.md` (palette, type, card/button shapes, Vietnamese UI register, page-state convention). This document only adds what the shell design did not draw: the Frontend spec §5 row "Settings & Integration Modal" has no §7 wireframe, so the layout below is the wireframe.
**Wire contracts:** backend spec §6.4 — `POST /api/v1/settings/notifications` `{notification_time, timezone?, push_subscription?{endpoint, p256dh, auth}}` → `{status: "updated", notification_time, next_reminder_at}`; `POST /api/v1/integrations/google/sync` → `{status: "synced", calendar_event_id, tasks_created_count}`; error envelope `{error: code}`.

**Subject and job.** The same learner as every other screen, on the one day they open settings: right after onboarding, or the day they realise they keep forgetting. The page's single job is to make the two things the server already does for them — nudge them at a time of their choosing, and put the 30 minutes in their Google calendar — actually happen. It is a page you visit twice in your life; it must be finished in under a minute and never leave you guessing whether it worked.

**Register.** Vietnamese UI copy, sentence case, one verb per action kept through the flow ("Bật nhắc học" → "Đang nhắc lúc 20:00"; "Đồng bộ với Google" → "Đã đồng bộ"). Errors say what happened and what to do next. No marketing.

---

## 1. The signature: the reminder time is the hero

The page opens with the time, not a form. The plant's speech bubble (the existing `SpeechBubble`) says when it will call — *"Mình sẽ nhắc bạn lúc **20:00**."* — and the time itself is set in **Fraunces at the hero size (40/44)**, the same treatment the home screen gives the "20 / 30" minute figure. Tapping the time opens the native `<input type="time">` underneath it; the bubble re-reads live as the value changes. This is the one bold move on the page: the shell reserves Fraunces for two sizes, title and hero number, and the reminder time is a hero number — it is the whole point of the screen. Everything else is quiet body text in cards.

Why it earns its place: the structure encodes the true fact that a reminder is a time, and it puts the plant — the retention character — in the role of the one doing the reminding, which is exactly what the push notification will look like when it arrives (`DEFAULT` payload: "Cây của bạn đang chờ bạn").

---

## 2. Layout (mobile, one column, `max-w-md`, 8-pt grid)

```
┌──────────────────────────────────────┐
│ AppHeader (avatar · name · Lộ trình) │
│ Cài đặt                     (title)  │
├──────────────────────────────────────┤
│ NHẮC HỌC MỖI NGÀY            (card)  │
│  💬 "Mình sẽ nhắc bạn lúc 20:00."    │  ← SpeechBubble, time in Fraunces hero
│        [ 20 : 00 ]  (native time)    │
│  ─────────────────────────────────── │
│  Bật nhắc học                 (◯━━)  │  ← AppSwitch, growth when on
│  Đang tắt · Bạn sẽ không nhận thông  │  ← one status line under the switch
│  báo.                                │
│  [ Lưu giờ nhắc ]              (btn) │  ← only when time changed & saved≠current
├──────────────────────────────────────┤
│ GOOGLE LỊCH & NHIỆM VỤ       (card)  │
│  Tạo một sự kiện học 30 phút lặp mỗi │
│  ngày và một danh sách 28 nhiệm vụ.  │
│  [ Đồng bộ với Google ]   (primary)  │
│  Đã đồng bộ · 28 nhiệm vụ · 20:14    │  ← status line after success
└──────────────────────────────────────┘
```

Two cards, in the order the §5.1 onboarding flow does them (reminder → Google). No tabs, no modal (the spec's word "modal" is from a desktop mindset; on a phone a page is the modal). The status line under each control is the only place state is written, so the eye always knows where to look.

---

## 3. Components

| Component | Status | Role |
| --- | --- | --- |
| `AppHeader`, `AppCard`, `AppButton`, `StateBlock` | existing | shell |
| `SpeechBubble` | existing | the hero line; `line` prop receives the rendered sentence |
| `AppSwitch` (`components/ui/AppSwitch.vue`) | **new** | `role="switch"`, `aria-checked`, 44×24, track `mute/30` → `growth`, knob white, 200 ms ease, `prefers-reduced-motion` → instant; label is the visible text beside it (`aria-labelledby`), never a bare icon |
| native `<input type="time">` | — | styled as a card field: `rounded-btn border border-ink/15 h-12 px-4 font-display text-2xl tabular-nums`; the hero number above it is the same value, so the input is the editor and the bubble is the display |

No new icons. No custom time picker (the native one is what the learner's phone already knows).

---

## 4. States

### 4.1 Reminder card

| State | Switch | Status line (under the switch) | Notes |
| --- | --- | --- | --- |
| `off` (default; no subscription known) | off | "Đang tắt · Bạn sẽ không nhận thông báo." | time still editable; saving the time alone posts without `push_subscription` |
| `requesting` | off, disabled | "Đang xin phép trình duyệt…" | `Notification.requestPermission()` + `pushManager.subscribe` in flight; switch shows the AppButton spinner style |
| `on` | on | "Đang nhắc lúc **20:00** mỗi ngày." | after a 200; `next_reminder_at` is not shown (the time is what the user chose; the server's UTC instant would only confuse) |
| `denied` | off, disabled | "Trình duyệt đang chặn thông báo. Mở cài đặt trang web, cho phép Thông báo, rồi thử lại." | `Notification.permission === 'denied'`; no retry button — the browser owns this, we say where to go |
| `unsupported` | hidden | "Thiết bị này chưa nhắc được qua trình duyệt. Trên iPhone: Chia sẻ → Thêm vào Màn hình chính, rồi mở lại." | no `PushManager`/`serviceWorker`, or iOS Safari not installed as PWA |
| `no-key` | hidden | *(nothing)* | `NUXT_PUBLIC_VAPID_PUBLIC_KEY` empty: the switch and its line are not rendered; the time field and "Lưu giờ nhắc" stay (they need no key). Block is hidden, not broken. |
| `error` | unchanged | "Không lưu được. Thử lại." (`alert` text) | network / 500 / 404 `user_not_found`; the switch returns to its previous position |
| `invalid` | unchanged | "Giờ nhắc không hợp lệ." | 400 `invalid_request` — unreachable with the native input, but the code path exists |

Turning the switch **off**: `PushSubscription.unsubscribe()` client-side, status line → `off`. There is no unsubscribe endpoint; the server prunes the dead endpoint on its next 404/410 (CODEMAP `notify`). This is stated in the plan, not to the user.

"Lưu giờ nhắc" appears only when the time differs from the last saved value; success collapses it and the bubble re-reads. Saving with reminders `on` re-sends the existing subscription so the server re-slots `queue:webpush:delay` to the new time.

### 4.2 Google card

| State | Button | Status line |
| --- | --- | --- |
| idle | "Đồng bộ với Google" (primary) | *(none)* — the sentence above the button says what will happen |
| loading | same, `loading` spinner | "Đang đồng bộ…" |
| synced | "Đồng bộ lại" (ghost) | "Đã đồng bộ · **28** nhiệm vụ · hôm nay 20:14" — count from `tasks_created_count`, time from the client clock, persisted locally so it survives a reload |
| `reauth_required` (409) | "Cho phép lại với Google" (primary) | "Google cần bạn cho phép lại để ghi lịch và nhiệm vụ." — button routes to the same consent URL `/login` builds (`googleAuthUrl`, already `prompt=consent`) |
| `google_unavailable` (502) | "Thử lại" (primary) | "Google chưa phản hồi. Thử lại sau ít phút." |
| other error (500, network) | "Thử lại" (danger) | "Không đồng bộ được. Thử lại." |

### 4.3 Page-level

- Guarded route (existing middleware). No loading skeleton is needed: there is no `GET`, the page renders from local state instantly.
- Prefill: the time is the last value this client submitted (onboarding writes it; this page writes it), default `20:00`. No read endpoint exists; adding `GET /api/v1/settings` is a later backend idea, not this ticket — the design accepts that a second device shows the default until you save once.
- Onboarding's result step gains one ghost link under "Xem nhiệm vụ hôm nay": **"Bật nhắc học và đồng bộ Google →"** → `/settings`. Nothing else is duplicated onto onboarding.

---

## 5. Tokens (all existing; nothing new is registered)

- Colour: `growth` (switch on, primary buttons, the hero time), `mute` (status lines, switch off track), `alert` (error lines, danger retry), `ink`/`paper` surfaces per shell.
- Type: hero time `font-display text-[40px] leading-[44px] tabular-nums`; card labels `text-xs font-semibold uppercase tracking-wider text-mute` (existing `AppCard` title); status lines `text-sm text-mute` / `text-sm text-alert`.
- Shape: cards 16 px, buttons and the time field 12 px, switch fully rounded.
- Motion: switch knob 200 ms; bubble text change none (it is text). Reduced motion → no transition.
- Focus: shell's `ring-2 ring-growth ring-offset-2` on switch, field and buttons.

## 6. Self-critique

Removed before shipping: a "next reminder in 3 h 12 m" countdown (a second hero number competing with the first; and it would drift), a per-device list of subscriptions (no endpoint, no need), and a success toast (the status line already says it, in place). The one risk taken is spending Fraunces on the time; it is defensible because the shell already defines "hero number" as a Fraunces role, and the time is this page's only number that matters.
