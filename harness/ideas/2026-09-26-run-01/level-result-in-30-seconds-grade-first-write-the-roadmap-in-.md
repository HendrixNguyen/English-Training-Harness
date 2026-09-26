---
type: feature
status: proposed
source: ideator
run: 2026-09-26-run-01
order: 1
---

## Why
The first thing a new learner does after signing in is wait. `POST /api/v1/onboarding/assessment` grades the quiz (up to 30 s) and then writes the whole 28-day roadmap (53–78 s on the OpenAI/DeepSeek fallbacks, 180 s deadline) **inside the same request**, and `pages/onboarding.vue` tells them "thường mất 1–2 phút. Đừng đóng trang." A spinner of one to two minutes before any value is the single worst moment in the product: mobile-onboarding benchmarks put time-to-first-core-action under 60 s, and a first-session delay reads as "the app is broken". The retro-onboarding design already fixes what the learner *sees* during the wait (companion line + indeterminate bar); this idea shortens how long they wait for something *real*. The grading result is ready at ~30 s and is already staged server-side in the quiz hash (`_level`, providertimeout plan) — the learner should get it then: their CEFR level, their plant sprouting, and a hub that says the map is being drawn, while the roadmap is generated in the background. They can close the tab and come back. It also makes the calibration regenerate (the selected "session a learner wants to finish" idea and retro-onboarding §calibration) bearable, because a ±1-level rewrite becomes a background job with the same "map being drawn" state instead of a second 60–90 s freeze.

## Expected output
User-visible:
- After "Kết thúc", the learner sees their level and the sprout within ~30 s (the grading call only), with the line "Tớ đang vẽ bản đồ cho cậu…" and a way to go to the hub. Nothing asks them to keep the page open.
- The hub (`/`) has a new state between "no roadmap" and "quests": *map being drawn* — the plant, the level, the companion line and an indeterminate bar; it polls until the roadmap exists and then shows day 1. Reloading, closing the PWA or coming back an hour later lands in the same state or, once done, in the quests.
- If generation fails (bad output after the retry, provider outage, deadline), the hub shows the failure copy from the shell strings with "Thử lại", which re-runs generation **without re-grading** (the staged level is reused) and without a second `ratelimit:ai` slot for grading.

Technical (backend `onboarding` + `quests`, frontend hub + onboarding page):
- `Assess` is split: grade → stage level (existing) → persist `users.cefr_current`, `target_goal`, `notification_time`, `timezone` and `Pet.Ensure` → answer **202** `{status: "generating", assessed_level, pet_state}` → start the roadmap step as a background job (goroutine in-process, same `airouter.TaskTimeout`, the existing one-transaction save that deactivates the previous roadmap and inserts 84 exercises). A job key in Redis (`roadmap:gen:{user_id}`, TTL = 2 × 180 s, key builder in `store/keys.go` — an addition to §4 like `pet:revive`) makes a second submit while one is running answer 202 again instead of double-spending the AI (the concurrent-submit inbox bug, cited below, gets its guard from this).
- `GET /api/v1/quests/daily` answers `404 {"error":"roadmap_generating"}` (new code, additive) while the job key exists and no active roadmap does; `no_active_roadmap` stays for the true empty case. A failed job leaves a short-lived `roadmap:gen:{user_id}` state `failed` so the hub can show the copy and offer retry, which calls `POST /api/v1/onboarding/assessment` again with the same body (already a no-op for grading thanks to the staged level).
- The existing synchronous 201 path stays behind the same handler for a body that already has an active roadmap (200, unchanged).
- Backend spec §6.1 gains the 202 exchange and the new error code; 1st-thinking §5.1 steps 5–8 get a note that step 8 (init dashboard) now precedes the roadmap. CODEMAP `onboarding` and `quests` updated. Tests: the split with a scripted provider (grade ok, roadmap slow), the job-key guard on a concurrent submit, the failed-job retry that does not re-grade, and the frontend hub state machine (`generating` → `daily`) with a stub that flips on the second poll.
- Estimate: one working day (backend ~5 h, frontend ~3 h). Needs the designer role for the hub state and the onboarding result step; the retro-onboarding design already names both moments.

## Evidence
- Spec: 1st-thinking §5.1 (steps 4–8: submit test → route task → return roadmap → init dashboard); backend spec §6.1 (`POST /onboarding/assessment` 201 body); CODEMAP `onboarding` ("one assessment is bounded by 2 × 30 s + 2 × 180 s"; the staged `_level` reuse), `airouter` (`TaskTimeout` 180 s for `roadmap_generation`).
- Code: `frontend/pages/onboarding.vue:135` ("Đang chấm bài và soạn lộ trình 28 ngày — thường mất 1–2 phút. Đừng đóng trang."); `backend/internal/onboarding/service.go:84-99` (grade then roadmap in one `Assess`); `handler.go:59-61` (201/200 only).
- Design context: `harness/designs/retro-onboarding.md` (waiting step "Tớ đang vẽ bản đồ cho cậu… (khoảng một phút)", calibration row depends on a regenerate endpoint) — this idea gives that waiting state a place to live on the hub too.
- Related inbox bugs (not re-filed): `harness/ideas/_inbox/two-concurrent-assessment-submits-double-spend-the-ai-and-or.md` (the job key is the same guard); `harness/ideas/_inbox/assessment-request-has-no-overall-cap-malformed-output-retry.md` (a background job makes the cap a job property, not a request one).
- Research: mobile onboarding time-to-first-core-action under 60 s and "in first sessions, delays communicate product quality; a blank spinner feels broken" — https://www.digia.tech/post/mobile-app-onboarding-activation-retention/ ; 1 in 4 users drop after one session, often before the core product — https://www.saasfactor.co/blogs/why-users-drop-off-during-onboarding-and-how-to-fix-it
