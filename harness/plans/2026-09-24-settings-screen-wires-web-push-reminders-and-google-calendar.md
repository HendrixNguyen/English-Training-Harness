---
idea: harness/ideas/2026-09-24-run-01/settings-screen-wires-web-push-reminders-and-google-calendar.md
status: approved
priority: medium
merged: false
design: harness/designs/settings.md
---
# Settings screen: daily Web Push reminder and Google Calendar/Tasks sync wired to the shipped backend — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Team:** Feature team — ticket **F1** of 2026-09-24. **Estimate:** 6 h. **Branch:** `harness/2026-09-24-medium-settings-screen-wires-web-push-reminders-and-google-calendar`.
**Approval:** medium feature, approved by the owner's 2026-09-24 planning instruction (override of the auto-approve rule; recorded in the idea's `## Evaluation`).

**Idea:** `harness/ideas/2026-09-24-run-01/settings-screen-wires-web-push-reminders-and-google-calendar.md`
**Design (spec for this plan):** `harness/designs/settings.md` — layout, every state, copy. Read it first; the copy strings below are copied from it.

**Goal:** `/settings` lets a signed-in learner choose a daily reminder time, turn Web Push reminders on and off, and push the 30-minute practice event plus the 28-day task list to Google — using only `POST /api/v1/settings/notifications` and `POST /api/v1/integrations/google/sync`, which are already on `main`.

**Architecture:** Frontend only. One pure util (`utils/push.ts`: VAPID key decoding, subscription flattening, support detection), one Pinia store (`stores/settings.ts`: the two API calls, error-code mapping, local persistence of the last saved time and last sync), one composable for browser glue (`composables/useReminders.ts`: permission, service-worker `pushManager`, state machine), one new UI primitive (`AppSwitch`), and the page. Everything goes through `useApi()`; no backend change; no new dependency.

**Tech stack:** Nuxt 3 (`ssr: false`), Vue 3, Pinia, Tailwind tokens from `tailwind.config.ts`, Vitest + `@vue/test-utils` on `happy-dom` (see `tests/unit/onboardingPage.test.ts` for the page-test pattern: mock `~/composables/useApi`, stub `navigateTo`).

**Decisions (from the evaluation):** no `GET` endpoint exists → prefill from `localStorage['aelp.settings']`, default `20:00`; "off" = `PushSubscription.unsubscribe()` client-side (server prunes on 404/410); `409 reauth_required` → the existing consent URL (`utils/googleAuth.ts`); one ghost link from onboarding's result step; **no Playwright spec** in this ticket (CI's `frontend` job runs lint/typecheck/unit/build, not e2e; the Vitest page test covers every state in the design — an e2e pass is a follow-up idea).

## Global Constraints

- Work in `.worktrees/<slug>`; never edit the main checkout (AGENTS.md). Run everything from `frontend/`.
- CI `frontend` job = `npm run lint && npm run typecheck && npm run test:unit && npm run build` — all four must pass before push. `typescript: { strict: true }`.
- Request body is the backend's **flat** shape: `{notification_time: "HH:MM:00", timezone: <IANA>, push_subscription?: {endpoint, p256dh, auth}}` (CODEMAP `notify`; not the browser's nested `keys`).
- UI copy is Vietnamese, exactly as in `harness/designs/settings.md` §4.
- Colours only via tokens (`growth`, `mute`, `alert`, `ink`, `paper`); no hex in components.
- `localStorage` access always through a `storageOrNull()` guard (pattern in `stores/pet.ts`).

## Review Focus

1. `Notification.permission === 'denied'` before the user touches anything → the page opens in the `denied` state with the switch disabled (not `off`). Task 3 test "denied on mount".
2. `pushManager.subscribe` rejects (e.g. `NotAllowedError` after a prompt dismissal) → state `off`, an `alert` line, **no** POST sent. Task 3.
3. `POST /settings/notifications` fails after `subscribe()` succeeded → unsubscribe the just-created subscription so the browser does not hold a subscription the server never learned about. Task 3 test "rolls back the subscription on a failed save".
4. Saving the time while reminders are `on` must re-send the existing subscription (so the server re-slots the queue) — Task 3 `saveTime()` reads `getSubscription()` first; test "re-sends the subscription with a new time".
5. `409 reauth_required` must not trigger the API client's 401 sign-out path (it is a 409, and `ApiError.code === 'reauth_required'`) — Task 2 maps it explicitly; Task 4 renders "Cho phép lại với Google".

---

## File structure

| Path | Change |
| --- | --- |
| `frontend/utils/push.ts` | **New**: `urlBase64ToUint8Array`, `flattenSubscription`, `pushSupport` |
| `frontend/tests/unit/pushClient.test.ts` | **New** |
| `frontend/stores/settings.ts` | **New**: `useSettingsStore` — `notificationTime`, `remindersOn`, `lastSync`, `hydrate`, `saveReminder`, `syncGoogle`, persistence at `aelp.settings` |
| `frontend/tests/unit/settingsStore.test.ts` | **New** |
| `frontend/composables/useReminders.ts` | **New**: browser glue + state machine |
| `frontend/tests/unit/useReminders.test.ts` | **New** |
| `frontend/components/ui/AppSwitch.vue` | **New** |
| `frontend/pages/settings.vue` | Rewrite (replaces the placeholder) |
| `frontend/tests/unit/settingsPage.test.ts` | **New** |
| `frontend/pages/onboarding.vue` | Result step: write the chosen time to the settings store; add the ghost link |
| `frontend/tests/unit/onboardingPage.test.ts` | One assertion for the link |
| `frontend/.env.example` | Comment: reminders switch hidden when `NUXT_PUBLIC_VAPID_PUBLIC_KEY` is empty |
| `harness/CODEMAP.md` | `shell` bullet: `/settings` sentence; stores list gains `settings.ts` |

---

## Tasks

### Task 1: `utils/push.ts` — the pure parts

**Files:**
- Create: `frontend/utils/push.ts`
- Test: `frontend/tests/unit/pushClient.test.ts`

