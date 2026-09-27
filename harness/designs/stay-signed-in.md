# Design: Stay signed in — silent renewal and the "session expired" title screen

**Idea:** `harness/ideas/2026-09-25-run-01/stay-signed-in-sessions-renew-on-use-so-a-daily-learner-neve.md`
**Inherits:** `harness/designs/frontend-shell.md` §2.1 (`/login` today) · `harness/designs/retro-onboarding.md` "Title (`/login`)" (where the screen is going)
**Kit:** `harness/UI-KIT.md` v2 — but see §6: this plan lands on a branch cut from `origin/main`, where only the v1 components exist.
**Spec wireframe:** frontend spec §7.1 upper half. No departure: one line of copy is added above the Google button; nothing else on the screen moves.

## 0. Research
- **Learner's job:** open the app from the reminder push and be on the hub, practising, within one tap — and on the rare day they were away, understand in one glance why they are looking at the title screen instead.
- **The moment that earns the next minute:** there is none on this screen and that is the point — the feature's success is the learner *never seeing* `/login` while they keep their daily habit. When they do see it, the sentence removes the "did I get logged out? is something broken?" doubt so the Google tap feels routine, not suspicious.
- **What today does wrong:** `stores/auth.ts` `isAuthenticated` is a hard 24 h from sign-in; `middleware/auth.global.ts` drops the session and lands on `/login` with no explanation; `pages/login.vue` then says "Chào mừng bạn! 🌱" — a first-time greeting to a learner on day 12 who did everything right. The Google consent screen (`prompt=consent`) follows. The learner has no way to tell "expired" from "the app forgot me" or "my account was revoked".
- **Open questions, answered:**
  - *Does the retro title-screen doc cover `/login`?* Yes — `retro-onboarding.md` §3 "Title (`/login`)": seed sprite, VT323 title "HỌC 30 PHÚT", one body line, Google `RetroButton`, consent caption. It has no expired state; this doc adds one and the retro plan inherits it (§6).
  - *v1 now or wait for the kit?* **v1 now.** The renewal is invisible and the expired state is one sentence; both are learner impact today. The sentence is placed exactly where the retro doc will keep it, so the restyle changes the shell, not the copy or its position.
  - *Should the sentence also appear after a 401 (revoked / second device)?* No. `useApi`'s `onUnauthorized` keeps navigating to plain `/login`: the single-session bug is its own selected idea and a revoked session is not "24 h of inactivity", so saying so would be false.
  - *Does the SW interfere with renewal?* Yes, unless handled: `sw.ts` caches `GET /quests/daily` and `GET /pet/status` with `NetworkFirst` in `api-state`, headers included, so a cache-served response offline would carry a token that is older than the one in the store. §4 "Offline" rules it out.

## 1. The signature
The learner's session follows their habit. Every day they show up, the token quietly moves forward with them; only 24 hours of *absence* ends it, and when it does the title screen says so in one plain sentence above the button — the app owns the sign-out instead of leaving the learner to wonder.

## 2. Flow
- **Renewal (invisible):** any successful API response carrying `X-Session-Token` → `useAuthStore().renew(token, expiresIn)` → nothing on screen changes. No route change, no re-render of the current page beyond what the response itself caused, no toast, no flash, no `api-state` cache drop.
- **Expired session:** guarded route (`/`, `/learn/:id`, `/roadmap`, `/revive`, `/settings`) with a stored token whose `expiresAt ≤ now` → `auth.signOut()` (as today) → `navigateTo('/login?reason=expired', { replace: true })` → title screen with the expired notice → Google → `/` as today.
- **No session at all** (first visit, cleared storage) → `/login` with no `reason`, unchanged.
- **Deliberate sign-out** (`AppHeader` "Đăng xuất") → `/login` with no `reason`, unchanged.
- **401 from the API** (revoked or superseded) → `/login` with no `reason`, unchanged.
- `/login` opened directly with an expired token (the `to.path === '/login'` branch) drops the session as today and shows **no** notice — the learner did not ask for a protected page, so there is nothing to explain.

## 3. Layout (mobile-first, `max-w-md`)
Today's `pages/login.vue` column, unchanged, with one new block between the tagline and the button:

