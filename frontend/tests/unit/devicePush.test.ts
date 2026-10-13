import { afterEach, expect, it, vi } from 'vitest'
import { pushCapability, subscribeDevice } from '~/utils/devicePush'
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })
function browser(permission = 'default') {
 const requestPermission = vi.fn().mockResolvedValue(permission)
 vi.stubGlobal('Notification', { permission, requestPermission }); vi.stubGlobal('PushManager', vi.fn())
 const subscribe = vi.fn().mockResolvedValue({ toJSON: () => ({ endpoint: 'https://push.test/local', keys: { p256dh: 'public', auth: 'local' } }) })
 vi.stubGlobal('navigator', { serviceWorker: { ready: Promise.resolve({ pushManager: { getSubscription: vi.fn().mockResolvedValue(null), subscribe } }) } })
 return { requestPermission, subscribe }
}
it('capability is honest for unsupported, missing service/client key, and denied', () => {
 vi.stubGlobal('Notification', undefined); expect(pushCapability(true, 'abc')).toBe('unsupported')
 browser(); expect(pushCapability(false, 'abc')).toBe('no-key'); expect(pushCapability(true, '')).toBe('no-key')
 browser('denied'); expect(pushCapability(true, 'abc')).toBe('denied')
})
it('capability does not request permission', () => { const b = browser(); expect(pushCapability(true, 'abc')).toBe('off'); expect(b.requestPermission).not.toHaveBeenCalled() })
it('denial never subscribes', async () => { const b = browser('denied'); expect(await subscribeDevice('YWJj', () => true)).toBeNull(); expect(b.subscribe).not.toHaveBeenCalled() })
it('flattens successful browser subscription', async () => { browser('granted'); expect(await subscribeDevice('YWJj', () => true)).toEqual({ endpoint: 'https://push.test/local', p256dh: 'public', auth: 'local' }) })
it('an account switch during permission prevents subscription', async () => { const b = browser('granted'); expect(await subscribeDevice('YWJj', () => false)).toBeNull(); expect(b.subscribe).not.toHaveBeenCalled() })
it('an account switch while reading an existing subscription does not create one', async () => {
 const b = browser('granted')
 let finish!: (value: null) => void
 let current = true
 vi.stubGlobal('navigator', { serviceWorker: { ready: Promise.resolve({ pushManager: { getSubscription: () => new Promise<null>(resolve => { finish = resolve }), subscribe: b.subscribe } }) } })
 const pending = subscribeDevice('YWJj', () => current)
 await vi.waitFor(() => expect(finish).toBeTypeOf('function'))
 current = false
 finish(null)
 expect(await pending).toBeNull()
 expect(b.subscribe).not.toHaveBeenCalled()
})
it('worker readiness is bounded and failed subscription is recoverable', async () => {
 vi.useFakeTimers(); browser('granted'); vi.stubGlobal('navigator', { serviceWorker: { ready: new Promise(() => {}) } }); const p = subscribeDevice('YWJj', () => true); const check = expect(p).rejects.toThrow('worker_timeout'); await vi.advanceTimersByTimeAsync(5001); await check
})
