import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { useAuthStore } from '~/stores/auth'
import { useSettingsStore } from '~/stores/settings'
import { usePetStore } from '~/stores/pet'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import ConfirmSheet from '~/components/retro/ConfirmSheet.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { ApiError } from '~/utils/apiClient'
const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const { default: Settings } = await import('~/pages/settings.vue')
const prefs = { notification_time: '07:30:00', timezone: 'Asia/Ho_Chi_Minh', next_reminder_at: '2026-10-14T00:30:00Z', push_configured: true }
const config = { public: { vapidPublicKey: 'YWJj', googleClientId: 'test' } }
const navigate = vi.fn()
const wrappers: ReturnType<typeof mount>[] = []
function page() { const w = mount(Settings, { attachTo: document.body, global: { components: { RetroPanel, RetroButton, ConfirmSheet, StateBlock }, stubs: { NuxtLink: { template: '<a><slot /></a>' } }, mocks: { navigateTo: navigate } } }); wrappers.push(w); return w }
function button(w: ReturnType<typeof mount>, label: string) { const found = w.findAll('button').find(b => b.text().includes(label)); if (!found) throw new Error(`missing ${label}`); return found }
function browser(permission: string = 'default') {
 const requestPermission = vi.fn().mockResolvedValue(permission)
 vi.stubGlobal('Notification', { permission, requestPermission }); vi.stubGlobal('PushManager', vi.fn())
 const subscribe = vi.fn().mockResolvedValue({ toJSON: () => ({ endpoint: 'https://push.test', keys: { p256dh: 'public', auth: 'secret-test' } }) })
 vi.stubGlobal('navigator', { onLine: true, serviceWorker: { ready: Promise.resolve({ pushManager: { getSubscription: vi.fn().mockResolvedValue(null), subscribe } }) } })
 return { requestPermission, subscribe }
}
beforeEach(() => { setActivePinia(createPinia()); api.get.mockReset().mockResolvedValue({ ...prefs }); api.post.mockReset().mockResolvedValue({ status: 'updated', notification_time: '09:15:00', next_reminder_at: '2026-10-14T02:15:00Z' }); config.public.vapidPublicKey = 'YWJj'; vi.stubGlobal('useRuntimeConfig', () => config); vi.stubGlobal('navigateTo', navigate); navigate.mockReset(); browser() })
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.unstubAllGlobals() })
it('loads server time on reload without prompting permission or claiming Google sync', async () => {
 const b = browser(); let w = page(); await flushPromises(); expect(w.find('input').element.value).toBe('07:30'); expect(b.requestPermission).not.toHaveBeenCalled(); expect(w.text()).not.toContain('Đã đồng bộ'); w.unmount(); w = page(); await flushPromises(); expect(w.find('input').element.value).toBe('07:30')
})
it('reports an existing browser subscription on reload without claiming server delivery or writing', async () => {
 const b = browser('granted')
 const registration = await navigator.serviceWorker.ready
 const existing = { toJSON: () => ({ endpoint: 'https://push.test/old', keys: { p256dh: 'old', auth: 'old' } }) }
 const getSubscription = vi.spyOn(registration.pushManager, 'getSubscription').mockResolvedValue(existing as unknown as PushSubscription)
 let w = page(); await flushPromises()
 expect(w.text()).toContain('Trình duyệt đã có đăng ký thông báo')
 expect(w.text()).toContain('Chưa xác nhận máy chủ')
 expect(w.text()).not.toContain('Đã đăng ký nhắc trên thiết bị này')
 expect(b.requestPermission).not.toHaveBeenCalled()
 expect(api.post).not.toHaveBeenCalled()
 w.unmount(); w = page(); await flushPromises()
 expect(getSubscription).toHaveBeenCalledTimes(2)
 expect(w.text()).toContain('Trình duyệt đã có đăng ký thông báo')
 await button(w, 'Đăng ký lại với máy chủ').trigger('click')
 await flushPromises()
 expect(api.post.mock.calls.at(-1)?.[1]).toHaveProperty('push_subscription', { endpoint: 'https://push.test/old', p256dh: 'old', auth: 'old' })
 expect(w.text()).toContain('Đã đăng ký nhắc trên thiết bị này')
})
it('does not adopt a late prior-owner or unmounted subscription read', async () => {
 browser('granted')
 let finish!: (value: PushSubscription | null) => void
 const subscription = { toJSON: () => ({ endpoint: 'https://push.test/old', keys: { p256dh: 'old', auth: 'old' } }) } as unknown as PushSubscription
 const registration = await navigator.serviceWorker.ready
 vi.spyOn(registration.pushManager, 'getSubscription').mockImplementation(() => new Promise(resolve => { finish = resolve }))
 const w = page()
 await vi.waitFor(() => expect(finish).toBeTypeOf('function'))
 await useAuthStore().signOut()
 finish(subscription)
 await flushPromises()
 expect(w.text()).not.toContain('Trình duyệt đã có đăng ký thông báo')
 expect(api.post).not.toHaveBeenCalled()
 w.unmount()
 const next = page()
 await vi.waitFor(() => expect(finish).toBeTypeOf('function'))
 next.unmount()
 finish(subscription)
 await flushPromises()
 expect(api.post).not.toHaveBeenCalled()
})
it.each(['unsupported', 'no-key', 'denied'])('time saving preserves %s capability and does not invent fallback', async state => {
 if (state === 'unsupported') vi.stubGlobal('PushManager', undefined)
 if (state === 'no-key') config.public.vapidPublicKey = ''
 if (state === 'denied') browser('denied')
 const w = page(); await flushPromises(); const before = w.find('[role="status"]').text(); await w.find('input').setValue('09:15'); await w.find('form').trigger('submit'); await flushPromises()
 expect(w.find('[role="status"]').text()).toBe(before); expect(api.post.mock.calls[0]?.[1]).not.toHaveProperty('push_subscription'); expect(w.text()).not.toContain('Bật nhắc trên thiết bị'); expect(w.text()).not.toContain('nhắc qua lịch Google')
})
it('loads unknown/error retry, and shows invalid time save without false success', async () => {
 api.get.mockRejectedValueOnce(new Error()); const w = page(); await flushPromises(); expect(w.find('input').exists()).toBe(false); await button(w, 'Thử lại').trigger('click'); await flushPromises(); api.post.mockRejectedValueOnce(new ApiError(400, 'invalid_request')); await w.find('input').setValue('09:15'); await w.find('form').trigger('submit'); await flushPromises(); expect(w.text()).toContain('Giờ nhắc không hợp lệ'); expect(w.text()).toContain('Giờ đã lưu: 07:30')
})
it('successfully registers flattened device subscription only after explicit tap', async () => {
 const b = browser('granted'); const w = page(); await flushPromises(); expect(b.requestPermission).not.toHaveBeenCalled(); await button(w, 'Bật nhắc').trigger('click'); await flushPromises(); expect(api.post.mock.calls[0]?.[1]).toHaveProperty('push_subscription', { endpoint: 'https://push.test', p256dh: 'public', auth: 'secret-test' }); expect(w.text()).toContain('Đã đăng ký nhắc trên thiết bị này')
})
it('account switch during permission cannot submit old subscription/preferences under new owner', async () => {
 const b = browser('granted'); let finish!: (value: string) => void; b.requestPermission.mockImplementation(() => new Promise(resolve => { finish = resolve })); const w = page(); await flushPromises(); await button(w, 'Bật nhắc').trigger('click'); await useAuthStore().signOut(); finish('granted'); await flushPromises(); expect(api.post).not.toHaveBeenCalled(); expect(b.subscribe).not.toHaveBeenCalled(); expect(w.text()).not.toContain('Đã đăng ký')
})
it('Google 409 leads to existing safe login flow; 502 retries real sync and reports count', async () => {
 const w = page(); await flushPromises(); api.post.mockRejectedValueOnce(new ApiError(409, 'reauth_required')); await button(w, 'Đồng bộ với Google').trigger('click'); await flushPromises(); const origin = window.location.origin; await button(w, 'Cho phép lại').trigger('click'); expect(sessionStorage.getItem('aelp.oauth_state')).toMatch(/^[0-9a-f]{32}$/); expect(window.location.href).toContain('accounts.google.com/o/oauth2/v2/auth'); const consent = new URL(window.location.href); expect(consent.searchParams.get('prompt')).toBe('consent'); expect(consent.searchParams.get('state')).toBe(sessionStorage.getItem('aelp.oauth_state')); expect(consent.searchParams.get('redirect_uri')).toBe(`${origin}/login`)
 useSettingsStore().googleError = 'google_unavailable'; await flushPromises(); api.post.mockResolvedValueOnce({ status: 'synced', tasks_created_count: 12 }); await button(w, 'Thử lại').trigger('click'); await flushPromises(); expect(w.text()).toContain('Đã đồng bộ · 12 nhiệm vụ')
})
it('confirmation traps focus, Escape restores opener, and central signout clears preferences', async () => {
 usePetStore().status = { plant_name: 'Mầm Non', health_points: 80, stage: 'sprout', current_streak: 2, last_practiced_at: null }
 const w = page(); await flushPromises(); const opener = button(w, 'Rời quán'); (opener.element as HTMLElement).focus(); await opener.trigger('click'); await flushPromises(); const dialog = w.find('[role="dialog"]'); const controls = dialog.findAll('button'); expect(document.activeElement).toBe(controls[0]!.element)
 expect(dialog.text()).toContain('Mầm Non')
 await controls[0]!.trigger('keydown', { key: 'Tab', shiftKey: true }); expect(document.activeElement).toBe(controls[1]!.element); await controls[1]!.trigger('keydown', { key: 'Tab' }); expect(document.activeElement).toBe(controls[0]!.element)
 await dialog.trigger('keydown', { key: 'Escape' }); await flushPromises(); expect(w.find('[role="dialog"]').exists()).toBe(false); expect(document.activeElement).toBe(opener.element)
 await opener.trigger('click'); await flushPromises(); await w.find('[role="dialog"]').findAll('button')[1]!.trigger('click'); await flushPromises(); expect(useSettingsStore().preferences).toBeNull(); expect(navigate).toHaveBeenCalledWith('/login')
})
it('offline disables time/push/Google writes and explains reconnect', async () => {
 const w = page(); await flushPromises(); vi.stubGlobal('navigator', { onLine: false }); window.dispatchEvent(new Event('offline')); await flushPromises(); expect(button(w, 'Lưu giờ').attributes('disabled')).toBeDefined(); expect(button(w, 'Đồng bộ').attributes('disabled')).toBeDefined(); expect(w.text()).toContain('Cần mạng'); expect(api.post).not.toHaveBeenCalled()
})
it.each([false, true])('preserves dirty edit across entry refresh (save=%s)', async save => {
 useSettingsStore().data = { ...prefs }; let finish!: (value: unknown) => void
 api.get.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
 const w = page(); await w.find('input').setValue('09:15')
 if (save) { await w.find('form').trigger('submit'); await flushPromises() }
 finish({ ...prefs, notification_time: save ? '07:30:00' : '08:00:00' }); await flushPromises()
 expect(w.find('input').element.value).toBe('09:15'); expect(w.text()).toContain(`Giờ đã lưu: ${save ? '09:15' : '08:00'}`)
})
