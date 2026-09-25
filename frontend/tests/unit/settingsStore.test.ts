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
