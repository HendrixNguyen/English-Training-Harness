import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'
import { useAuthStore } from '~/stores/auth'
import { hasUsableContent } from '~/utils/content'
import { clampDuration } from '~/utils/progress'

export const TASK_ORDER = ['vocabulary', 'reading', 'practice'] as const
export type TaskType = (typeof TASK_ORDER)[number]

/** Backend spec §6.2 GET /quests/daily task. content_json is the roadmap task object (airouter). */
export interface QuestTask {
  id: string
  task_type: TaskType | string
  title: string
  duration_minutes: number
  is_completed: boolean
  content_json: unknown
}

export interface DailyQuests {
  date: string
  day_number: number
  total_minutes_required: number
  accumulated_seconds: number
  is_target_met: boolean
  tasks: QuestTask[]
}

/** Backend spec §6.2 POST /quests/progress 200 body. */
export interface ProgressResponse {
  daily_seconds_spent: number
  daily_minutes_spent: number
  is_target_met: boolean
  // omitted when the backend's pet read failed (CODEMAP quests); never 0-for-unknown
  pet_health?: number
  streak_count?: number
}

/** What `complete()` resolves: the §6.2 body plus whether this call newly met today's target (hub growth moment). */
export type CompleteResult = ProgressResponse & { targetMetChanged: boolean }

/** A per-task countdown anchored to the wall clock so reloads, background tabs and PWA suspension never lose minutes. */
export interface Timer {
  startedAt: number // epoch ms of the learner's first open of the task
  totalSeconds: number
  date: string // GET /quests/daily `date` the anchor belongs to; other days are discarded on load
}

export const TIMER_STORAGE_KEY = 'aelp.timers'

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