```
│        [PlantSvg sprout 96]             │ unchanged
│        Chào mừng bạn! 🌱                │ unchanged (h1)
│ Học 30 phút mỗi ngày, nuôi một cái cây. │ unchanged (text-mute)
│ ┌────────────────────────────────────┐  │ NEW — only when ?reason=expired
│ │ ⏳ Phiên đã hết hạn sau 24 giờ     │  │ AppCard (v1) → RetroPanel tone=plain (v2)
│ │    không hoạt động. Đăng nhập lại  │  │ text-sm, text-ink, left-aligned, glyph aria-hidden
│ │    để tiếp tục.                    │  │
│ └────────────────────────────────────┘  │
│ [    Đăng nhập bằng Google    ]         │ unchanged (AppButton block)
│ error card (unchanged, only on error)   │
│ caption: quyền lịch / nhiệm vụ Google   │ unchanged
```

Reads `route.query.reason` only. Desktop: same column, centred, as today.

## 4. States
- **Default (`reason` absent):** identical to today, byte for byte. The e2e `login.spec.ts` heading assertion keeps passing.
- **Expired (`reason=expired`):** the card above. It is part of the first paint, so it is **not** a live region — a screen reader meets it in reading order, after the tagline and before the button, which is the order a sighted learner reads it. No `aria-live`, no `role="alert"` (it is not an error and must not steal focus from the button).
- **Unknown `reason` (`reason=foo`, `reason=`, repeated param):** treated as absent — nothing rendered, nothing logged. The lookup is a pure helper (`utils/loginReason.ts`, `loginNotice(reason: unknown): string | null`) so the branch is unit-testable without mounting the page.
- **Expired + OAuth error:** the expired card stays above the button and the existing `alert` card appears below it, as today. Two cards, two registers (info above, error below) — never merged into one.
- **Loading (after the Google redirect):** the button becomes the spinner row as today; the expired card stays (the URL is `/login?code=…&state=…` by then, so in practice `reason` is gone and the card is not shown — acceptable, the learner has already acted).
- **Offline:** renewal cannot happen offline, and a cache-served response must not pretend it did. `sw.ts` adds a `cacheWillUpdate` plugin to the `api-state` `NetworkFirst` strategy that stores the response **without** `X-Session-Token` / `X-Session-Expires-In`. So offline the session simply keeps counting down from the last real renewal; the first online request renews it. `utils/session.ts` is unchanged — `clearApiCache()` is still what sign-in and sign-out call, and renewal never calls it.
- **Reduced motion:** nothing moves in either state.

## 5. Interactions and feedback
- **Silent renewal** — `utils/apiClient.ts` gains `onRenew?: (token: string, expiresIn: number) => void` in `ApiClientOptions` (keeps the util Pinia-free, same shape as `getToken` / `onUnauthorized`). After a response with `res.ok`, if `X-Session-Token` is present and differs from `getToken()`, call `onRenew(token, Number(X-Session-Expires-In))`. Error responses are ignored even if they carry the header. `useApi.ts` wires `onRenew` to `auth.renew`.
- **`stores/auth.ts` `renew(token, expiresIn, now = Date.now())`** — sets `accessToken`, `expiresAt = now + expiresIn * 1000`, rewrites `localStorage['aelp.auth']` with the same `user`; ignores the call when `expiresIn` is not a positive finite number or `user` is null; **never** calls `clearApiCache()`. Returns `void`.
- **One retry after a renewal** — on a `401`, if `getToken()` now differs from the token that was sent (a parallel response renewed meanwhile), re-send the same request once with the new token. A second `401` → `onUnauthorized()` as today. If no renewal happened, no retry — `onUnauthorized()` immediately, as today.
- **Expired redirect** — `middleware/auth.global.ts` guarded branch: `accessToken !== null && !isAuthenticated` → `navigateTo({ path: '/login', query: { reason: 'expired' } }, { replace: true })`; `accessToken === null` → `/login` with no query.
- **Google button** — unchanged; tapping it clears nothing extra. The `reason` param stays in the address bar until Google returns (the page already replaces the URL on `?code=`).

## 6. Components
| Component | New/existing | Props / events | Notes |
|---|---|---|---|
| `AppCard` | existing (v1) | default slot | The expired card today. **Decision:** build with v1 now; the retro title-screen plan swaps `AppCard` → `RetroPanel tone=plain` and the `⏳` text glyph → a 16-px pixel hourglass, keeping the copy, position and `data-testid="login-expired"`. |
| `AppButton` | existing (v1) | — | unchanged |
| `StateBlock` | existing | — | **not used**: its `empty`/`error` states carry `role="status"` / `text-mute` or `text-alert` semantics that are wrong for a neutral notice. |
| `utils/loginReason.ts` | new (pure) | `loginNotice(reason: unknown): string \| null` | `'expired'` → the copy in §7; anything else → `null`. |
| `apiClient` `onRenew` | new option | `(token, expiresIn) => void` | see §5 |
| `auth.renew` | new action | `(token, expiresIn, now?)` | see §5 |

