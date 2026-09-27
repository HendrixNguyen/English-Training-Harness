# Design: Level result in 30 seconds — the map is drawn in the background (`/onboarding` result step, `/` drawing state)

**Idea:** `harness/ideas/2026-09-26-run-01/level-result-in-30-seconds-grade-first-write-the-roadmap-in-.md`
**Inherits:** `harness/designs/frontend-shell.md` §2.2–2.3 (guards, error register, NetworkFirst), `retro-onboarding.md` (waiting + ready steps — this doc splits them), `retro-hub.md` (every hub state below composes with its layout), `growth-moment.md` and `plant-name.md` (their hub changes are untouched: `start(pet.consumeDelta())` still runs before first paint; `plant_name` is still the speaker).
**Kit:** `harness/UI-KIT.md` v2 — tokens, components, motion, copy register.
**Spec wireframe:** frontend spec §7.1 (onboarding) and §7.2 (hub). Departures: §7.1 gains a result step that shows the level *before* the roadmap exists; §7.2 gains a state the wireframe has no row for — the quest list region while the roadmap is being written. `retro-onboarding.md` departure: day-1 tiles and the calibration row leave the onboarding "ready" step (they cannot exist at 30 s) and live on the hub; the "waiting" step now covers grading only.

## 0. Research
- **Learner's job:** finish the placement, learn their level, and reach the hub without being told to wait or to keep a tab open.
- **The moment that earns the next minute:** the hatch at ~30 s — the seed becomes a sprout and the rank plate says "Cấp B1" — followed by the companion *doing something visible for them* on the hub (drawing the map) rather than a blank wait. When the map lands, the path is revealed under the companion the learner already met.
- **What today does wrong:** `pages/onboarding.vue` holds the learner on the last quiz question with a loading button and "thường mất 1–2 phút. Đừng đóng trang." for the full grade + roadmap round trip (30 s + 53–78 s on fallbacks, 180 s deadline). Closing the tab loses the answers; the hub has no state between `no_active_roadmap` and a roadmap, so a learner who does leave sees "Cậu chưa có hành trình" and is sent back to re-take the quiz.
- **Open questions, answered from spec and code:**
  1. *Where does the level come from after a reload?* `AuthUser.cefr_current` (`stores/auth.ts`, persisted). The backend persists `users.cefr_current` before the 202, but the stored user predates the assessment, so the onboarding page writes `assessed_level` into `auth.user.cefr_current` (through the store's persist path) on the 202. A fresh sign-in on another device gets it from the sign-in response.
  2. *How does "Thử lại" re-post without the answers?* The assessment request body is persisted on the 202 (`localStorage` `aelp.assessment` = `{request, assessed_level, submitted_at}`), cleared when `quests/daily` answers 200. The staged level lives 2 h server-side (`store.PlacementQuizTTL`); a retry inside that window is graded from the stage, not the AI. Without a stored body the button sends the learner back to the encounter.
  3. *How does the hub learn "failed"?* `ApiError` carries only `code`. **Assumption (plan decides the wire):** `GET /quests/daily` answers `404 {"error":"roadmap_generating","status":"generating"|"failed"}` and `utils/apiClient.ts` keeps the parsed body on `ApiError.body`. If the plan prefers a sibling code `roadmap_failed`, the store maps both to the same `roadmapJob` value; nothing in this doc changes.
  4. *Polling cadence?* Gemini roadmaps take ~20–40 s, fallbacks 53–78 s, deadline 180 s, one retry, job key TTL 360 s. Backoff by elapsed time since `submitted_at`: **every 5 s for the first 60 s, every 10 s until 240 s, every 30 s until 420 s**, then the job is treated as stale (§4 failed). Never while `document.visibilityState !== 'visible'`; an immediate poll on `visibilitychange` → visible, `online`, and on mount. Elapsed time survives reload because `submitted_at` is stored; with no record (other device) it starts at 0.
  5. *Calibration row (retro-onboarding, flagged)?* Out of scope here; when the regenerate endpoint lands it re-enters the hub drawing state below instead of a second onboarding wait. Recorded, not designed.

## 1. The signature
**The map is drawn where the path will be.** On the hub the three quest tiles already stand in their places, but under fog with no icons or titles, and a `DrawingBar` sweeps above them while the companion says "Tớ đang vẽ bản đồ cho cậu…". When the poll succeeds the fog lifts in one frame, the connector lights cell by cell down to tile 1 and the `▶` cursor lands on it — the learner watches the road appear under the companion they hatched a minute ago. Nothing spins; something is built.

## 2. Flow
`/onboarding` encounter → "Kết thúc" → **grading** (button loading ≤ 30 s, waiting panel line about grading) → 202 → **result** step (hatch, rank plate, line, `DrawingBar`) → "Vào trại" → `/` **drawing** state (poll) → 200 → **reveal** → the first-day hub (`retro-hub.md` first-time state) → `/learn/:id`.
Side paths: 201/200 from the assessment (roadmap already exists) → the same result step with a static "Bản đồ đã sẵn sàng." caption instead of the bar → hub with quests, no drawing state. Reload / PWA reopen / return after an hour → `/` → `quest.load()` → 200 (plain hub, no reveal), `roadmap_generating` (drawing state resumed), `status: failed` (failed state), `no_active_roadmap` (shell empty state, unchanged). `/onboarding` guard: a user whose `quest.load()` answers `roadmap_generating` is redirected to `/` like a user with a roadmap.

## 3. Layout (mobile-first, `max-w-md`; desktop: the same column centred)
**Onboarding — grading** (replaces retro-onboarding "waiting"): `RetroPanel speaker={plant_name}` with the seed sprite (2-frame wobble) and the line "Tớ đang xem lại các lượt của cậu…"; `DrawingBar` under it; no button. Reads nothing yet.

**Onboarding — result**
```
│ ┌ Mầm Non ─────────────────────────┐   │ RetroPanel speaker=pet_state.plant_name
│ │ [sprout 128, hatch once]         │   │ CompanionSprite stage=pet_state.stage health=pet_state.health_points
│ │          CẤP B1                  │   │ RankPlate level=assessed_level (VT323 34, torch outer line)
│ │ "Tớ nở rồi! Cậu ở cấp B1. Giờ    │   │ SpeechBox line (§7)
│ │  tớ đang vẽ bản đồ cho cậu…"     │   │
│ └──────────────────────────────────┘   │
│ BẢN ĐỒ NGÀY 1                          │ eyebrow VT323 16
│ [░░░░▓▓▓▓░░░░░░░░░░░░]                 │ DrawingBar (indeterminate); 201/200: caption "Bản đồ đã sẵn sàng."
│ Đóng ứng dụng cũng được — bản đồ vẫn   │ caption font-body 14 ink-1
│ được vẽ tiếp.                          │
│ [          VÀO TRẠI →          ]       │ bottom bar, RetroButton primary
```
Reads the 202 body: `assessed_level`, `pet_state.plant_name`, `pet_state.stage`, `pet_state.health_points`; `status` picks bar vs caption. Writes `auth.user.cefr_current`, `aelp.assessment`.

**Hub — drawing state** (regions 1–2 and 5 of `retro-hub.md` unchanged; regions 3–4 replaced)
```
│ [av] Hendrix        [🔥 x0]  [🛡][🛡]   │ 1 status bar as retro-hub
│ ┌ Mầm Non ─────────────────────────┐   │ 2 companion panel as retro-hub: GET /pet/status
│ │ [sprite 128]  HP 100/100          │   │   + level label under the HpBar: VT323 20 text-torch "Cấp B1"
│ │               Cấp B1              │   │     from auth.user.cefr_current (hidden when null)
│ │ "Tớ đang vẽ bản đồ cho cậu…       │   │   SpeechBox line (§7)
│ │  Đóng ứng dụng cũng được."        │   │
│ └──────────────────────────────────┘   │
│ BẢN ĐỒ NGÀY 1                          │ 3' eyebrow (replaces "Phòng hôm nay" + DayBar)
│ [░░░░▓▓▓▓░░░░░░░░░░░░]                 │    DrawingBar
│ ┌──────────────────────────────────┐   │ 4' RetroPanel fog: three placeholder rows —
│ │ [▒▒]  ▒▒▒▒▒▒▒▒▒▒▒▒               │   │    56-px ground-2 square + a ground-2 title bar,
│ │ [▒▒]  ▒▒▒▒▒▒▒▒▒                  │   │    connector line-dim, aria-hidden; caption
│ │ [▒▒]  ▒▒▒▒▒▒▒▒▒▒▒                │   │    "Ba nhiệm vụ đầu tiên sẽ hiện ở đây." (ink-1)
│ └──────────────────────────────────┘   │
│                                        │ 5 bottom bar empty (nothing to enter yet)
```
Reads `GET /quests/daily` → `ApiError.code === 'roadmap_generating'` + `body.status`; `auth.user.cefr_current`; `aelp.assessment.submitted_at` for the backoff.

**Hub — failed state**: regions 1–2 as above (line changes, §7); region 3'–4' replaced by one `RetroPanel tone=ember`: cross glyph + "Không tạo được hành trình. Thử lại." + `RetroButton primary` "Thử lại" (or "Làm lại trận đầu tiên" when `aelp.assessment` is absent); the fog panel stays under it so the page keeps its shape. Bottom bar empty.

## 4. States
- **Loading (hub):** as retro-hub — the fog placeholders *are* the loading tiles; until the first `quest.load()` settles the eyebrow is absent and no `DrawingBar` shows, so a normal roadmap never flashes the drawing copy.
- **Drawing:** §3. Poll per §0.4. Companion idle breath only; the `DrawingBar` sweep is this state's one motion.
- **Done → reveal:** on the first 200 after a `roadmap_generating`: fog panel → real path in one frame (`QuestNode` ×3 from `tasks`, `DayBar` `0/30`, eyebrow "Phòng hôm nay"); connector to tile 1 lights cell by cell (300 ms, `steps`); cursor `▶` on tile 1; sprite `react=hit`; line "Vẽ xong rồi! Phòng đầu tiên ở ngay đây, cậu."; bottom button "Vào nhiệm vụ 1 →". Plays once per mounted page; `aelp.assessment` cleared. A fresh load that gets 200 is the plain first-time hub (retro-hub line), no reveal.
- **Failed (`status: failed`, or 420 s of `generating`):** §3 failed panel. Stale wording differs (§7). "Thử lại" → `RetroButton loading` → 202 → drawing state with `submitted_at` reset; 201/200 → `quest.load()` → quests. Errors on the retry: `rate_limited`, `ai_*`, `invalid_request`, network → the panel text swaps to the matching string (§7), button restored, answers still stored.
- **Empty (`no_active_roadmap`):** unchanged from retro-hub. The 2-h stage has expired or nothing was ever submitted; the button still goes to `/onboarding`.
- **Error (other codes) while drawing:** stay in the drawing state; a network error shows one `RetroToast` "Đang ngoại tuyến — bản đồ vẫn được vẽ, tớ sẽ xem lại khi có mạng." and polling pauses until `online`. Never show the retro-hub "Không tải được nhiệm vụ" panel over a drawing state.
- **Offline (onboarding result):** "Vào trại" is a local navigation and works; the hub then follows the rule above. "Thử lại" offline → toast "Cần mạng để thử lại." and the button stays enabled.
- **First-time:** this whole flow is first-time; the level label in region 2 stays on the hub only while drawing/failed — the normal hub is retro-hub's.
- **Reduced motion:** hatch = single frame swap (seed → sprout); `DrawingBar` static (a fixed 8-cell `growth` block at cells 6–13, caption unchanged); reveal = final frame at once, connector lit, chips none; the line appears whole.

## 5. Interactions and feedback
| Control | Within 100 ms | Then | Saved |
|---|---|---|---|
| "Kết thúc" | button drops → loading dots; grading panel | `POST /onboarding/assessment` → 202 result step (201/200: result step, static caption) | server: level, pet; client: `auth.user.cefr_current`, `aelp.assessment` |
| "Vào trại →" | drop | `navigateTo('/', {replace: true})`; hub `quest.load()` on mount | — |
| (poll) | — | `quest.load()` on the §0.4 schedule; owned by `usePollRoadmap()`; cleared on unmount, paused when hidden | `quest.roadmapJob` |
| "Thử lại" | drop → loading | `POST /onboarding/assessment` with `aelp.assessment.request` — one request, disabled while in flight | `submitted_at` reset |
| "Làm lại trận đầu tiên" | drop | `/onboarding` (guard lets a `failed` user in) | — |
| Streak badge / avatar | as retro-hub | `/roadmap` (shows the shell empty state while drawing) / `/settings` | — |

Store: `quest.roadmapJob: 'none' | 'generating' | 'failed'` set by `load()` from the 404 body (`noRoadmap` stays for `no_active_roadmap`); `quest.load()` success sets `'none'`. `usePollRoadmap()` reads `roadmapJob`, `aelp.assessment.submitted_at`, `document.visibilityState` and the `online` event; exposes `elapsedSeconds`, `stale`.

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `RetroPanel`, `RetroButton`, `CompanionSprite`, `SpeechBox`, `HpBar`, `DayBar`, `QuestNode`, `RetroToast`, `StateBlock` | kit | as kit | `RetroPanel fog` (kit prop) wraps the placeholders; `QuestNode connector` lights on reveal |
| `RankPlate` | kit addition from `retro-onboarding.md` | `level` | onboarding result only; the hub uses a VT323 20 `text-torch` label |
| `DrawingBar` | **new (kit addition)** | `reduced`; `aria-label="Đang vẽ bản đồ"`, `role="progressbar"` without `aria-valuenow` (indeterminate) | The indeterminate bar retro-onboarding described but never named: an `HpBar` track (`cells=40`, 164 px) with an 8-cell `growth` block sweeping left→right, `steps(8)`, 1.6 s, looping; reduced/`prefers-reduced-motion`: the block fixed at cells 6–13. Reason: needed by both the onboarding grading/result steps and the hub; nothing in the kit is indeterminate |
| `usePollRoadmap` | **new composable** | `start()`, `stop()`, `elapsedSeconds`, `stale` | Schedule §0.4; `setTimeout` chain, cleared on unmount; listens to `visibilitychange` and `online`; module-free (per page instance) |
| `stores/quest.ts` | modify | `roadmapJob`, `load()` mapping | reads `ApiError.body?.status` |
| `utils/apiClient.ts` | modify | `ApiError.body?: Record<string, unknown>` | the parsed error JSON, in addition to `code` |
| `pages/onboarding.vue` | modify | grading + result steps; persists `aelp.assessment`; sets `auth.user.cefr_current` | keeps every retro-onboarding string not listed in §7 |
| `pages/index.vue` | modify | drawing / failed / reveal branches between `noRoadmap` and the quests | the growth-moment `start(pet.consumeDelta())` and the `plant_name` speaker stay exactly as on their branches |

**Kit additions:** `DrawingBar` (above). No new token, type size or colour.

## 7. Copy
Onboarding grading: "Tớ đang xem lại các lượt của cậu…" · button (loading) "Kết thúc".
Onboarding result: "Cấp {level}" · line "Tớ nở rồi! Cậu ở cấp {level}. Giờ tớ đang vẽ bản đồ cho cậu…" · eyebrow "Bản đồ ngày 1" · caption "Đóng ứng dụng cũng được — bản đồ vẫn được vẽ tiếp." · ready caption (201/200) "Bản đồ đã sẵn sàng." · "Vào trại →".
Assessment errors (answers kept, as today with the kit's pronoun): `rate_limited` "Cậu vừa gửi quá nhiều lần. Đợi một phút rồi thử lại." · `ai_*` "Máy chủ AI đang bận, chưa chấm được bài. Thử lại sau ít phút." · `invalid_request` "Máy chủ không nhận thông tin đã gửi. Kiểm tra lại mục tiêu, tên cây và giờ nhắc học rồi thử lại." · other "Không tạo được hành trình. Thử lại."
Hub drawing: level label "Cấp {level}" · line "Tớ đang vẽ bản đồ cho cậu… Đóng ứng dụng cũng được." · eyebrow "Bản đồ ngày 1" · fog caption "Ba nhiệm vụ đầu tiên sẽ hiện ở đây." · offline toast "Đang ngoại tuyến — bản đồ vẫn được vẽ, tớ sẽ xem lại khi có mạng."
Hub reveal: line "Vẽ xong rồi! Phòng đầu tiên ở ngay đây, cậu." · bottom "Vào nhiệm vụ 1 →".
Hub failed: line "Bản đồ chưa xong. Cậu bấm thử lại giúp tớ nhé." · panel "Không tạo được hành trình. Thử lại." · stale (420 s) panel "Vẽ bản đồ lâu hơn bình thường. Thử lại." · "Thử lại" · "Làm lại trận đầu tiên" · retry errors: the assessment strings above · offline "Cần mạng để thử lại."
`DrawingBar` `aria-label` "Đang vẽ bản đồ".

## 8. Self-critique
- Traded away: the result step no longer shows the three real day-1 tiles (retro-onboarding acceptance 5). The learner judges the map on the hub instead — that is where the tiles will live every other day, so the trade is a shorter first wait for one fewer preview.
- Persisting the assessment body in `localStorage` keeps ten quiz answers on the device until the roadmap exists. It is the only way "Thử lại" avoids a second quiz; the record is cleared on the first 200 and on sign-out (`auth.signOut()` already clears per-user state — the plan adds this key).
- The failed signal (§0.3) is an assumption on the backend plan; if it lands as a distinct code the store change is one line, but the executor must not ship a hub that treats *any* 404 as "failed".
- 420 s is a client guess at "stale" (2 × TTL + margin). A slow but live job past that point shows the failed panel; "Thử lại" then answers 202 again (the job key guard) and the drawing state resumes — annoying, not harmful.
- Executor traps: never flip to the retro-hub quest error panel on a network error while `roadmapJob === 'generating'`; the poll must not run in the background tab (battery, and the `ratelimit:ai` slot is not touched by GET but the API host is free-tier); `submitted_at` must come from the stored record, not `Date.now()` at mount, or the backoff restarts on every reload; the reveal must key off the *transition* (`generating` → 200 in one mounted page), not off `day_number === 1`; `auth.user.cefr_current` is written through the store so the persisted copy updates; keep the 201/200 path rendering the result step (the hub then has quests — no drawing state).
- Review checks: `grep -rn "Đừng đóng trang" frontend/` is empty; the drawing state has no `rounded-*` above 2 px, no easing; `DrawingBar` has no `aria-valuenow`; timers cleared on unmount (fake-timer test leaks none).

## Acceptance
1. With a stubbed assessment that answers 202 `{status: "generating", assessed_level: "B1", pet_state}` and a `quests/daily` stub that never completes, the result step shows the rank plate "Cấp B1", the sprout (hatch once) and the `DrawingBar` — and no string in `frontend/` tells the learner to keep the page open.
2. "Vào trại →" navigates to `/`; while `GET /quests/daily` answers `404 roadmap_generating` (`status: generating`) the hub shows the companion panel with "Cấp B1", the drawing line, the eyebrow "Bản đồ ngày 1", the `DrawingBar`, the fogged placeholders and no bottom button.
3. Polling (fake timers): `quests/daily` is requested at 5-s intervals for the first 60 s, 10-s until 240 s, 30-s after; no request fires while the document is hidden; one fires immediately on `visibilitychange` → visible and on `online`; at 420 s of `generating` the failed panel appears with the stale string.
4. A reload or PWA reopen during generation shows the same drawing state, the level from `auth.user.cefr_current`, and the backoff continues from the stored `submitted_at`; no assessment POST is sent on load. A fresh load after completion renders the plain first-time hub with no reveal.
5. When a poll returns 200 the fog is replaced by the three `QuestNode`s and the `DayBar` within one frame, the connector to tile 1 lights, the cursor lands on tile 1, the bottom button reads "Vào nhiệm vụ 1 →", the line is the reveal line, and `aelp.assessment` is removed — once per mounted page.
6. `status: failed` shows the ember panel "Không tạo được hành trình. Thử lại." with "Thử lại"; tapping it sends exactly one `POST /onboarding/assessment` whose body equals the stored request (`target_goal`, `notification_time`, `timezone`, `answers`, `plant_name` when present), disables while in flight, and on 202 returns to the drawing state. With no stored record the button reads "Làm lại trận đầu tiên" and opens `/onboarding`.
7. A network error during polling keeps the drawing state and shows exactly one offline toast; the quest error panel never appears while `roadmapJob === 'generating'`.
8. A 201/200 assessment (roadmap already active) renders the result step with the "Bản đồ đã sẵn sàng." caption and "Vào trại →" lands on a hub with quests; `no_active_roadmap` still renders the shell empty state.
9. Under `prefers-reduced-motion` (and `reduced`) no keyframe runs: the hatch is a frame swap, the `DrawingBar` block is static, the reveal is the final frame, and every state is distinguishable by text and glyph.
10. `npm run lint`, `npm run typecheck`, `npm run test:unit`, `npm run build` green; CI green on the pushed branch.