**Interfaces (produces):**
```ts
export type PushSupport = 'ok' | 'unsupported' | 'no-key'
export function pushSupport(vapidPublicKey: string, w: Pick<Window, 'navigator'> & { PushManager?: unknown, Notification?: unknown } = window): PushSupport
export function urlBase64ToUint8Array(b64url: string): Uint8Array
export interface FlatSubscription { endpoint: string, p256dh: string, auth: string }
export function flattenSubscription(json: PushSubscriptionJSON): FlatSubscription | null
```

- [ ] **Step 1: Write the failing tests**

```ts
import { describe, expect, it } from 'vitest'
import { flattenSubscription, pushSupport, urlBase64ToUint8Array } from '~/utils/push'

describe('urlBase64ToUint8Array', () => {
  it('decodes base64url (no padding, -_ alphabet) to the raw bytes', () => {
    // "BAgQ" is base64 for [0x04, 0x08, 0x10]; base64url may drop '=' padding
    expect(Array.from(urlBase64ToUint8Array('BAgQ'))).toEqual([4, 8, 16])
    expect(Array.from(urlBase64ToUint8Array('-_8'))).toEqual([251, 255])
  })
})

describe('flattenSubscription', () => {
  it('maps the browser shape to the backend flat shape', () => {
    expect(flattenSubscription({ endpoint: 'https://push.example/abc', keys: { p256dh: 'BNc5', auth: 'aX8v' } }))
      .toEqual({ endpoint: 'https://push.example/abc', p256dh: 'BNc5', auth: 'aX8v' })
  })
  it('returns null when any field is missing', () => {
    expect(flattenSubscription({ endpoint: 'https://push.example/abc', keys: { p256dh: 'BNc5' } })).toBeNull()
    expect(flattenSubscription({ keys: { p256dh: 'a', auth: 'b' } })).toBeNull()
  })
})

describe('pushSupport', () => {
  const full = { navigator: { serviceWorker: {} } as Navigator, PushManager: function () {}, Notification: function () {} }
  it('is ok when serviceWorker, PushManager and Notification exist and a key is configured', () => {
    expect(pushSupport('BKey', full)).toBe('ok')
  })
  it('is no-key when the VAPID public key is empty, whatever the browser', () => {
    expect(pushSupport('', full)).toBe('no-key')
  })
  it('is unsupported without PushManager (iOS Safari outside an installed PWA) or without serviceWorker', () => {
    expect(pushSupport('BKey', { ...full, PushManager: undefined })).toBe('unsupported')
    expect(pushSupport('BKey', { ...full, navigator: {} as Navigator })).toBe('unsupported')
  })
})
```

- [ ] **Step 2: Run** — `npx vitest run tests/unit/pushClient.test.ts` → FAIL (module not found).

- [ ] **Step 3: Implement** `frontend/utils/push.ts`:

```ts
/** Client-side Web Push helpers (the service worker's receive side lives in service-worker/push.ts). */

export type PushSupport = 'ok' | 'unsupported' | 'no-key'

/** no-key hides the reminders switch (design §4.1); unsupported explains "Add to Home Screen". */
export function pushSupport(
  vapidPublicKey: string,
  w: Pick<Window, 'navigator'> & { PushManager?: unknown, Notification?: unknown } = window,
): PushSupport {
  if (!vapidPublicKey) return 'no-key'
  if (!('serviceWorker' in w.navigator) || !w.PushManager || !w.Notification) return 'unsupported'
  return 'ok'
}

/** applicationServerKey wants raw bytes; VAPID keys are published base64url. */
export function urlBase64ToUint8Array(b64url: string): Uint8Array {
  const padded = b64url + '='.repeat((4 - (b64url.length % 4)) % 4)
  const b64 = padded.replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(b64)
  return Uint8Array.from(raw, c => c.charCodeAt(0))
}

/** Backend spec §6.4 push_subscription — flat keys, not the browser's nested `keys`. */
export interface FlatSubscription { endpoint: string, p256dh: string, auth: string }

export function flattenSubscription(json: PushSubscriptionJSON): FlatSubscription | null {
  const endpoint = json.endpoint
  const p256dh = json.keys?.p256dh
  const auth = json.keys?.auth
  if (!endpoint || !p256dh || !auth) return null
  return { endpoint, p256dh, auth }
}
```

- [ ] **Step 4: Run** → PASS. `npm run lint` clean. **Step 5: Commit** — `git add utils/push.ts tests/unit/pushClient.test.ts && git commit -m "frontend: push helpers — VAPID key bytes, flat subscription, support detection"`.

### Task 2: `stores/settings.ts` — the two API calls and local memory

**Files:**
- Create: `frontend/stores/settings.ts`
- Test: `frontend/tests/unit/settingsStore.test.ts`

**Interfaces (produces):**
```ts
export const SETTINGS_STORAGE_KEY = 'aelp.settings'
export interface SettingsResponse { status: 'updated', notification_time: string, next_reminder_at?: string }
export interface SyncResponse { status: 'synced', calendar_event_id: string, tasks_created_count: number }
export type SyncError = 'reauth_required' | 'google_unavailable' | 'other'
// state: notificationTime: string ('HH:MM', default '20:00'), remindersOn: boolean,
//        lastSync: { at: string, tasksCreatedCount: number } | null, saving: boolean, syncing: boolean,
//        saveError: 'invalid' | 'other' | null, syncError: SyncError | null
// actions: hydrate(), rememberTime(time), saveReminder(time, sub: FlatSubscription | null): Promise<boolean>,
//          setRemindersOn(on), syncGoogle(): Promise<SyncResponse | null>
```

- [ ] **Step 1: Write the failing tests**