Kit additions: none. Retro follow-up (not this plan): `retro-onboarding.md` "Title (`/login`)" gains the expired state as a `RetroPanel tone=plain` between the body line and the button, same copy — noted here so the retro plan's designer/executor carries it.

## 7. Copy
- Expired notice, `data-testid="login-expired"`: **"Phiên đã hết hạn sau 24 giờ không hoạt động. Đăng nhập lại để tiếp tục."** — glyph `⏳` before it, `aria-hidden="true"`. App voice, not the companion's (a session is the app's business, not the plant's).
- Everything else on `/login` is unchanged: "Chào mừng bạn! 🌱", "Học 30 phút mỗi ngày, nuôi một cái cây.", "Đăng nhập bằng Google", "Đang đăng nhập…", the three error strings, the consent caption.

## 8. Acceptance (the reviewer's list)
- [ ] A renewal (`X-Session-Token` on any `2xx`) changes `auth.accessToken`, `auth.expiresAt` and `localStorage['aelp.auth']` and **nothing visible**: no navigation, no reload, no toast, no re-fetch, no focus change. The `api-state` cache still holds its entries afterwards (`caches.delete` is not called).
- [ ] The header is ignored on non-`2xx` responses and when it equals the current token.
- [ ] A `401` that arrives after a renewal (token sent ≠ token now) is retried exactly once with the newest token; a `401` with no renewal in between signs out immediately, as today.
- [ ] A guarded route with an expired stored token redirects to `/login?reason=expired` (replace); a guarded route with no token redirects to `/login` with no query.
- [ ] `/login?reason=expired` shows the `⏳` card with the exact §7 sentence **between the tagline and the Google button**, not as a live region; `/login`, `/login?reason=foo` and `/login?reason=` render exactly as today.
- [ ] After "Đăng xuất" in `AppHeader`, and after a `401` sign-out, `/login` shows no notice.
- [ ] Offline, a cache-served `quests/daily` / `pet/status` response carries no `X-Session-Token` (the SW stripped it), so the stored token never moves backwards.
- [ ] Vitest — `tests/unit/authStore.test.ts`: `renew updates accessToken, expiresAt and aelp.auth without touching the api-state cache`; `renew ignores a non-positive expires_in and a missing user`. `tests/unit/apiClient.test.ts`: `calls onRenew with X-Session-Token and X-Session-Expires-In on a 2xx`; `ignores X-Session-Token on an error response and when it matches the current token`; `retries a 401 once with the newest token when a renewal landed since the request was sent`; `does not retry a 401 when no renewal happened`. `tests/unit/authMiddleware.test.ts`: `redirects an expired session to /login?reason=expired`; `redirects a missing session to /login without a reason`. `tests/unit/loginReason.test.ts`: `maps expired to the 24-hour sentence and everything else to null`. `tests/e2e/login.spec.ts`: `/login?reason=expired explains the 24-hour expiry before the Google button`; `/login without a reason shows no notice`.

## 9. Self-critique
- **Traded away:** a "Chào mừng trở lại!" heading for the expired state would be warmer, but it would fork the title copy the retro doc already fixes ("HỌC 30 PHÚT") and break the e2e heading match for one query param; the sentence does the work alone.
- **Traded away:** the companion could say it ("Tớ đợi cậu cả ngày rồi…"), but the kit gives the plant feelings and the app facts; a security-flavoured statement in the plant's voice would read as a guilt trip on the one day the learner was away — the opposite of what a habit product wants.
- **Where the executor could go wrong:** (1) calling `clearApiCache()` inside `renew` because `signIn` does — the acceptance list checks it; (2) adopting the header from a `NetworkFirst` cache hit — the `cacheWillUpdate` strip is the fix, not a client-side heuristic; (3) retrying every `401` — retry only when the token actually changed mid-flight; (4) using `StateBlock` for the notice; (5) putting `reason=expired` on the `/login`-direct branch or the `401` path.
- **What to check in review:** load the hub with a token within its half-life, watch the network tab for the header and confirm no re-render; set `expiresAt` in the past in `localStorage`, reload `/`, read the sentence; then "Đăng xuất" and confirm it is gone.
