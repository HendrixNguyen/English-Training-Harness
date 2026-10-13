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