function readTimers(): Record<string, Timer> {
  try {
    const raw = storageOrNull()?.getItem(TIMER_STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Record<string, Partial<Timer>>
    const out: Record<string, Timer> = {}
    for (const [id, t] of Object.entries(parsed)) {
      if (typeof t.startedAt === 'number' && typeof t.totalSeconds === 'number' && typeof t.date === 'string') out[id] = { startedAt: t.startedAt, totalSeconds: t.totalSeconds, date: t.date }
    }
    return out
  } catch {
    return {}
  }
}

function writeTimers(timers: Record<string, Timer>) {
  try {
    storageOrNull()?.setItem(TIMER_STORAGE_KEY, JSON.stringify(timers))
  } catch {
    // storage denied or full: the in-memory anchor still works for this page lifetime
  }
}

function rank(type: string): number {
  const i = (TASK_ORDER as readonly string[]).indexOf(type)
  return i === -1 ? TASK_ORDER.length : i
}

const contentCacheKey = 'aelp.exercise-content'
const pendingHydration = new WeakMap<object, Map<string, Promise<QuestTask>>>()
const pendingDaily = new WeakMap<object, Promise<void>>()
function sessionScope(): string {
  const auth = useAuthStore()
  return `${auth.user?.id ?? ''}:${auth.accessToken ?? ''}`
}
function cachedContent(id: string): unknown {
  try {
    const cache = JSON.parse(storageOrNull()?.getItem(contentCacheKey) ?? 'null')
    if (cache?.owner === (useAuthStore().user?.id ?? null)) return cache.tasks?.[id]
  } catch { /* denied storage or an obsolete cache */ }
  return undefined
}
function cacheContent(id: string, content: unknown) {
  try {
    const owner = useAuthStore().user?.id ?? null
    const stored = JSON.parse(storageOrNull()?.getItem(contentCacheKey) ?? 'null')
    const tasks = stored?.owner === owner && typeof stored.tasks === 'object' && stored.tasks !== null ? stored.tasks : {}
    tasks[id] = content
    storageOrNull()?.setItem(contentCacheKey, JSON.stringify({ owner, tasks }))
  } catch { /* in-memory content remains usable */ }
}

export const useQuestStore = defineStore('quest', {
  state: () => ({
    daily: null as DailyQuests | null,
    accountScope: sessionScope(),
    noRoadmap: false,
    loading: false,
    error: null as string | null,
    timers: readTimers(),
    nowMs: Date.now(),
  }),
  getters: {
    sortedTasks: (s): QuestTask[] => (s.daily ? [...s.daily.tasks].sort((a, b) => rank(a.task_type) - rank(b.task_type)) : []),
    nextTaskId(): string | null {
      return this.sortedTasks.find(t => !t.is_completed)?.id ?? null
    },
    accumulatedSeconds: s => s.daily?.accumulated_seconds ?? 0,
    targetMet: s => s.daily?.is_target_met ?? false,
  },
  actions: {
    resetForAccount() {
      if (this.accountScope === sessionScope()) return
      pendingDaily.delete(this)
      this.accountScope = sessionScope()
      this.daily = null
      this.noRoadmap = false
      this.error = null
      this.loading = false
      this.timers = {}
      writeTimers(this.timers)
    },
    async load() {
      this.resetForAccount()
      const scope = sessionScope()
      const existing = pendingDaily.get(this)
      if (existing) return existing
      this.loading = true
      this.error = null
      const request = (async () => {
        try {
          const response = await useApi().get<DailyQuests>('/api/v1/quests/daily')
          if (scope !== sessionScope()) return
          for (const task of response.tasks) {
            const previous = this.taskById(task.id)
            if (previous?.is_completed) task.is_completed = true
            if (!hasUsableContent(task.content_json)) {
              const cached = previous && hasUsableContent(previous.content_json) ? previous.content_json : cachedContent(task.id)
              if (hasUsableContent(cached)) task.content_json = cached
            }
          }
          this.daily = response
          this.pruneTimers()
          this.noRoadmap = false
        } catch (e) {
          if (scope !== sessionScope()) return
          if (e instanceof ApiError && e.code === 'no_active_roadmap') {
            this.daily = null
            this.noRoadmap = true
          } else {
            this.error = e instanceof ApiError ? e.code : 'network_error'
          }
        } finally {
          if (scope === sessionScope()) this.loading = false
        }
      })()
      pendingDaily.set(this, request)
      try { await request } finally { if (pendingDaily.get(this) === request) pendingDaily.delete(this) }
    },
    /** Fetch only outlines; the caller can invalidate application after a route change. */
    async hydrateTask(id: string, current: () => boolean = () => true): Promise<QuestTask | null> {
      this.resetForAccount()
      const task = this.taskById(id)
      if (!task || task.is_completed || hasUsableContent(task.content_json)) return task
      const scope = sessionScope()
      const daily = this.daily
      let pending = pendingHydration.get(this)
      if (!pending) { pending = new Map(); pendingHydration.set(this, pending) }
      const key = `${scope}:${id}`
      let request = pending.get(key)
      if (!request) {
        request = useApi().get<QuestTask>(`/api/v1/quests/exercises/${encodeURIComponent(id)}`)
        pending.set(key, request)
      }
      try {
        const response = await request
        if (scope !== sessionScope() || daily !== this.daily || !current()) return null
        if (response.id !== id || !hasUsableContent(response.content_json)) throw new ApiError(502, 'ai_bad_output')
        const existing = this.taskById(id)
        if (!existing || existing.is_completed) return existing
        const index = this.daily!.tasks.findIndex(t => t.id === id)
        this.daily!.tasks[index] = response
        cacheContent(id, response.content_json)
        return response
      } finally {
        if (pending.get(key) === request) pending.delete(key)
      }
    },
    taskById(id: string): QuestTask | null {
      return this.daily?.tasks.find(t => t.id === id) ?? null
    },
    /** Idempotent: re-entering a task resumes its wall-clock anchor (design §2.4), across reloads via aelp.timers. Needs `daily` for the anchor's date. */
    startTimer(taskId: string, durationMinutes: number, now = Date.now()) {
      this.nowMs = now
      const date = this.daily?.date
      if (!date || this.timers[taskId]) return
      this.timers[taskId] = { startedAt: now, totalSeconds: Math.max(60, Math.floor(durationMinutes * 60)), date }
      writeTimers(this.timers)
    },
    /** Re-renders every countdown from the clock; the page calls it each second and on visibilitychange/focus. */
    tick(now = Date.now()) {
      this.nowMs = now
    },
    elapsedSeconds(taskId: string, now?: number): number {
      const at = now ?? this.nowMs
      const t = this.timers[taskId]
      return t ? Math.max(0, Math.floor((at - t.startedAt) / 1000)) : 0
    },
    remainingSeconds(taskId: string, now?: number): number {
      const at = now ?? this.nowMs
      const t = this.timers[taskId]
      return t ? Math.max(0, t.totalSeconds - this.elapsedSeconds(taskId, at)) : 0
    },
    /** Drops anchors from another day, for tasks not in today's list, or for tasks already completed. */
    pruneTimers() {
      if (!this.daily) return
      const today = this.daily
      for (const [id, t] of Object.entries(this.timers)) {
        const task = today.tasks.find(x => x.id === id)
        if (t.date !== today.date || !task || task.is_completed) Reflect.deleteProperty(this.timers, id)
      }
      writeTimers(this.timers)
    },
    async complete(exerciseId: string, durationSeconds: number, userAnswers?: Record<string, string>): Promise<CompleteResult> {
      const body: Record<string, unknown> = { exercise_id: exerciseId, duration_seconds: clampDuration(durationSeconds) }
      if (userAnswers && Object.keys(userAnswers).length > 0) body.user_answers = userAnswers
      const scope = sessionScope()
      const daily = this.daily
      const wasMet = this.daily?.is_target_met ?? false
      const res = await useApi().post<ProgressResponse>('/api/v1/quests/progress', body)
      if (scope !== sessionScope() || daily !== this.daily) return { ...res, targetMetChanged: false }
      if (this.daily) {
        this.daily.accumulated_seconds = res.daily_seconds_spent
        this.daily.is_target_met = res.is_target_met
        const t = this.daily.tasks.find(x => x.id === exerciseId)
        if (t) t.is_completed = true
      }
      Reflect.deleteProperty(this.timers, exerciseId)
      writeTimers(this.timers)
      return { ...res, targetMetChanged: res.is_target_met && !wasMet }
    },
  },
})
