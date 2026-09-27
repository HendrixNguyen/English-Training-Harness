# Design: Retro settings — the inn (`/settings`)

**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`
**Inherits:** `harness/designs/settings.md` — its wire contracts (`POST /api/v1/settings/notifications` `{notification_time, timezone?, push_subscription?}` → `{status, notification_time, next_reminder_at}`; `POST /api/v1/integrations/google/sync` → `{status, calendar_event_id, tasks_created_count}`), its states and its "reminder time is the hero" decision are unchanged. `frontend-shell.md` §2.7 (sign out lives here).
**Kit:** `harness/UI-KIT.md` v2.
**Spec wireframe:** none (§5 row "Settings & Integration Modal" only). Departures from `settings.md`: none in structure; the two cards become two panels with an innkeeper framing, and sign out is drawn as "Rời quán" in a danger secondary.

## 0. Research
- **Learner's job:** set the reminder time, sync the 30 minutes into Google, and leave — in under a minute, twice in a lifetime.
- **The moment that earns the next minute:** none needed; the screen's success is that it is finished quickly and the learner knows it worked ("Đang nhắc lúc 20:00").
- **What today does wrong:** `pages/settings.vue` is a placeholder with two disabled buttons; `settings.md` designs the real thing in v1 style.
- **Open questions, answered:** (a) *Does the inn have a keeper character?* No new character — the companion is the only sprite in the product; the inn is a framing in copy ("Quán trọ") and the companion's portrait sits on the panel. (b) *Where is sign out?* Bottom of the page, secondary danger, with a confirm panel.

## 1. The signature
**A menu, in the JRPG sense.** Two panels with a `▶` cursor that follows focus, each holding one fact and one verb. Status is a sentence in the panel, never a toggle without words: "Đang nhắc lúc 20:00 · lần tới 20:00 hôm nay." Nothing decorates; the only motion is the cursor.

## 2. Flow
Entry: hub avatar. Exits: "‹ Trại" → `/`; "Rời quán" → confirm → sign out → `/login`. Returning to the hub changes nothing there.

## 3. Layout (mobile-first, `max-w-md`)
```
│ ‹ Trại                                   │ 1 top bar
│ QUÁN TRỌ                                 │ 2 eyebrow VT323 16; title "Cài đặt" VT323 28
│ ┌ Nhắc học ────────────────────────┐     │ 3 reminder panel (speaker tab = section name; no portrait)
│ │ ▶ Giờ nhắc   [ 20:00 ▼ ]         │     │   time field (RetroPanel frame=input), VT323 20 digits
│ │ Đang nhắc lúc 20:00 · lần tới     │     │   status sentence font-body 16 ink-1
│ │ 20:00 hôm nay.                    │     │
│ │ [     LƯU GIỜ NHẮC     ]          │     │   RetroButton primary; label per state (settings.md §4.1)
│ └───────────────────────────────────┘     │
│ ┌ Lịch Google ─────────────────────┐     │ 4 Google panel
│ │ Đưa 30 phút mỗi ngày vào lịch và  │     │   one sentence font-body
│ │ nhiệm vụ Google của cậu.          │     │
│ │ Đã đồng bộ · 28 nhiệm vụ.         │     │   status after success
│ │ [   ĐỒNG BỘ VỚI GOOGLE   ]        │     │   secondary → "Đã đồng bộ" disabled after success
│ └───────────────────────────────────┘     │
│ [ Rời quán ]                             │ 5 sign out: secondary with ember text, 48 px, not full-width
```
Data: reminder — `notification_time`, `timezone` (`Intl`), push subscription from the SW when permission is granted; response `next_reminder_at` → the status sentence. Google — response `tasks_created_count`. Desktop: same column.

## 4. States
Per `settings.md` §4, restyled:
- **Reminder:** idle (no time yet) "Chưa đặt giờ nhắc." · saving (loading dots) · saved "Đang nhắc lúc {t} · lần tới {relative}." · push denied "Trình duyệt chặn thông báo — giờ vẫn được lưu, tớ nhắc qua lịch Google." (ink-1, not ember) · error ember strip "Chưa lưu được giờ nhắc. Thử lại."
- **Google:** idle · syncing · synced "Đã đồng bộ · {n} nhiệm vụ." · error "Google chưa nhận lịch. Thử lại." · not connected (`google_not_linked`) "Đăng nhập lại bằng Google để cấp quyền lịch." + secondary "Đăng nhập lại".
- **Loading (page):** panels render with their static copy; nothing to fetch (no GET exists; the last saved time comes from the auth store's user if present).
- **Empty:** n/a. **Offline:** buttons disabled with toast "Cần mạng để lưu cài đặt."
- **First-time:** reminder shows the time chosen at onboarding; status "Đang nhắc lúc {t}." from the stored user.
- **Sign out confirm:** `RetroPanel tone=ember` sheet "Rời quán? {plant_name} sẽ đợi cậu ở đây." with danger "Rời quán" and secondary "Ở lại".
- **Reduced motion:** cursor static.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| Time field | cursor moves to the row | validates `HH:MM` | local until saved |
| "Lưu giờ nhắc" | drop → loading | request push permission if not decided → `POST /settings/notifications` → status sentence | server |
| "Đồng bộ với Google" | drop → loading | `POST /integrations/google/sync` → "Đã đồng bộ · n nhiệm vụ." | server |
| "Rời quán" | drop | confirm sheet | — |
| Confirm "Rời quán" | drop | `auth.signOut()` → `/login` | token cleared |
| "‹ Trại" | drop | `/` | — |

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel` | kit | `speaker` as section name, `frame=input` for the field, `tone=ember` for the sheet | — |
| `RetroButton`, `RetroToast`, `StateBlock` | kit | — | — |
| `ConfirmSheet` | new (kit addition) | `title`, `confirmLabel`, `cancelLabel`; emits `confirm`, `cancel` | a bottom `RetroPanel` over a `ground-0` 80 % scrim; focus trapped; Esc cancels |
**Kit additions:** `ConfirmSheet` (reason: sign out is the first destructive action in the product and the kit has no modal).

