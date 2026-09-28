---
plan: harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md
verdict: pass-with-bugs
bugs: [harness/ideas/_inbox/settings-saving-the-reminder-time-replaces-the-no-key-unsupp.md harness/ideas/_inbox/settings-reminder-switch-can-disagree-with-the-real-push-sub.md harness/ideas/_inbox/settings-sync-status-always-says-hom-nay-for-a-sync-persiste.md harness/ideas/_inbox/settings-re-consent-with-google-returns-the-learner-to-the-h.md harness/ideas/_inbox/settings-page-misses-the-design-hero-time-and-breaks-ui-kit-.md harness/ideas/_inbox/settings-reminders-requesting-renders-the-switch-on-a-stale-.md]
---
# Review — Settings screen: daily Web Push reminder and Google Calendar/Tasks sync wired to the shipped backend

**Plan:** `harness/plans/2026-09-24-settings-screen-wires-web-push-reminders-and-google-calendar.md`
**Branch/worktree:** `harness/2026-09-25-medium-settings-screen-wires-web-push-reminders-and-google-calendar` / `.worktrees/settings-screen-wires-web-push-reminders-and-google-calendar`
**Diff:** `git diff main...harness/2026-09-25-medium-settings-screen-wires-web-push-reminders-and-google-calendar --stat`

## Plan vs idea
Largely delivered. `/settings` lets a signed-in learner pick a reminder time, turn Web Push on/off (`POST /settings/notifications` with the flat `push_subscription`) and push the Calendar event + tasks (`POST /integrations/google/sync`), with `409 reauth_required` → Google consent and `502` → retry, and onboarding links to it. No backend change, no new dependency, as planned. The gaps are in the reminder card's state machine (below): one of them breaks the `no-key` / `unsupported` / `denied` states that production users will actually be in, so the idea's "never leave you guessing whether it worked" is not met there.

## Code vs plan
Reviewed at `origin/harness/2026-09-25-medium-settings-…` head `1f1922a`. CI on head: run 36108138415 `success` (2026-09-25). The branch is **133 commits behind `origin/main`** (kit v2 / retro components landed in between), so I verified the **merged** result that would ship: scratch worktree at `origin/main` + `git merge --no-ff` of the branch → code merges clean; **`harness/CODEMAP.md` conflicts** (shell bullet — resolved to main's side for the test run only).

On the merged tree, `frontend/`:
```
npm ci                 -> ok
npm run lint           -> clean
npm run typecheck      -> clean
npm run test:unit      -> Test Files 39 passed (39) · Tests 293 passed (293)
npm run build          -> ✨ Build complete!
```
Tasks: 1 `utils/push.ts` followed · 2 `stores/settings.ts` followed · 3 `useReminders.ts` followed (the `as BufferSource` cast is justified) · 4 `AppSwitch` + page followed (the synchronous `saveError`/`syncError` reset is a reasonable, documented deviation) · 5 onboarding link + `rememberTime` followed · 6 docs followed. Review Focus 1–5: 1 (denied on mount), 2 (subscribe rejects → no POST), 3 (rollback unsubscribe), 4 (re-send subscription on time save), 5 (409 not a sign-out) all hold in code and tests.

**Design walk (browser):** merged tree, `nuxi dev` on :3194, empty `NUXT_PUBLIC_VAPID_PUBLIC_KEY` (the production state), fake session in `localStorage['aelp.auth']`, API unreachable.
- `no-key`: the switch is hidden and the time field plus sync card render. Correct.
- Change the time to 07:30 → "Lưu giờ nhắc" appears. Correct.
- Tap it → "Không lưu được. Thử lại." Correct. **But the "Bật nhắc học" switch now appears** (`[role=switch]` present). Scratch Vitest confirms the successful-save variant too: `saveTime` turns `no-key`→`off`, `unsupported`→`off`, `denied`→`off`. Then `enable()` on the unsupported window throws `TypeError … reading 'requestPermission'` and sticks in `requesting`. **Blocker.**
- Also seen: the design's hero time (40/44) is absent, and the native time-picker icon is nearly invisible on the dark card.
- Not checked in the browser: the real permission prompt or subscription (no human to answer a browser permission prompt, and no real VAPID pair). Covered by `useReminders.test.ts` / `settingsPage.test.ts` fakes.

**Kit v2:** the page is v1-styled, which `retro-README.md` row 6 (`retro-settings.md`) explicitly defers. The new `AppSwitch` (rounded-full, blurred knob shadow, eased motion) and the `text-mute` status lines (~3:1 on the dark card) break kit v2 rules. Filed low, for plan 6 to absorb.

## Quality
- Correctness: the blocker above. The switch can also disagree with the real subscription: a failed save while on shows off, `init` trusts `localStorage`, and a save with the subscription gone still says on (medium). `requesting` renders the switch on, and a stale `saveError` can mislabel a failed enable (low).
- UX: the re-consent round trip lands on `/` rather than back on `/settings` (low). The persisted sync line always says "hôm nay" (low).
- Tests (the-validator pass, since the diff is >200 lines): several guards survive obvious mutants (`unsupported` hiding, `auth`-only / `Notification`-only support checks, dismissed prompt). No test drives `saveTime` from `no-key` / `unsupported` / `denied`, which is why the blocker passed.
- Boundaries: everything goes through `useApi()`, `localStorage` through `storageOrNull()`, and the `OAUTH_STATE_KEY` extraction is shared cleanly with `/login`. Tokens only, no hex.
- CODEMAP: the shell sentence is accurate, but it **conflicts with main** and must be re-applied when the branch is synced.

## Bugs filed
- `harness/ideas/_inbox/settings-saving-the-reminder-time-replaces-the-no-key-unsupp.md` — **high, blocker**: saving the time replaces `no-key` / `unsupported` / `denied` with `off`, showing a switch that can't work (iPhone: TypeError, stuck).
- `harness/ideas/_inbox/settings-reminder-switch-can-disagree-with-the-real-push-sub.md` — medium: switch vs real subscription (failed save while on; stale `on` on init; save with the subscription gone).
- `harness/ideas/_inbox/settings-sync-status-always-says-hom-nay-for-a-sync-persiste.md` — low.
- `harness/ideas/_inbox/settings-re-consent-with-google-returns-the-learner-to-the-h.md` — low.
- `harness/ideas/_inbox/settings-page-misses-the-design-hero-time-and-breaks-ui-kit-.md` — low: hero time, AppSwitch vs kit v2, contrast, picker icon.
- `harness/ideas/_inbox/settings-reminders-requesting-renders-the-switch-on-a-stale-.md` — low: `requesting` shown on, stale `saveError`, mutation-surviving tests.

## Verdict
pass-with-bugs — the feature is built and verified on the merged tree, but the branch is **held by a blocker**. The amend should sync the branch with `origin/main` (CODEMAP conflict), keep `saveTime` from overwriting the availability states, and guard `enable()`. Keep it out of today's daily PR.
