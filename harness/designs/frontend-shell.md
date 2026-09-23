# Design: Frontend shell — Nuxt 3 PWA

**Idea:** `harness/ideas/2026-09-22-run-02/frontend-shell-nuxt-3-pwa-with-auth-daily-quest-and-pet-scre.md`
**Source of truth:** *Frontend Technical Specification* §6.1 (visual identity) and §7 (five wireframes). Wire shapes: *Backend Technical Specification* §6. Where this document says "the spec", it means the Frontend spec.

**Subject and job.** A Vietnamese learner of English opens the app once a day to spend 30 minutes on three ten-minute tasks and keep a virtual plant alive. Every screen exists to get them into a task or show them what their time did to the plant. Nothing else earns a pixel.

**Register.** UI copy is Vietnamese, as drawn in every §7 wireframe; learning content (the exercises themselves) is English. Copy is sentence case, plain verbs, and one name per action through a whole flow ("Hoàn thành" on the button → "Đã hoàn thành" on the task row). Errors say what happened and what to do; empty screens are an invitation to act.

---

## 1. Visual identity (from §6.1)

### Palette — exactly the four spec colours plus neutrals

| Token | Hex | Spec role (§6.1) | Used for |
| --- | --- | --- | --- |
| `growth` | `#10B981` (Emerald) | plant growth, success, completed lessons | primary buttons, filled progress segments, completed task rows, healthy plant leaves |
| `streak` | `#F59E0B` (Amber) | streak counts, rewards, milestones | streak chip, roadmap "completed" stars, flowering/fruitful accents on the plant |
| `alert` | `#EF4444` (Coral) | health depletion warnings, wilted alerts | health bar under 30 %, wilted plant tint, revival banner, destructive confirms |
| `ink` | `#1E293B` (Slate) | high-contrast text and containers | body text (light mode), card surfaces (dark mode) |
| `paper` | `#F8FAFC` | — (neutral, derived) | page background, light mode |
| `paper-dark` | `#0F172A` | — (neutral, derived) | page background, dark mode |
| `mute` | `#64748B` | — (neutral, derived) | secondary text, locked roadmap nodes, empty progress segments |

Tailwind: the four spec colours are registered under the token names above so no component ever writes a hex value. Dark mode follows the OS (`prefers-color-scheme`); the only inversion is `ink`/`paper` swapping roles — the three semantic colours are identical in both modes and are checked for contrast against both backgrounds.

### Typography

