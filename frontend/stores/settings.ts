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