```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { SETTINGS_STORAGE_KEY, useSettingsStore } = await import('~/stores/settings')
const sub = { endpoint: 'https://push.example/abc', p256dh: 'BNc5', auth: 'aX8v' }

describe('useSettingsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.post.mockReset()
  })

  it('defaults to 20:00, reminders off, no sync', () => {
    const s = useSettingsStore()
    s.hydrate()
    expect(s.notificationTime).toBe('20:00')
    expect(s.remindersOn).toBe(false)
    expect(s.lastSync).toBeNull()
  })

  it('saveReminder posts the §6.4 flat body with seconds and the IANA timezone, then persists', async () => {
    api.post.mockResolvedValue({ status: 'updated', notification_time: '19:30:00' })
    const s = useSettingsStore()
    const ok = await s.saveReminder('19:30', sub)
    expect(ok).toBe(true)
    expect(api.post).toHaveBeenCalledWith('/api/v1/settings/notifications', {
      notification_time: '19:30:00',
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      push_subscription: sub,
    })
    expect(s.notificationTime).toBe('19:30')
    expect(JSON.parse(localStorage.getItem(SETTINGS_STORAGE_KEY)!).notificationTime).toBe('19:30')
  })

  it('saveReminder without a subscription omits push_subscription', async () => {
    api.post.mockResolvedValue({ status: 'updated', notification_time: '07:00:00' })
    await useSettingsStore().saveReminder('07:00', null)
    expect(api.post.mock.calls[0][1]).not.toHaveProperty('push_subscription')
  })

  it('maps 400 to invalid and anything else to other; the time is not changed on failure', async () => {
    const s = useSettingsStore()
    api.post.mockRejectedValueOnce(new ApiError(400, 'invalid_request'))
    expect(await s.saveReminder('25:00', null)).toBe(false)
    expect(s.saveError).toBe('invalid')
    expect(s.notificationTime).toBe('20:00')
    api.post.mockRejectedValueOnce(new TypeError('fetch failed'))
    expect(await s.saveReminder('08:00', null)).toBe(false)
    expect(s.saveError).toBe('other')
  })

  it('syncGoogle records the count and time on success', async () => {
    api.post.mockResolvedValue({ status: 'synced', calendar_event_id: 'evt', tasks_created_count: 28 })
    const s = useSettingsStore()
    const res = await s.syncGoogle()
    expect(api.post).toHaveBeenCalledWith('/api/v1/integrations/google/sync')
    expect(res?.tasks_created_count).toBe(28)
    expect(s.lastSync?.tasksCreatedCount).toBe(28)
    expect(JSON.parse(localStorage.getItem(SETTINGS_STORAGE_KEY)!).lastSync.tasksCreatedCount).toBe(28)
  })

  it('syncGoogle maps 409 → reauth_required, 502 → google_unavailable, else other', async () => {
    const s = useSettingsStore()
    api.post.mockRejectedValueOnce(new ApiError(409, 'reauth_required'))
    expect(await s.syncGoogle()).toBeNull()
    expect(s.syncError).toBe('reauth_required')
    api.post.mockRejectedValueOnce(new ApiError(502, 'google_unavailable'))
    await s.syncGoogle()
    expect(s.syncError).toBe('google_unavailable')
    api.post.mockRejectedValueOnce(new ApiError(500, 'internal_error'))
    await s.syncGoogle()
    expect(s.syncError).toBe('other')
  })

  it('hydrate reads a persisted time, flag and last sync; junk is discarded', () => {
    localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify({ notificationTime: '06:15', remindersOn: true, lastSync: { at: '2026-09-24T13:14:00.000Z', tasksCreatedCount: 28 } }))
    const s = useSettingsStore()
    s.hydrate()
    expect(s.notificationTime).toBe('06:15')
    expect(s.remindersOn).toBe(true)
    expect(s.lastSync?.tasksCreatedCount).toBe(28)
    localStorage.setItem(SETTINGS_STORAGE_KEY, '{not json')
    setActivePinia(createPinia())
    const t = useSettingsStore()
    t.hydrate()
    expect(t.notificationTime).toBe('20:00')
    expect(localStorage.getItem(SETTINGS_STORAGE_KEY)).toBeNull()
  })
})
```

- [ ] **Step 2: Run** → FAIL (module not found).

- [ ] **Step 3: Implement** `frontend/stores/settings.ts`:

```ts
import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'
import type { FlatSubscription } from '~/utils/push'

/** Backend spec §6.4 POST /settings/notifications 200 body (next_reminder_at is additive). */
export interface SettingsResponse { status: 'updated', notification_time: string, next_reminder_at?: string }
/** Backend spec §6.4 POST /integrations/google/sync 200 body. */
export interface SyncResponse { status: 'synced', calendar_event_id: string, tasks_created_count: number }
export type SyncError = 'reauth_required' | 'google_unavailable' | 'other'

export const SETTINGS_STORAGE_KEY = 'aelp.settings'
export const DEFAULT_REMINDER_TIME = '20:00'

interface Persisted {
  notificationTime: string
  remindersOn: boolean
  lastSync: { at: string, tasksCreatedCount: number } | null
}

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

/**
 * No GET endpoint exists for settings, so this store is the client's memory of
 * what it last told the server (onboarding writes the time too). A second
 * device shows the default until it saves once — accepted in the design.
 */
export const useSettingsStore = defineStore('settings', {
  state: () => ({
    notificationTime: DEFAULT_REMINDER_TIME,
    remindersOn: false,
    lastSync: null as Persisted['lastSync'],
    saving: false,
    syncing: false,
    saveError: null as 'invalid' | 'other' | null,
    syncError: null as SyncError | null,
  }),
  actions: {
    hydrate() {
      const raw = storageOrNull()?.getItem(SETTINGS_STORAGE_KEY)
      if (!raw) return
      try {
        const p = JSON.parse(raw) as Partial<Persisted>
        if (typeof p.notificationTime === 'string') this.notificationTime = p.notificationTime
        if (typeof p.remindersOn === 'boolean') this.remindersOn = p.remindersOn
        if (p.lastSync && typeof p.lastSync.at === 'string' && typeof p.lastSync.tasksCreatedCount === 'number') this.lastSync = p.lastSync
      } catch {
        storageOrNull()?.removeItem(SETTINGS_STORAGE_KEY)
      }
    },
    persist() {
      const p: Persisted = { notificationTime: this.notificationTime, remindersOn: this.remindersOn, lastSync: this.lastSync }
      storageOrNull()?.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(p))
    },
    /** Onboarding calls this after a successful assessment so /settings prefills. */
    rememberTime(time: string) {
      this.notificationTime = time
      this.persist()
    },
    setRemindersOn(on: boolean) {
      this.remindersOn = on
      this.persist()
    },
    /** time is "HH:MM" from the native input; the server wants "HH:MM:SS". */
    async saveReminder(time: string, sub: FlatSubscription | null): Promise<boolean> {
      this.saving = true
      this.saveError = null
      try {
        const body: Record<string, unknown> = {
          notification_time: `${time}:00`,
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        }
        if (sub) body.push_subscription = sub
        await useApi().post<SettingsResponse>('/api/v1/settings/notifications', body)
        this.notificationTime = time
        this.persist()
        return true
      } catch (e) {
        this.saveError = e instanceof ApiError && e.code === 'invalid_request' ? 'invalid' : 'other'
        return false
      } finally {
        this.saving = false
      }
    },
    async syncGoogle(): Promise<SyncResponse | null> {
      this.syncing = true
      this.syncError = null
      try {
        const res = await useApi().post<SyncResponse>('/api/v1/integrations/google/sync')
        this.lastSync = { at: new Date().toISOString(), tasksCreatedCount: res.tasks_created_count }
        this.persist()
        return res
      } catch (e) {
        if (e instanceof ApiError && (e.code === 'reauth_required' || e.code === 'google_unavailable')) this.syncError = e.code
        else this.syncError = 'other'
        return null
      } finally {
        this.syncing = false
      }
    },
  },
})
```

