# Design: Retro onboarding — title screen to the first encounter (`/login`, `/onboarding`)

**Idea:** `harness/ideas/2026-09-25-run-01/retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, plus the calibration tap from `harness/ideas/2026-09-25-run-01/a-session-a-learner-wants-to-finish-level-true-content-do-to.md`.
**Inherits:** `harness/designs/frontend-shell.md` §2.1–2.2 (OAuth flow, `state` check, assessment errors keep answers, guards), `plant-name.md` (the name field and its rules).
**Kit:** `harness/UI-KIT.md` v2.
**Spec wireframe:** frontend spec §7.1. Kept: welcome + Google button, the two goal cards, the reminder-time picker. Departures: login and goal are two screens, not one (the wireframe stacks them, but the goal step needs a session); the placement quiz is drawn as the first encounter (the spec has no quiz wireframe); a **waiting** step and a **calibration** row are added because roadmap generation takes tens of seconds (providertimeout plan) and the owner was graded A2 with A1 content.

## 0. Research
- **Learner's job:** get from "I want to learn English" to "here is my first quest" with as few decisions as possible, and trust the result.
- **The moment that earns the next minute:** the companion hatching at the end of the placement with the learner's level as a rank — "Cấp B1" — and three concrete day-1 quests they can judge with one tap.
- **What today does wrong:** `pages/onboarding.vue` is three white cards; the quiz has no framing, so ten ungraded questions feel like a form; the result card says "Trình độ của bạn: A2" with no way to object, and the owner's plan opened with greetings and numbers. Generation can take long enough to time out with nothing on screen.
- **Open questions, answered:** (a) *Is the placement graded live?* No — `GET /onboarding/quiz` has no answer keys; the encounter shows no hit/miss, the companion narrates instead ("Tớ đang xem cậu đánh…"). (b) *What does calibration call?* `POST /api/v1/roadmaps/regenerate?level=` proposed by the session idea; **not in the API today**. The row is designed and hidden by a feature flag until that endpoint lands; the plan for this screen depends on it. (c) *Waiting contract?* The executor follows the providertimeout plan's staged path; this doc only fixes what the learner sees while waiting.

## 1. The signature
**The placement is the first encounter, not a form.** The companion (still an egg/seed sprite) stands beside the dialogue box; each question is a "turn"; the eyebrow reads "Lượt 3 / 10"; the seed cracks one frame further every three answers. At the end the seed hatches into `sprout` (`levelup` frames) and the level appears as a rank plate in VT323 34. Nothing is graded on screen — the drama is the hatching, which is true: the assessment creates the pet.

## 2. Flow
`/login` (title) → Google → `/onboarding` step **goal** (goal cards, companion name, reminder time) → **encounter** (10 turns) → **waiting** (generation) → **ready** (level, day-1 quests, calibration) → `/` first-time state. Guards as shell: signed-in users skip `/login`; users with a roadmap skip `/onboarding`. Back from the encounter returns to goal with answers kept. Calibration "Quá dễ"/"Quá khó" → regenerate → waiting → ready again (once; then the row disappears).

## 3. Layout (mobile-first, `max-w-md`)
**Title (`/login`)**
```
│        [seed sprite 128, idle]          │ CompanionSprite stage=seed
│            HỌC 30 PHÚT                  │ VT323 34, title
│   Mỗi ngày một phòng. Một người bạn.    │ font-body 17 ink-1
│ [   ĐĂNG NHẬP BẰNG GOOGLE   ]           │ RetroButton primary (bottom bar)
│ caption: quyền lịch/nhiệm vụ Google     │ font-body 14 ink-2
```
Loading after the redirect: button → loading dots, caption "Đang đăng nhập…". Error: `tone=ember` panel with the shell's strings.

**Goal**
```
│ CHỌN HÀNH TRÌNH                          │ eyebrow
│ ┌────────────┐ ┌──────────────────┐     │ two GoalCard tiles (RetroPanel, role=radio), 96 px,
│ │ [🎓] IELTS │ │ [💼] Business    │     │ selected = growth outer line + ▶ cursor
│ │      7.0   │ │      English     │     │
│ └────────────┘ └──────────────────┘     │
│ Đặt tên cho bạn đồng hành (không bắt    │ plant-name field, RetroPanel frame=input, placeholder "Mầm Non"
│ buộc)  [ Mầm Non              ]         │
│ Giờ nhắc học   [ 20:00 ▼ ]              │ time input, 48 px, VT323 20 digits
│ [   BẮT ĐẦU TRẬN ĐẦU TIÊN   ]           │ primary, disabled until goal + valid time
```
Binds `target_goal`, `plant_name`, `notification_time` (HH:MM:SS), `timezone`.

**Encounter** — the learning-room question panel (`ItemQuestion` with `revealed=false`), eyebrow "Lượt {n} / 10", seed sprite left of the box cracking at turns 4 and 7, button "Chọn" (disabled until an option) → next turn; last turn "Kết thúc". No feedback strip. Binds `GET /onboarding/quiz` → `questions[]`; answers `{question_id, selected_option}`.

**Waiting** — `RetroPanel speaker={plant_name}`: sprite `seed` with a 2-frame wobble; line "Tớ đang vẽ bản đồ cho cậu… (khoảng một phút)"; a `DayBar`-style indeterminate bar (cells sweep, steps(8), 1.6 s); no button. Binds the assessment/generation call; on `ai_*` errors the panel turns ember with the shell's strings and a secondary "Thử lại" (answers kept).

**Ready**
```
│ ┌ Mầm Non ────────────────────────┐     │ RetroPanel speaker
│ │ [sprout 128, levelup once]      │     │
│ │        CẤP B1                    │     │ rank plate VT323 34, torch
│ │ "Tớ nở rồi! Cậu ở cấp B1 —      │     │ SpeechBox
│ │  hành trình Business English."   │     │
│ └──────────────────────────────────┘     │
│ NGÀY 1                                   │ eyebrow
│ [📖] Email openings and closings   10'   │ three QuestNode tiles, state=open, no connector
│ [📜] Reading: a meeting recap      10'   │
│ [🗡] Practice: replying to a client 10'  │
│ Ngày 1 trông thế nào?                    │ calibration row (flagged): three 48-px secondary buttons
│ [ Quá dễ ] [ Vừa sức ] [ Quá khó ]       │   "Vừa sức" = primary tone; others regenerate ±1 level
│ [        VÀO TRẠI →          ]           │ bottom bar, primary
```
Binds `assessed_level`, `pet_state{plant_name, stage, health_points}`, day-1 `tasks[]` from `GET /quests/daily` (loaded before showing this step).

## 4. States
- **Loading:** goal step never loads (static); encounter shows `StateBlock loading` in the box; ready waits on `quests/daily` with the waiting panel.
- **Empty (quiz has no questions):** "Chưa có trận đầu tiên. Quay lại sau." + "Về trang chính".
- **Error:** assessment errors per shell (`rate_limited`, `ai_*`, `invalid_request`) in an ember panel; answers kept; regenerate errors: "Chưa vẽ lại được. Giữ hành trình hiện tại?" with "Giữ" (primary) and "Thử lại" (secondary).
- **Offline:** title and goal work; starting the encounter offline shows `RetroToast` "Cần mạng để bắt đầu trận đầu tiên." and the button stays enabled for retry.
- **First-time:** this whole flow is first-time; nothing is remembered except a kept `plant_name` and answers on error.
- **Done:** "Vào trại" → `/` with the hub's first-day line.
- **Reduced motion:** seed does not crack or wobble — it swaps frames at turns 4/7 and at the hatch without keyframes; the indeterminate bar becomes a static "…" `StateBlock`.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| Google button | drop → loading | OAuth redirect; return exchanges the code once | token (auth store) |
| Goal tile | cursor + growth line | enables start when time valid | `goal` |
| Name field | — | validated 1–30 chars per plant-name | `plant_name` |
| "Bắt đầu trận đầu tiên" | drop | `GET /onboarding/quiz` → encounter | — |
| Option / "Chọn" | cursor / drop | next turn; sprite frame at 4, 7 | `answers[]` |
| "Kết thúc" | drop | `POST /onboarding/assessment` → waiting → ready | pet + roadmap (server) |
| Calibration button | drop, others disabled | regenerate ±1 level → waiting → ready; row hidden after one use | roadmap (server) |
| "Vào trại" | drop | `quest.load()` → `/` | — |

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel`, `RetroButton`, `CompanionSprite`, `SpeechBox`, `QuestNode`, `StateBlock`, `RetroToast` | kit | — | `QuestNode` with `connector=none` on ready |
| `ItemQuestion` | from `retro-learning-room.md` | `revealed=false` | no feedback strip in the encounter |
| `GoalCard` | restyled | `label`, `icon`, `selected`; emits `select` | a `RetroPanel` tile with `role=radio` |
| `RankPlate` | new (kit addition) | `level` | VT323 34 on a `torch` outer line; reused by the roadmap header |
| `CalibrationRow` | new (kit addition, flagged) | `disabled`; emits `calibrate('easier'|'ok'|'harder')` | three secondary buttons; hidden without the endpoint |
**Kit additions:** `CompanionSprite` gains `seed` crack frames (2) and the `hatch` reaction (= `levelup` frames from seed to sprout); `RankPlate`; `CalibrationRow`; `RetroPanel frame=input` for text/time fields (2-px `line-lit` inset box, 48 px).

