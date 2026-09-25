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
    // the prompt is shown, then refused; `permission` is readonly in lib.dom's type, mutable on our fake
    ;(win.Notification as unknown as { permission: NotificationPermission }).permission = 'default'
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