- [ ] **Step 4: Run** → PASS; `npm run lint && npm run typecheck` clean. **Step 5: Commit** — `git add stores/settings.ts tests/unit/settingsStore.test.ts && git commit -m "frontend: settings store — reminder time, push subscription save, Google sync"`.

### Task 3: `composables/useReminders.ts` — permission, subscription, state machine

**Files:**
- Create: `frontend/composables/useReminders.ts`
- Test: `frontend/tests/unit/useReminders.test.ts`

**Interfaces (produces):**
```ts
export type ReminderState = 'off' | 'requesting' | 'on' | 'denied' | 'unsupported' | 'no-key' | 'error' | 'invalid'
export function useReminders(deps?: { vapidPublicKey: string, win?: typeof window }): {
  state: Ref<ReminderState>
  init(): void                          // sets unsupported / no-key / denied / on|off from store + permission
  enable(time: string): Promise<void>   // requestPermission → subscribe → store.saveReminder(time, sub) ; rollback on failure
  disable(): Promise<void>              // getSubscription()?.unsubscribe(); store.setRemindersOn(false)
  saveTime(time: string): Promise<void> // store.saveReminder(time, currentSubscriptionOrNull)
}
```

- [ ] **Step 1: Write the failing tests** (a fake `serviceWorker.ready` registration with a fake `pushManager`; `Notification` stubbed as a function with static `permission` and `requestPermission`):

```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { useReminders } = await import('~/composables/useReminders')
const { useSettingsStore } = await import('~/stores/settings')

const VAPID = 'BAgQ'
function fakeBrowser(opts: { permission?: NotificationPermission, existing?: boolean, subscribeError?: Error } = {}) {
  const unsubscribe = vi.fn().mockResolvedValue(true)
  const subscription = {
    unsubscribe,
    toJSON: () => ({ endpoint: 'https://push.example/abc', keys: { p256dh: 'BNc5', auth: 'aX8v' } }),
  }
  const subscribe = vi.fn().mockImplementation(() => opts.subscribeError ? Promise.reject(opts.subscribeError) : Promise.resolve(subscription))
  const getSubscription = vi.fn().mockResolvedValue(opts.existing ? subscription : null)
  const requestPermission = vi.fn().mockResolvedValue(opts.permission ?? 'granted')
  const Notification = Object.assign(function () {}, { permission: opts.permission ?? 'default', requestPermission })
  const win = {
    navigator: { serviceWorker: { ready: Promise.resolve({ pushManager: { subscribe, getSubscription } }) } },
    PushManager: function () {},
    Notification,
  } as unknown as typeof window
  return { win, subscribe, getSubscription, unsubscribe, requestPermission }
}

describe('useReminders', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.post.mockReset()
  })

  it('init reports no-key without a VAPID key and unsupported without PushManager', () => {
    const { win } = fakeBrowser()
    const a = useReminders({ vapidPublicKey: '', win })
    a.init()
    expect(a.state.value).toBe('no-key')
    const b = useReminders({ vapidPublicKey: VAPID, win: { ...win, PushManager: undefined } as unknown as typeof window })
    b.init()
    expect(b.state.value).toBe('unsupported')
  })

  it('init is denied on mount when the browser already blocks notifications', () => {
    const { win } = fakeBrowser({ permission: 'denied' })
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    expect(r.state.value).toBe('denied')
  })

  it('enable asks permission, subscribes with the VAPID key bytes, posts the flat subscription, and lands on', async () => {
    api.post.mockResolvedValue({ status: 'updated', notification_time: '20:00:00' })
    const { win, subscribe, requestPermission } = fakeBrowser()
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    await r.enable('20:00')
    expect(requestPermission).toHaveBeenCalled()
    const arg = subscribe.mock.calls[0][0] as PushSubscriptionOptionsInit
    expect(arg.userVisibleOnly).toBe(true)
    expect(Array.from(arg.applicationServerKey as Uint8Array)).toEqual([4, 8, 16])
    expect(api.post.mock.calls[0][1]).toMatchObject({ push_subscription: { endpoint: 'https://push.example/abc', p256dh: 'BNc5', auth: 'aX8v' } })
    expect(r.state.value).toBe('on')
    expect(useSettingsStore().remindersOn).toBe(true)
  })

  it('enable goes denied and never posts when permission is refused', async () => {
    const { win, subscribe } = fakeBrowser({ permission: 'denied' })
    win.Notification.permission = 'default' // the prompt is shown, then refused
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    await r.enable('20:00')
    expect(subscribe).not.toHaveBeenCalled()
    expect(api.post).not.toHaveBeenCalled()
    expect(r.state.value).toBe('denied')
  })

  it('enable goes error (still off) when subscribe rejects', async () => {
    const { win } = fakeBrowser({ subscribeError: new Error('NotAllowedError') })
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    await r.enable('20:00')
    expect(api.post).not.toHaveBeenCalled()
    expect(r.state.value).toBe('error')
    expect(useSettingsStore().remindersOn).toBe(false)
  })

  it('enable rolls back the subscription when the server save fails', async () => {
    api.post.mockRejectedValue(new ApiError(500, 'internal_error'))
    const { win, unsubscribe } = fakeBrowser()
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    await r.enable('20:00')
    expect(unsubscribe).toHaveBeenCalled()
    expect(r.state.value).toBe('error')
  })

  it('disable unsubscribes and turns the flag off without a request', async () => {
    const { win, unsubscribe } = fakeBrowser({ existing: true })
    useSettingsStore().setRemindersOn(true)
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    expect(r.state.value).toBe('on')
    await r.disable()
    expect(unsubscribe).toHaveBeenCalled()
    expect(api.post).not.toHaveBeenCalled()
    expect(r.state.value).toBe('off')
  })

  it('saveTime re-sends the existing subscription with the new time so the server re-slots', async () => {
    api.post.mockResolvedValue({ status: 'updated', notification_time: '06:30:00' })
    const { win } = fakeBrowser({ existing: true })
    useSettingsStore().setRemindersOn(true)
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    await r.saveTime('06:30')
    expect(api.post.mock.calls[0][1]).toMatchObject({ notification_time: '06:30:00', push_subscription: { endpoint: 'https://push.example/abc' } })
  })

  it('saveTime with reminders off posts the time alone and maps 400 to invalid', async () => {
    api.post.mockRejectedValue(new ApiError(400, 'invalid_request'))
    const { win } = fakeBrowser()
    const r = useReminders({ vapidPublicKey: VAPID, win })
    r.init()
    await r.saveTime('99:99')
    expect(api.post.mock.calls[0][1]).not.toHaveProperty('push_subscription')
    expect(r.state.value).toBe('invalid')
  })
})
```

