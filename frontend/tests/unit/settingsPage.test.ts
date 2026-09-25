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