## 7. Copy
Title: "Học 30 phút" · "Mỗi ngày một phòng. Một người bạn." · "Đăng nhập bằng Google" · "Đang đăng nhập…" · caption "Bằng cách tiếp tục, cậu cho phép ứng dụng đọc lịch và nhiệm vụ Google để lên lịch học." Errors as shell §2.1.
Goal: "Chọn hành trình" · "IELTS 7.0" · "Business English" · "Đặt tên cho bạn đồng hành (không bắt buộc)" · placeholder "Mầm Non" · "Tên dài 1–30 ký tự, chỉ gồm chữ, số và dấu cách." · "Giờ nhắc học" · "Chọn một giờ nhắc học để tiếp tục." · "Bắt đầu trận đầu tiên".
Encounter: "Lượt {n} / 10" · "Chọn" · "Kết thúc" · companion (turn 1, once) "Tớ đang xem cậu đánh… cứ chọn theo cảm giác." · empty "Chưa có trận đầu tiên. Quay lại sau."
Waiting: "Tớ đang vẽ bản đồ cho cậu… (khoảng một phút)" · errors as shell; "Chưa vẽ lại được. Giữ hành trình hiện tại?" · "Giữ" · "Thử lại".
Ready: "Cấp {level}" · "Tớ nở rồi! Cậu ở cấp {level} — hành trình {goal}." · "Ngày 1" · "Ngày 1 trông thế nào?" · "Quá dễ" · "Vừa sức" · "Quá khó" · "Vào trại →". Offline toast: "Cần mạng để bắt đầu trận đầu tiên."