## 7. Copy
"‹ Trại" · "Quán trọ" · "Cài đặt" · "Nhắc học" · "Giờ nhắc" · "Chưa đặt giờ nhắc." · "Lưu giờ nhắc" · "Đang nhắc lúc {t} · lần tới {relative}." · "Trình duyệt chặn thông báo — giờ vẫn được lưu, tớ nhắc qua lịch Google." · "Chưa lưu được giờ nhắc. Thử lại." · "Lịch Google" · "Đưa 30 phút mỗi ngày vào lịch và nhiệm vụ Google của cậu." · "Đồng bộ với Google" · "Đã đồng bộ · {n} nhiệm vụ." · "Google chưa nhận lịch. Thử lại." · "Đăng nhập lại bằng Google để cấp quyền lịch." · "Đăng nhập lại" · "Rời quán" · "Rời quán? {plant_name} sẽ đợi cậu ở đây." · "Ở lại" · "Cần mạng để lưu cài đặt."

## 8. Self-critique
- Traded away: nothing structural from `settings.md`; the risk is tone — an inn framing on a utilities page can feel forced. It is held to one word ("Quán trọ") and one sheet line; if review finds it cute, the eyebrow reverts to "Cài đặt".
- "Rời quán" for sign out is idiomatic only inside the frame; the confirm sheet says what it does in plain words.
- Executor traps: the push-permission prompt must come from the tap, not on page load; the denied branch is informational (ink-1), not an error; the time on the wire stays `HH:MM:SS`.
- Review checks: status sentence after save uses the server's `next_reminder_at`; sync button disables after success; the sheet traps focus and Esc cancels.

## Acceptance
1. Two panels render with section names on their tabs and one primary verb each; no toggles without words.
2. Saving the reminder posts `HH:MM:SS` + timezone (+ subscription when granted) and the status sentence reflects the server's `next_reminder_at`.
3. A denied push permission still saves the time and shows the informational line, not an error.
4. Google sync shows the task count on success and the exact error strings on `google_not_linked` and other failures.
5. Sign out requires the confirm sheet; confirming clears the session and lands on `/login`.
6. Offline disables both primary buttons with one toast.
7. The `▶` cursor follows keyboard focus; the sheet traps focus.