- [ ] **Step 2: Run** → FAIL (module not found).

- [ ] **Step 3: Implement** `frontend/composables/useReminders.ts`:

```ts
import { ref, type Ref } from 'vue'
import { useSettingsStore } from '~/stores/settings'
import { flattenSubscription, pushSupport, urlBase64ToUint8Array, type FlatSubscription } from '~/utils/push'

export type ReminderState = 'off' | 'requesting' | 'on' | 'denied' | 'unsupported' | 'no-key' | 'error' | 'invalid'

interface Deps {
  vapidPublicKey: string
  win?: typeof window
}

/** Design harness/designs/settings.md §4.1 — the reminder card's state machine over the browser Push API. */
export function useReminders(deps?: Deps): {
  state: Ref<ReminderState>
  init(): void
  enable(time: string): Promise<void>
  disable(): Promise<void>
  saveTime(time: string): Promise<void>
} {
  const win = deps?.win ?? window
  const vapid = deps?.vapidPublicKey ?? ''
  const store = useSettingsStore()
  const state = ref<ReminderState>('off')

  async function pushManager(): Promise<PushManager> {
    const reg = await win.navigator.serviceWorker.ready
    return reg.pushManager
  }

  async function currentSubscription(): Promise<{ sub: PushSubscription, flat: FlatSubscription } | null> {
    if (pushSupport(vapid, win) !== 'ok') return null
    const sub = await (await pushManager()).getSubscription()
    const flat = sub ? flattenSubscription(sub.toJSON()) : null
    return sub && flat ? { sub, flat } : null
  }

  function init() {
    const support = pushSupport(vapid, win)
    if (support !== 'ok') {
      state.value = support
      return
    }
    if (win.Notification.permission === 'denied') {
      state.value = 'denied'
      return
    }
    state.value = store.remindersOn ? 'on' : 'off'
  }

  async function enable(time: string) {
    state.value = 'requesting'
    const permission = await win.Notification.requestPermission()
    if (permission !== 'granted') {
      state.value = permission === 'denied' ? 'denied' : 'off'
      return
    }
    let sub: PushSubscription
    try {
      sub = await (await pushManager()).subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapid),
      })
    } catch {
      state.value = 'error'
      return
    }
    const flat = flattenSubscription(sub.toJSON())
    if (!flat || !(await store.saveReminder(time, flat))) {
      // The server never learned about this subscription: do not leave the
      // browser holding one that will never be sent to.
      await sub.unsubscribe()
      state.value = store.saveError === 'invalid' ? 'invalid' : 'error'
      return
    }
    store.setRemindersOn(true)
    state.value = 'on'
  }

  async function disable() {
    const cur = await currentSubscription()
    if (cur) await cur.sub.unsubscribe()
    // No unsubscribe endpoint: the server prunes this endpoint on its next 404/410.
    store.setRemindersOn(false)
    state.value = 'off'
  }

  async function saveTime(time: string) {
    const cur = store.remindersOn ? await currentSubscription() : null
    const ok = await store.saveReminder(time, cur?.flat ?? null)
    if (!ok) {
      state.value = store.saveError === 'invalid' ? 'invalid' : 'error'
      return
    }
    state.value = store.remindersOn ? 'on' : 'off'
  }

  return { state, init, enable, disable, saveTime }
}
```

- [ ] **Step 4: Run** → PASS; lint + typecheck clean. **Step 5: Commit** — `git add composables/useReminders.ts tests/unit/useReminders.test.ts && git commit -m "frontend: useReminders — permission, push subscription and the reminder state machine"`.

### Task 4: `AppSwitch` and the `/settings` page