## 8. Self-critique
- Traded away: one-screen onboarding. Five steps is longer, but each has one action and the encounter is the only long one.
- The "encounter" metaphor promises a fight the quiz does not deliver (no hit/miss). The seed cracking is the substitute; if it reads as hollow in review, drop the cracks and keep only the hatch.
- Calibration depends on an endpoint that does not exist; shipping the row behind a flag risks dead UI in the tree — the plan must gate it on the backend plan, not on a runtime flag left on.
- Executor traps: never show the quiz's `answer` (there is none) or fake feedback; keep answers on every error path; the ready step must fetch `quests/daily` before rendering the tiles; the hatch plays once (store a `hatched` flag for the session).
- Review checks: OAuth `state` mismatch still refused; time input still `HH:MM:SS` on the wire; `plant_name` default not sent as a literal.

## Acceptance
1. `/login` shows the seed sprite, title and one Google button; after the redirect the button becomes a loading state and the code is exchanged once.
2. The goal step enables "Bắt đầu trận đầu tiên" only with a goal and a valid time; the name field is optional and validated per `plant-name.md`.
3. The encounter shows one question per turn with "Lượt n / 10", no correctness feedback, and posts all ten answers at "Kết thúc".
4. While the roadmap is generated the waiting panel is visible with the companion line and an indeterminate bar; no blank screen for the whole wait.
5. The ready step shows the level as a rank plate, the hatch reaction once, and the three real day-1 task titles from `quests/daily`.
6. The calibration row appears only when the regenerate endpoint is available, disables itself after one use, and returns to waiting → ready.
7. Assessment and regenerate errors keep the learner's answers and offer a retry.
8. Under reduced motion the seed and the hatch are static frame swaps.
