import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, expect, it, vi } from 'vitest'
import { useAuthStore } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const { useSettingsStore } = await import('~/stores/settings')
const preferences = { notification_time: '07:30:00', timezone: 'Asia/Ho_Chi_Minh', next_reminder_at: '2026-10-14T00:30:00Z', push_configured: true }
beforeEach(() => { setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset() })
it('reads persisted preferences and saves wire time independently of push', async () => {
 api.get.mockResolvedValue(preferences); const s = useSettingsStore(); await s.load(); expect(s.preferences).toEqual(preferences)
 api.post.mockResolvedValue({ status: 'updated', notification_time: '08:45:00', next_reminder_at: '2026-10-14T01:45:00Z' })
 await s.save('08:45'); expect(api.post).toHaveBeenCalledWith('/api/v1/settings/notifications', { notification_time: '08:45:00', timezone: preferences.timezone }); expect(s.preferences?.notification_time).toBe('08:45:00')
})
it('read failure remains unknown and retry reads server', async () => {
 const s = useSettingsStore(); api.get.mockRejectedValue(new Error()); await s.load(); expect(s.preferences).toBeNull(); expect(s.error).toBe('network_error')
 api.get.mockResolvedValue(preferences); await s.load(); expect(s.preferences).toEqual(preferences)
})
it('failed time save keeps the saved preference', async () => {
 api.get.mockResolvedValue(preferences); const s = useSettingsStore(); await s.load(); api.post.mockRejectedValue(new ApiError(400, 'invalid_request')); await s.save('09:00'); expect(s.error).toBe('invalid_request'); expect(s.preferences).toEqual(preferences)
})
it.each(['reauth_required', 'google_unavailable', 'internal_error'])('Google exposes real %s and successful retry', async code => {
 const s = useSettingsStore(); api.post.mockRejectedValue(new ApiError(code === 'reauth_required' ? 409 : 502, code)); await s.sync(); expect(s.googleError).toBe(code); expect(s.googleCount).toBeNull()
 api.post.mockResolvedValue({ status: 'synced', tasks_created_count: 28 }); await s.sync(); expect(s.googleCount).toBe(28)
})
it('logout clears private preferences and rejects late read/sync', async () => {
 let finish!: (value: unknown) => void; api.get.mockImplementation(() => new Promise(resolve => { finish = resolve })); const s = useSettingsStore(); const pending = s.load(); await useAuthStore().signOut(); finish(preferences); await pending; expect(s.preferences).toBeNull()
 let synced!: (value: unknown) => void; api.post.mockImplementation(() => new Promise(resolve => { synced = resolve })); const write = s.sync(); await useAuthStore().signOut(); synced({ tasks_created_count: 28 }); await write; expect(s.googleCount).toBeNull()
})
it('keeps confirmed saves over earlier reads, and accepts later genuine refreshes', async () => {
 const s = useSettingsStore(); api.get.mockResolvedValue(preferences); await s.load()
 let old!: (value: unknown) => void; let during!: (value: unknown) => void; let saved!: (value: unknown) => void
 api.get.mockImplementationOnce(() => new Promise(resolve => { old = resolve })); const first = s.load()
 api.post.mockImplementationOnce(() => new Promise(resolve => { saved = resolve })); const write = s.save('09:15')
 api.get.mockImplementationOnce(() => new Promise(resolve => { during = resolve })); const second = s.load()
 saved({ notification_time: '09:15:00', next_reminder_at: preferences.next_reminder_at }); expect(await write).toBe(true)
 during(preferences); await second; old(preferences); await first
 expect(s.preferences?.notification_time).toBe('09:15:00'); expect(s.loading).toBe(false)
 api.get.mockResolvedValue({ ...preferences, notification_time: '10:00:00' }); await s.load(); expect(s.preferences?.notification_time).toBe('10:00:00')
})
it('accepts newest overlapping read and retains loading until it finishes', async () => {
 const s = useSettingsStore(); let old!: (value: unknown) => void; let newest!: (value: unknown) => void
 api.get.mockImplementationOnce(() => new Promise(resolve => { old = resolve })).mockImplementationOnce(() => new Promise(resolve => { newest = resolve }))
 const first = s.load(); const second = s.load(); old(preferences); await first; expect(s.loading).toBe(true); expect(s.preferences).toBeNull()
 newest(preferences); await second; expect(s.loading).toBe(false); expect(s.preferences).toEqual(preferences)
})
it('rejects late save after owner invalidation', async () => {
 const s = useSettingsStore(); api.get.mockResolvedValue(preferences); await s.load(); let finish!: (value: unknown) => void
 api.post.mockImplementation(() => new Promise(resolve => { finish = resolve })); const pending = s.save('09:15'); await useAuthStore().signOut()
 finish({ notification_time: '09:15:00', next_reminder_at: preferences.next_reminder_at }); expect(await pending).toBe(false); expect(s.preferences).toBeNull(); expect(s.saving).toBe(false)
})