**Files:**
- Create: `frontend/components/ui/AppSwitch.vue`
- Rewrite: `frontend/pages/settings.vue`
- Test: `frontend/tests/unit/settingsPage.test.ts`

**Interfaces:**
- Consumes: `useReminders`, `useSettingsStore`, `googleAuthUrl`/`randomState` (`utils/googleAuth.ts`), `useRuntimeConfig().public.vapidPublicKey` / `.googleClientId`, `SpeechBubble`, `AppCard`, `AppButton`, `AppHeader`.
- `AppSwitch` props: `modelValue: boolean`, `disabled?: boolean`, `busy?: boolean`, `labelledby: string`; emits `update:modelValue`.

- [ ] **Step 1: Write the failing page test** (pattern from `onboardingPage.test.ts`; `useReminders` is mocked so the page test is about rendering each state, and the composable's own test covers the browser glue):

```ts
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import AppSwitch from '~/components/ui/AppSwitch.vue'
import SpeechBubble from '~/components/plant/SpeechBubble.vue'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const reminders = { state: ref<string>('off'), init: vi.fn(), enable: vi.fn(), disable: vi.fn(), saveTime: vi.fn() }
vi.mock('~/composables/useReminders', () => ({ useReminders: () => reminders }))
const navigateTo = vi.fn()
vi.stubGlobal('navigateTo', navigateTo)
vi.stubGlobal('useRuntimeConfig', () => ({ public: { vapidPublicKey: 'BKey', googleClientId: 'cid', apiBase: '' } }))

const { default: SettingsPage } = await import('~/pages/settings.vue')

function mountPage() {
  return mount(SettingsPage, {
    global: { components: { AppButton, AppCard, AppSwitch, SpeechBubble }, stubs: { AppHeader: true, NuxtLink: true } },
  })
}

describe('/settings', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.post.mockReset()
    reminders.state.value = 'off'
    reminders.enable.mockReset()
    reminders.disable.mockReset()
    reminders.saveTime.mockReset()
  })

  it('opens on the default time, off, with the plant saying when it will remind', () => {
    const w = mountPage()
    expect(w.text()).toContain('Mình sẽ nhắc bạn lúc 20:00')
    expect(w.text()).toContain('Đang tắt')
    expect((w.find('input[type="time"]').element as HTMLInputElement).value).toBe('20:00')
    expect(w.find('button[type="button"]:not([role])').exists()).toBe(true)
  })

  it('turning the switch on calls enable with the current time; the on state reads the time back', async () => {
    const w = mountPage()
    await w.find('[role="switch"]').trigger('click')
    expect(reminders.enable).toHaveBeenCalledWith('20:00')
    reminders.state.value = 'on'
    await flushPromises()
    expect(w.text()).toContain('Đang nhắc lúc 20:00 mỗi ngày')
  })

  it('denied disables the switch and says where to fix it', async () => {
    reminders.state.value = 'denied'
    const w = mountPage()
    await flushPromises()
    expect(w.find('[role="switch"]').attributes('aria-disabled')).toBe('true')
    expect(w.text()).toContain('Trình duyệt đang chặn thông báo')
  })

  it('unsupported explains Add to Home Screen; no-key hides the switch but keeps the time field', async () => {
    reminders.state.value = 'unsupported'
    let w = mountPage()
    await flushPromises()
    expect(w.text()).toContain('Thêm vào Màn hình chính')
    reminders.state.value = 'no-key'
    w = mountPage()
    await flushPromises()
    expect(w.find('[role="switch"]').exists()).toBe(false)
    expect(w.find('input[type="time"]').exists()).toBe(true)
  })

  it('changing the time shows Lưu giờ nhắc and saves through saveTime', async () => {
    const w = mountPage()
    expect(w.text()).not.toContain('Lưu giờ nhắc')
    await w.find('input[type="time"]').setValue('06:30')
    expect(w.text()).toContain('Mình sẽ nhắc bạn lúc 06:30')
    const save = w.findAll('button').find(b => b.text() === 'Lưu giờ nhắc')!
    await save.trigger('click')
    expect(reminders.saveTime).toHaveBeenCalledWith('06:30')
  })

  it('error and invalid states show the alert lines', async () => {
    reminders.state.value = 'error'
    let w = mountPage()
    await flushPromises()
    expect(w.text()).toContain('Không lưu được. Thử lại.')
    reminders.state.value = 'invalid'
    w = mountPage()
    await flushPromises()
    expect(w.text()).toContain('Giờ nhắc không hợp lệ.')
  })

  it('Google sync success shows the task count and switches the button to Đồng bộ lại', async () => {
    api.post.mockResolvedValue({ status: 'synced', calendar_event_id: 'evt', tasks_created_count: 28 })
    const w = mountPage()
    await w.findAll('button').find(b => b.text() === 'Đồng bộ với Google')!.trigger('click')
    await flushPromises()
    expect(api.post).toHaveBeenCalledWith('/api/v1/integrations/google/sync')
    expect(w.text()).toContain('Đã đồng bộ')
    expect(w.text()).toContain('28 nhiệm vụ')
    expect(w.text()).toContain('Đồng bộ lại')
  })

  it('409 offers re-consent through the Google auth URL; 502 and 500 offer retry with honest copy', async () => {
    const assign = vi.fn()
    vi.stubGlobal('location', { ...window.location, assign, origin: 'https://app.example' })
    api.post.mockRejectedValueOnce(new ApiError(409, 'reauth_required'))
    const w = mountPage()
    await w.findAll('button').find(b => b.text() === 'Đồng bộ với Google')!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Google cần bạn cho phép lại')
    await w.findAll('button').find(b => b.text() === 'Cho phép lại với Google')!.trigger('click')
    expect(String(assign.mock.calls[0][0])).toContain('accounts.google.com/o/oauth2/v2/auth')
    expect(String(assign.mock.calls[0][0])).toContain('prompt=consent')

    // re-mount for the 502/500 paths (the click above navigated away)
    api.post.mockRejectedValueOnce(new ApiError(502, 'google_unavailable'))
    const w2 = mountPage()
    await w2.findAll('button').find(b => b.text() === 'Đồng bộ với Google')!.trigger('click')
    await flushPromises()
    expect(w2.text()).toContain('Google chưa phản hồi')
    api.post.mockRejectedValueOnce(new ApiError(500, 'internal_error'))
    await w2.findAll('button').find(b => b.text() === 'Thử lại')!.trigger('click')
    await flushPromises()
    expect(w2.text()).toContain('Không đồng bộ được. Thử lại.')
  })
})
```

(`pages/login.vue` keeps the OAuth `state` at `sessionStorage['aelp.oauth_state']` (`const STATE_KEY`) and validates it on return. Move that constant to `utils/googleAuth.ts` as `export const OAUTH_STATE_KEY = 'aelp.oauth_state'`, import it in `login.vue` and in the settings page, so the re-consent hand-off from `/settings` is validated exactly like a fresh sign-in.)

- [ ] **Step 2: Run** → FAIL (AppSwitch missing; page is the placeholder).

- [ ] **Step 3: Implement `AppSwitch.vue`**

```vue
<script setup lang="ts">
defineProps<{ modelValue: boolean, disabled?: boolean, busy?: boolean, labelledby: string }>()
const emit = defineEmits<{ 'update:modelValue': [boolean] }>()
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :aria-disabled="disabled || busy || undefined"
    :aria-busy="busy || undefined"
    :aria-labelledby="labelledby"
    class="relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors duration-200 motion-reduce:transition-none disabled:cursor-not-allowed"
    :class="modelValue ? 'bg-growth' : 'bg-mute/30'"
    :disabled="disabled || busy"
    @click="emit('update:modelValue', !modelValue)"
  >
    <span
      class="inline-block size-5 rounded-full bg-white shadow-sm transition-transform duration-200 motion-reduce:transition-none"
      :class="modelValue ? 'translate-x-[22px]' : 'translate-x-0.5'"
      aria-hidden="true"
    />
  </button>
</template>
```

- [ ] **Step 4: Implement `pages/settings.vue`** (copy from `harness/designs/settings.md` §4 verbatim):

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useReminders } from '~/composables/useReminders'
import { useSettingsStore } from '~/stores/settings'
import { googleAuthUrl, randomState } from '~/utils/googleAuth'

