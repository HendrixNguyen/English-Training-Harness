import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'
import { registerPrivateMemory } from '~/utils/session'
export interface Preferences { notification_time: string; timezone: string; next_reminder_at: string; push_configured: boolean }
export interface DeviceSubscription { endpoint: string; p256dh: string; auth: string }
function scope() { const auth = useAuthStore(); return `${auth.user?.id ?? ''}:${auth.sessionEpoch}` }
export const useSettingsStore = defineStore('settings', {
 state: () => ({ data: null as Preferences | null, owner: scope(), serial: 0, loading: false, saving: false, error: null as string | null, syncing: false, googleError: null as string | null, googleCount: null as number | null }),
 getters: { preferences: s => s.owner === scope() ? s.data : null },
 actions: {
  invalidate() { this.serial++; this.owner = scope(); this.data = null; this.loading = false; this.saving = false; this.error = null; this.syncing = false; this.googleError = null; this.googleCount = null },
  prepare() { registerPrivateMemory(this); if (this.owner !== scope()) this.invalidate(); return { owner: scope(), serial: this.serial } },
  current(ticket: { owner: string; serial: number }) { return ticket.owner === scope() && ticket.serial === this.serial },
  async load() {
   const ticket = this.prepare(); this.loading = true; this.error = null
   try { const data = await useApi().get<Preferences>('/api/v1/settings'); if (this.current(ticket)) this.data = data }
   catch (e) { if (this.current(ticket)) this.error = e instanceof ApiError ? e.code : 'network_error' }
   finally { if (this.current(ticket)) this.loading = false }
  },
  async save(time: string, subscription?: DeviceSubscription) {
   if (this.saving || !this.preferences) return false
   const ticket = this.prepare(); this.saving = true; this.error = null
   try {
    const result = await useApi().post<{ status: string; notification_time: string; next_reminder_at: string }>('/api/v1/settings/notifications', { notification_time: `${time}:00`, timezone: this.preferences!.timezone, ...(subscription ? { push_subscription: subscription } : {}) })
    if (!this.current(ticket)) return false
    this.data = { ...this.data!, notification_time: result.notification_time, next_reminder_at: result.next_reminder_at }; return true
   } catch (e) { if (this.current(ticket)) this.error = e instanceof ApiError ? e.code : 'network_error'; return false }
   finally { if (this.current(ticket)) this.saving = false }
  },
  async sync() {
   if (this.syncing) return
   const ticket = this.prepare(); this.syncing = true; this.googleError = null; this.googleCount = null
   try { const result = await useApi().post<{ status: string; tasks_created_count: number }>('/api/v1/integrations/google/sync'); if (this.current(ticket)) this.googleCount = result.tasks_created_count }
   catch (e) { if (this.current(ticket)) this.googleError = e instanceof ApiError ? e.code : 'network_error' }
   finally { if (this.current(ticket)) this.syncing = false }
  },
 },
})