- **Display — Fraunces** (variable, soft optical size): the plant's speech bubble, screen titles, the big "20 / 30" number. Chosen because its soft serifs read as organic and friendly next to a plant; it is used at two sizes only (title, hero number) so it stays a signature rather than a wallpaper.
- **Body — Source Sans 3**: everything else. Neutral, tall x-height, comfortable at 15–16 px on a phone. Digits set tabular for the countdown so the timer does not jitter.
- Scale (mobile): title 28/32, hero number 40/44, section label 12/16 uppercase tracked (the wireframes' `TIẾN ĐỘ HÔM NAY`), body 16/24, caption 13/18.
- Fonts are self-hosted so the service worker can precache them (§3 — the shell must render offline).

### Shape, spacing, motion

- Cards: 16 px radius, 1 px `ink`/8 % border, no shadows (minimalist). Buttons: 12 px radius, 48 px tall on touch.
- Spacing on an 8-pt grid; one column, `max-width 28 rem` centred, so the phone layout is the desktop layout "auto-scaled" (§6.1).
- Motion is spent in one place: the plant. Segment fills and health-bar changes ease over 400 ms; the plant sways gently while healthy and droops (a single rotate + desaturate) when wilted. Everything respects `prefers-reduced-motion` (no sway, instant fills).
- Visible focus rings (`growth`, 2 px offset) on every interactive element.

### The signature: a three-segment day

The 30-minute daily target is not a percentage bar. It is **three segments of ten minutes**, one per task (§6.2: "3x 10-min tasks"), laid under the plant like three watering-can pours. A segment fills left to right as `accumulated_seconds` grows past 0, 600, 1200; the label reads `TIẾN ĐỘ HÔM NAY: 20 / 30 PHÚT` exactly as wireframe 7.2 writes it, with the minute figure in Fraunces. When all three are full the bar turns solid `growth` and the plant's bubble says the target is met. The structure encodes a true fact (three tasks, ten minutes each, thirty in a day), which is why it is allowed to be a structural device.

---

## 2. Screens

Six routes. Each section names the wireframe it implements, the layout top to bottom, the data it binds, and its loading / empty / error states. Guarded routes redirect to `/login` when there is no token or the stored token is past its `expires_in`.

### 2.1 `/login` — sign in (wireframe 7.1, upper half)

Layout: centred column. Plant mark (the `sprout` SVG, small) → title "Chào mừng bạn! 🌱" (Fraunces) → one line "Học 30 phút mỗi ngày, nuôi một cái cây." → primary button "Đăng nhập bằng Google" → caption "Bằng cách tiếp tục, bạn cho phép ứng dụng đọc lịch và nhiệm vụ Google của bạn để lên lịch học." (the consent covers `calendar.events` + `tasks` at first sign-in — the user should know why).

Binds: the Google consent URL (client id + `redirect_uri` = this page + random `state`); on return with `?code=` posts `{code, redirect_uri}` to `POST /api/v1/auth/google` and stores `{access_token, expires_in, user}`.

- Loading: after the redirect returns, the button is replaced by a spinner and "Đang đăng nhập…"; the code is exchanged once.
- Empty: n/a (this is the empty state of the app).
- Error: inline `alert` card under the button — "Không đăng nhập được. Thử lại." with the button restored; a `state` mismatch shows the same card with "Phiên đăng nhập không hợp lệ" and never posts the code.
- Already signed in: the page redirects to `/` immediately.

### 2.2 `/onboarding` — goal selection and placement (wireframe 7.1, lower half)

Layout: title "Mục tiêu học của bạn là gì?" → two large selectable goal cards side by side, "🎓 IELTS 7.0" and "💼 Business English" (exactly the two the wireframe draws; radio behaviour, `growth` border when selected) → "Chọn giờ nhắc học hằng ngày" with a native time select defaulting to 20:00 → primary "Bắt đầu bài kiểm tra đầu vào" → the placement quiz reuses the learning-room question layout (§2.4) without a timer, one question per screen with a "Câu n / N" counter → on the last question the button reads "Hoàn thành" and submits `{target_goal, notification_time, timezone, answers}` to `POST /api/v1/onboarding/assessment` → a result card "Trình độ của bạn: B1" with the new plant at `sprout` and a button "Xem nhiệm vụ hôm nay" → `/`.

Binds: `GET /api/v1/onboarding/quiz` for questions; `POST /api/v1/onboarding/assessment` for the result. **Both render against a documented stub until the onboarding slice lands**; the screen is real, the data is fixture.

- Loading: skeleton of two goal cards; the quiz shows a skeleton question card.
- Empty: quiz returns no questions → "Chưa có bài kiểm tra. Quay lại sau." with a link back to `/`.
- Error: assessment fails → `alert` card "Không tạo được lộ trình. Thử lại." keeping the answers so nothing is retyped.
- Guard: a user who already has an active roadmap is sent to `/` (the dashboard, not this screen, is home).

### 2.3 `/` — dashboard and plant hub (wireframe 7.2)

Layout, top to bottom, exactly as drawn:
1. **Header row**: avatar (initial in a `growth` circle) + `full_name` on the left; streak chip "🔥 Streak: 5 ngày" in `streak` on the right; a small "Lộ trình" link to `/roadmap`.
2. **Plant hub card**: the plant SVG (see §3) centred at 160 px; under it the health bar "Máu cây: [████████░░] 80 %"; under that the speech bubble in Fraunces — one line chosen from the plant's state (§3).
3. **Day progress card**: label `TIẾN ĐỘ HÔM NAY: 20 / 30 PHÚT`, the three-segment bar, percent caption.
4. **Quest list card**: "Nhiệm vụ hôm nay" and three rows in `task_type` order (vocabulary → reading → practice), each: state glyph `[✓] [▶] [ ]`, index, title, "(10m)", and a trailing button — "Xong" (disabled, `growth` text) when `is_completed`; "Học" (primary) on the first incomplete task; "Khóa" (disabled, `mute`) on the rest. Tapping "Học" goes to `/learn/<id>`.

Binds: `GET /api/v1/pet/status` → hub; `GET /api/v1/quests/daily` → progress + list; `POST /api/v1/quests/progress` responses (from the learning room) refresh both.

- Loading: header renders from the stored user immediately; hub and cards show skeletons; the plant is a faint `sprout` outline.
- Empty (`404 no_active_roadmap`): the plant hub still renders (the pet row exists from first sign-in); the progress and quest cards are replaced by one card "Bạn chưa có lộ trình học" + "Tạo lộ trình 28 ngày" → `/onboarding`.
- Error: each card fails independently — a compact `alert` strip inside the card, "Không tải được. Thử lại", with a retry that refetches only that card. The other card keeps its data.
- Wilted (`health_points == 0`, `stage == wilted`): a full-width `alert` banner above the hub — "⚠️ Cây xanh đang bị héo rũ!" + "Cứu cây ngay" → `/revive`. The quest list stays usable (a full 30-minute day also revives the plant).
- Target met (`is_target_met`): segments solid `growth`, caption "Mục tiêu hôm nay đã đạt ✓", quest buttons for remaining tasks stay "Học" (extra study is allowed).

### 2.4 `/learn/:exerciseId` — distraction-free learning room (wireframe 7.3)

Layout: no app header. A slim top bar: "‹ Quay lại" on the left, "⏱️ Thời gian: 09:42" on the right in tabular digits. Below, a single content card: a "Câu 2 / 10" counter when the content is a question list; the prompt in body type; answer options as full-width selectable rows `(A) (B) (C)`; at the bottom one primary button whose label is "Gửi đáp án" while there is a selected, unsubmitted option and "Tiếp tục" otherwise; on the last item (or for non-question content) it reads "Hoàn thành".

Content rendering by shape of `content_json` (the roadmap task object — its `content` field is free-form):
- `content.words[]` (vocabulary, §6.2 example): one card per word, `term` in Fraunces, `definition` under it, swiped or stepped with "Tiếp tục".
- `content.questions[]` with `prompt` + `options`: the question layout above.
- anything else: the task `title` as heading and the content pretty-printed in a monospace block — legible, never blank.

Timer: counts down from `duration_minutes` (default 10). At 00:00 it holds and the bottom button becomes "Hết giờ — Hoàn thành". "Hoàn thành" posts `{exercise_id, duration_seconds}` where `duration_seconds` is elapsed time clamped to 1..3600, then returns to `/` with the refreshed totals. Leaving via "Quay lại" keeps the timer paused for that task for the session (a re-entry resumes).

- Loading: the task is read from the quest store; if the store is cold the room fetches `GET /quests/daily` first and shows a skeleton card.
- Empty (id not in today's tasks): "Nhiệm vụ này không có trong hôm nay" + "Về trang chính".
- Error on complete (`400 invalid_request`, `404 exercise_not_found`, network): `alert` card above the button — "Chưa ghi được tiến độ. Thử lại." — the elapsed seconds are kept and resubmitted on retry; nothing is double-counted client-side because the button is disabled while a post is in flight.
- Already completed: the room still opens (review), the button reads "Đã hoàn thành" and is disabled.

### 2.5 `/roadmap` — 28-day curriculum tree (wireframe 7.4)

Layout: title "Lộ trình học 28 ngày"; below, 28 nodes in a vertical zig-zag (alternating left/right offset, joined by a thin `mute` line — the wireframe's `\` and `/`). Each node is a pill: glyph + "Ngày n" + status text. Three visual states:
- **Completed** (days before today): ⭐ in `streak`, "Đã hoàn thành".
- **Today** (`day_number`): 🌱, `growth` fill, "HÔM NAY - Đang học", larger; the page scrolls to it on open.
- **Locked** (days after today): 🔒, `mute` text and border, "Chưa mở khóa".

Weeks are separated by a caption "Tuần 1 · Tuần 2 …" every seven nodes, because the roadmap really is four modules of seven days (airouter `RoadmapSchema`). Tapping "today" goes to `/`. Other nodes are inert in this slice.

Binds: `GET /api/v1/quests/daily` → `day_number`. **There is no per-day completion endpoint**, so "completed" means "before today"; recorded as an open question.

- Loading: 28 skeleton pills.
- Empty (`404 no_active_roadmap`): the same "Bạn chưa có lộ trình học" card as the dashboard.
- Error: full-card `alert` strip with retry.

### 2.6 `/revive` — plant revival mode (wireframe 7.5)

Layout: an `alert` header band "⚠️ CÂY XANH ĐANG BỊ HÉO RŨ!"; the wilted plant SVG at 160 px, labelled "Cây héo - 0 %"; a quote card "Bạn đã bỏ học n ngày liên tiếp. Hãy hoàn thành Bài kiểm tra Cứu Cây 15 phút để hồi sinh!" (n derived from `last_practiced_at`; when it is missing the sentence drops the count); one primary `alert`-coloured button.

The button walks the §6.3 revive contract, which the pet slice implements as "record 15 minutes of study on today's counter after the challenge starts":
1. **Wilted, no challenge**: button "🚨 Cứu cây ngay (Quiz 15 phút)". Tapping posts `POST /api/v1/pet/revive`; `revival_passed: false` means the challenge has started.
2. **Challenge running**: the card changes to a 15-minute version of the three-segment bar — **one segment of fifteen minutes**, filled from study recorded since the challenge started — with "Học 15 phút để hồi sinh" and a button "Vào học ngay" → `/` (the quests are the quiz). A secondary button "Kiểm tra hồi sinh" re-posts revive.
3. **Passed** (`revival_passed: true`): the plant animates from wilted to `sprout` at 50 %, "Cây đã hồi sinh! Máu cây: 50 %", button "Về trang chính".

- Loading: skeleton plant + disabled button.
- Not wilted (`409 pet_not_wilted` or `health_points > 0` on load): "Cây của bạn vẫn khỏe 🌱" with "Về trang chính"; the page never shows the alert band for a healthy plant.
- Error: `alert` card "Không bắt đầu được thử thách. Thử lại." with the button restored.

### 2.7 `/settings` — placeholder

One card "Sắp ra mắt" with two disabled buttons "Bật nhắc học" and "Đồng bộ Google" so the notify and google slices attach here without a new screen. Reachable from the avatar in the dashboard header, together with "Đăng xuất" (clears the stored token, goes to `/login`).

---

## 3. The plant

One SVG component, six states, one silhouette so the progression reads as the same plant growing. Every state is a 160 × 160 drawing of a pot (rounded `ink`-tinted trapezoid), a soil line, and what grows from it. Health tints the leaves: 100–60 % full `growth`; 59–30 % `growth` mixed toward `streak`; 29–1 % mixed toward `alert`.

| `stage` | Drawing | When the engine produces it |
| --- | --- | --- |
| `seed` | soil mound with a single dark seed dot; no stem | in the DDL enum, never produced by the merged engine (default is `sprout`) — drawn so an unexpected value still renders |
| `sprout` | one short stem, two round leaves | streak 0–2 (and after revive: 50 % health) |
| `sapling` | taller stem, five leaves alternating, slight lean | streak 3–6 |
| `flowering` | the sapling plus three `streak`-coloured blossoms | streak 7–13 |
| `fruitful` | the flowering plant plus two `streak` fruit orbs, canopy widened | streak 14+ |
| `wilted` | the sapling silhouette rotated 12° toward the soil, leaves desaturated toward `alert`, one leaf on the soil; no sway | `health_points == 0` |

Unknown stage → `sprout` silhouette with a `mute` tint, so a new enum value degrades gracefully.

**Speech bubble** (Fraunces, one line):
- target not met, health ≥ 60: "Tưới cho tớ 10 phút học đi!" (wireframe 7.2)
- target not met, health 30–59: "Tớ hơi khát rồi… 10 phút thôi?"
- target not met, health 1–29: "Tớ sắp héo mất! Học một chút nhé?"
- target met: "Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿"
- wilted: "…" (the revival banner speaks instead)

**Health bar**: 8 px track, `growth` fill, turns `streak` under 60 % and `alert` under 30 %; label "Máu cây: 80 %" beside it. The bar animates between values; the plant does not change stage mid-animation.

---

## 4. Component inventory

| Component | Used by | Notes |
| --- | --- | --- |
| `AppHeader` | dashboard, roadmap, onboarding, settings | avatar + name, streak chip, roadmap link, avatar menu (settings, sign out) |
| `AppButton` | all | variants `primary` (`growth`), `danger` (`alert`), `ghost`; sizes; `loading` state replaces the label with a spinner and keeps width |
| `AppCard` | all | the 16 px-radius container; optional title slot |
| `StateBlock` | all | the shared loading / empty / error block: skeleton lines, or icon + one sentence + one action |
| `SegmentedProgress` | dashboard, revive | `segments`, `segmentSeconds`, `valueSeconds`, `label`; renders the signature bar |
| `HealthBar` | dashboard, revive | health 0–100, colour thresholds above |
| `PlantSvg` | dashboard, login, revive, onboarding result | `stage`, `health`, `size`; the six drawings |
| `SpeechBubble` | dashboard | one line, Fraunces |
| `QuestRow` | dashboard | glyph, index, title, duration, trailing button by state |
| `CountdownTimer` | learning room | remaining seconds → `mm:ss`, tabular digits |
| `ContentViewer` | learning room, onboarding quiz | picks words / questions / fallback renderer by `content_json` shape |
| `RoadmapNode` | roadmap | pill with the three states |
| `GoalCard` | onboarding | selectable card with emoji + label |

---

## 5. PWA shell

- Manifest: name "Học 30 phút", short name "Học30", `growth` theme colour, `paper` background, standalone, portrait; icons generated from the `sprout` mark.
- Offline: the app shell (routes, fonts, plant SVGs) is precached; `GET /quests/daily` and `GET /pet/status` are **NetworkFirst** (§3 — synchronised state wins when online, last-known state when not); static assets and exercise content are **StaleWhileRevalidate** (§3 — cached cards render instantly). Offline the dashboard shows last-known cards with a `mute` caption "Ngoại tuyến — hiển thị dữ liệu lần cuối"; "Hoàn thành" in the learning room is disabled offline with "Cần kết nối để ghi tiến độ" (progress queueing is out of scope for this slice; see the plan's Notes).
- Push: a received push shows a notification with the payload's title/body and opens its `url` on tap. Subscribing is the notify slice's job; the handler is here so the shell needs no second service-worker change.

---

## 6. Accessibility floor

- All states reachable by keyboard; focus visible; the plant SVG has an `aria-label` that states stage and health ("Cây đang ở giai đoạn sapling, máu 80 %").
- Colour is never the only signal: task rows carry the glyph and the button label; roadmap nodes carry the status text; the health bar has a numeric label.
- Touch targets ≥ 44 px. Countdown updates are `aria-live="off"` (a ticking live region is noise); the completion result is announced once.
- `prefers-reduced-motion`: no sway, no droop animation, fills snap.