const config = useRuntimeConfig()
const settings = useSettingsStore()
const reminders = useReminders({ vapidPublicKey: config.public.vapidPublicKey })

const time = ref('20:00')
onMounted(() => {
  settings.hydrate()
  time.value = settings.notificationTime
  reminders.init()
})

const timeChanged = computed(() => time.value !== settings.notificationTime)
const switchOn = computed(() => reminders.state.value === 'on' || reminders.state.value === 'requesting')
const showSwitch = computed(() => !['unsupported', 'no-key'].includes(reminders.state.value))

/** Design §4.1 status lines, one per state. */
const reminderLine = computed(() => ({
  off: 'Đang tắt · Bạn sẽ không nhận thông báo.',
  requesting: 'Đang xin phép trình duyệt…',
  on: `Đang nhắc lúc ${settings.notificationTime} mỗi ngày.`,
  denied: 'Trình duyệt đang chặn thông báo. Mở cài đặt trang web, cho phép Thông báo, rồi thử lại.',
  unsupported: 'Thiết bị này chưa nhắc được qua trình duyệt. Trên iPhone: Chia sẻ → Thêm vào Màn hình chính, rồi mở lại.',
  'no-key': '',
  error: 'Không lưu được. Thử lại.',
  invalid: 'Giờ nhắc không hợp lệ.',
} as Record<string, string>)[reminders.state.value] ?? '')
const reminderIsAlert = computed(() => reminders.state.value === 'error' || reminders.state.value === 'invalid')

function toggle(on: boolean) {
  if (on) void reminders.enable(time.value)
  else void reminders.disable()
}
function saveTime() {
  void reminders.saveTime(time.value)
}

/** Design §4.2. */
const syncLine = computed(() => {
  if (settings.syncing) return 'Đang đồng bộ…'
  if (settings.syncError === 'reauth_required') return 'Google cần bạn cho phép lại để ghi lịch và nhiệm vụ.'
  if (settings.syncError === 'google_unavailable') return 'Google chưa phản hồi. Thử lại sau ít phút.'
  if (settings.syncError === 'other') return 'Không đồng bộ được. Thử lại.'
  if (settings.lastSync) {
    const at = new Date(settings.lastSync.at)
    return `Đã đồng bộ · ${settings.lastSync.tasksCreatedCount} nhiệm vụ · hôm nay ${at.getHours().toString().padStart(2, '0')}:${at.getMinutes().toString().padStart(2, '0')}`
  }
  return ''
})
const syncButton = computed(() => {
  if (settings.syncError === 'reauth_required') return { label: 'Cho phép lại với Google', variant: 'primary' as const }
  if (settings.syncError === 'google_unavailable') return { label: 'Thử lại', variant: 'primary' as const }
  if (settings.syncError === 'other') return { label: 'Thử lại', variant: 'danger' as const }
  if (settings.lastSync) return { label: 'Đồng bộ lại', variant: 'ghost' as const }
  return { label: 'Đồng bộ với Google', variant: 'primary' as const }
})
function onSync() {
  if (settings.syncError === 'reauth_required') {
    // Same hand-off as /login: prompt=consent re-grants calendar.events + tasks.
    const state = randomState()
    sessionStorage.setItem('aelp.oauth_state', state) // login.vue's STATE_KEY — extract it to utils/googleAuth.ts as OAUTH_STATE_KEY and import in both pages
    window.location.assign(googleAuthUrl(config.public.googleClientId, `${window.location.origin}/login`, state))
    return
  }
  void settings.syncGoogle()
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader />
    <h1 class="mt-2 font-display text-[28px] leading-8">Cài đặt</h1>

    <AppCard title="Nhắc học mỗi ngày" class="mt-4">
      <SpeechBubble :line="`Mình sẽ nhắc bạn lúc ${time}.`" />
      <label class="mt-3 block">
        <span class="sr-only">Giờ nhắc</span>
        <input
          v-model="time"
          type="time"
          class="h-12 w-full rounded-btn border border-ink/15 bg-transparent px-4 text-center font-display text-2xl tabular-nums dark:border-paper/15"
        >
      </label>
      <div v-if="showSwitch" class="mt-4 flex items-center justify-between border-t border-ink/10 pt-4 dark:border-paper/10">
        <span id="reminder-label" class="font-semibold">Bật nhắc học</span>
        <AppSwitch
          :model-value="switchOn"
          :disabled="reminders.state.value === 'denied'"
          :busy="reminders.state.value === 'requesting'"
          labelledby="reminder-label"
          @update:model-value="toggle"
        />
      </div>
      <p v-if="reminderLine" class="mt-2 text-sm" :class="reminderIsAlert ? 'text-alert' : 'text-mute'" role="status">
        {{ reminderLine }}
      </p>
      <AppButton v-if="timeChanged" class="mt-4" block :loading="settings.saving" @click="saveTime">
        Lưu giờ nhắc
      </AppButton>
    </AppCard>

    <AppCard title="Google Lịch & Nhiệm vụ" class="mt-4">
      <p class="text-mute">
        Tạo một sự kiện học 30 phút lặp mỗi ngày và một danh sách 28 nhiệm vụ.
      </p>
      <AppButton class="mt-4" block :variant="syncButton.variant" :loading="settings.syncing" @click="onSync">
        {{ syncButton.label }}
      </AppButton>
      <p v-if="syncLine" class="mt-2 text-sm" :class="settings.syncError && settings.syncError !== 'reauth_required' ? 'text-alert' : 'text-mute'" role="status">
        {{ syncLine }}
      </p>
    </AppCard>
  </main>
</template>
```

The test asserts only the URL; the callback on `/login` validates the `state` written above.

- [ ] **Step 5: Run** the page test → PASS; `npm run lint && npm run typecheck && npm run test:unit` clean.

- [ ] **Step 6: Commit** — `git add components/ui/AppSwitch.vue pages/settings.vue tests/unit/settingsPage.test.ts utils/googleAuth.ts && git commit -m "frontend: /settings — daily reminder switch and Google sync (design harness/designs/settings.md)"`.

### Task 5: Onboarding remembers the time and links to settings

**Files:**
- Modify: `frontend/pages/onboarding.vue` (after `result.value = await api.assess(...)`; result-step template)
- Test: `frontend/tests/unit/onboardingPage.test.ts`

- [ ] **Step 1: Add the failing assertion** to the existing test that reaches the result step (the one asserting "Trình độ của bạn"): after it, `expect(w.text()).toContain('Bật nhắc học và đồng bộ Google')` and `expect(JSON.parse(localStorage.getItem('aelp.settings')!).notificationTime).toBe('20:00')`. Add `localStorage.clear()` to that file's `beforeEach` if absent.

- [ ] **Step 2: Run** → FAIL.

- [ ] **Step 3: Implement** — import `useSettingsStore`; after `result.value = await api.assess({...})` add `useSettingsStore().rememberTime(time.value)`; in the result card, under the "Xem nhiệm vụ hôm nay" button add:

```vue
      <NuxtLink to="/settings" class="mt-3 block text-sm text-mute underline-offset-2 hover:underline">
        Bật nhắc học và đồng bộ Google →
      </NuxtLink>
```

- [ ] **Step 4: Run** → PASS. **Step 5: Commit** — `git add pages/onboarding.vue tests/unit/onboardingPage.test.ts && git commit -m "frontend: onboarding remembers the reminder time and points to /settings"`.

### Task 6: Docs

**Files:**
- Modify: `frontend/.env.example`, `harness/CODEMAP.md`

- [ ] **Step 1:** `.env.example`: after `NUXT_PUBLIC_VAPID_PUBLIC_KEY=` add a comment line `# Empty: /settings hides the reminders switch (time can still be saved); set to the server's VAPID_PUBLIC_KEY to enable Web Push.`
- [ ] **Step 2:** CODEMAP `shell` bullet: replace `` `/settings` (placeholder for notify/google) `` with `` `/settings` (design `harness/designs/settings.md`: reminder time + Web Push switch via `composables/useReminders.ts` → `POST /settings/notifications` with the flat `push_subscription`; "off" is a client-side `unsubscribe()` — no endpoint; Google sync button → `POST /integrations/google/sync`, `409 reauth_required` re-enters Google consent, `502` retries; no `GET` exists so the time is the client's last submission at `localStorage['aelp.settings']`, default 20:00) ``; in the stores list add `` `stores/settings.ts` (reminder time, reminders flag, last sync; persisted) ``.
- [ ] **Step 3: Commit** — `git add .env.example ../harness/CODEMAP.md && git commit -m "docs: settings screen in CODEMAP and .env.example"`.

## Verification

From `frontend/`:

```bash
npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build
```

Expected: all four green; unit output lists `pushClient`, `settingsStore`, `useReminders`, `settingsPage` suites passing. Manual (optional, needs the backend up with `VAPID_*` set): `npm run dev`, sign in, `/settings`, flip the switch → browser prompt → `Đang nhắc lúc 20:00 mỗi ngày.`; `psql "$DATABASE_URL" -c 'select count(*) from push_subscriptions'` → 1. Push the branch; `gh run list --branch <branch>` green on `frontend`.

## Notes and open questions

- **Production inertness:** until `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` are set on Railway and `NUXT_PUBLIC_VAPID_PUBLIC_KEY` matches, the switch is hidden by design. CORS plan (2026-09-24) must be merged for the calls to work from the PWA origin.
- **Follow-ups (not this ticket):** `GET /api/v1/settings` to prefill on a second device; a Playwright spec stubbing both routes; an unsubscribe endpoint if stale rows ever matter (today the server prunes on 404/410).
