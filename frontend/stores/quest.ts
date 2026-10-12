import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'
import { useAuthStore } from '~/stores/auth'
import { hasTrustedContent, questionsOf, classifyContent } from '~/utils/content'
import { clearOwnerDrafts } from '~/utils/learningSession'
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
  content_origin?: 'generated' | 'legacy' | 'outline'
  content_revision?: string | null
  quiz_result?: QuizResult | null
}

export interface QuizAnswer { question_id: string, selected_option: string, correct_option: string | null, is_correct: boolean | null, explanation: string | null }
export interface QuizResult { status: 'graded' | 'ungraded', answered_count: number, question_count: number, correct_count: number | null, score_percent: number | null, duration_seconds: number, answers: QuizAnswer[] }

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
  quiz_result?: QuizResult | null
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
const pendingDaily = new WeakMap<object, Promise<boolean>>()
const pendingCompletion = new WeakMap<object, Map<string, Promise<CompleteResult>>>()
function sessionScope(): string {
  const auth = useAuthStore()
  return auth.user?.id ? `owner:${auth.user.id}` : `anonymous:${auth.accessToken ?? ''}`
}
function cachedContent(id: string): QuestTask | undefined {
  try {
    const cache = JSON.parse(storageOrNull()?.getItem(contentCacheKey) ?? 'null')
    if (cache?.owner === (useAuthStore().user?.id ?? null)) return cache.tasks?.[id]
  } catch { /* denied storage or an obsolete cache */ }
  return undefined
}
function cacheContent(id: string, content: QuestTask) {
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
    dailyMutation: 0,
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
      if (this.accountScope.startsWith('owner:')) clearOwnerDrafts(this.accountScope.slice(6))
      this.accountScope = sessionScope()
      this.dailyMutation += 1
      this.daily = null
      this.noRoadmap = false
      this.error = null
      this.loading = false
      this.timers = {}
      writeTimers(this.timers)
    },
    async load(fresh = false): Promise<boolean> {
      this.resetForAccount()
      const scope = sessionScope()
      const existing = pendingDaily.get(this)
      if (existing) {
        if (!fresh) return existing
        await existing
        if (scope !== sessionScope()) return false
        return this.load()
      }
      this.loading = true
      this.error = null
      const mutation = this.dailyMutation
      const request = (async () => {
        try {
          const response = await useApi().get<DailyQuests>('/api/v1/quests/daily')
          // A read started before accepted progress cannot undo that save.
          if (scope !== sessionScope() || mutation !== this.dailyMutation) return false
          for (const task of response.tasks) {
            if (hasTrustedContent(task)) cacheContent(task.id, task)
            else if (!task.is_completed && task.content_origin !== 'generated') {
              const previous = this.taskById(task.id)
              const cached = previous && hasTrustedContent(previous) ? previous : cachedContent(task.id)
              const offline = typeof navigator !== 'undefined' && !navigator.onLine
              if (cached && hasTrustedContent(cached) && cached.id === task.id && cached.task_type === task.task_type && (task.content_revision === cached.content_revision || (offline && task.content_origin === 'outline' && !task.content_revision))) {
                task.content_json = cached.content_json
                task.content_origin = cached.content_origin
                task.content_revision = cached.content_revision
              }
            }
          }
          this.daily = response
          this.pruneTimers()
          this.noRoadmap = false
          return true
        } catch (e) {
          if (scope !== sessionScope() || mutation !== this.dailyMutation) return false
          if (e instanceof ApiError && e.code === 'no_active_roadmap') {
            this.daily = null
            this.noRoadmap = true
          } else {
            this.error = e instanceof ApiError ? e.code : 'network_error'
          }
          return false
        } finally {
          if (scope === sessionScope()) this.loading = false
        }
      })()
      pendingDaily.set(this, request)
      try { return await request } finally { if (pendingDaily.get(this) === request) pendingDaily.delete(this) }
    },
    /** Fetch only outlines; the caller can invalidate application after a route change. */
    async hydrateTask(id: string, current: () => boolean = () => true, repair = false): Promise<QuestTask | null> {
      this.resetForAccount()
      const task = this.taskById(id)
      if (!task || task.is_completed || hasTrustedContent(task)) return task
      if (!repair && task.content_origin === 'generated' && !task.content_revision) throw new ApiError(409, 'exercise_content_invalid')
      const scope = sessionScope()
      const daily = this.daily
      const original = task.content_json
      const revision = task.content_revision
      let pending = pendingHydration.get(this)
      if (!pending) { pending = new Map(); pendingHydration.set(this, pending) }
      const key = `${scope}:${id}:${repair}`
      let request = pending.get(key)
      if (!request) {
        request = useApi().get<QuestTask>(`/api/v1/quests/exercises/${encodeURIComponent(id)}${repair ? '?repair=true' : ''}`)
        pending.set(key, request)
      }
      try {
        const response = await request
        if (scope !== sessionScope() || daily !== this.daily || !current() || this.taskById(id)?.content_json !== original || this.taskById(id)?.content_revision !== revision) return null
        if (response.id !== id || !hasTrustedContent(response)) throw new ApiError(502, 'ai_bad_output')
        const existing = this.taskById(id)
        if (!existing || existing.is_completed) return existing
        const index = this.daily!.tasks.findIndex(t => t.id === id)
        this.daily!.tasks[index] = response
        cacheContent(id, response)
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
    async complete(exerciseId: string, durationSeconds: number, userAnswers?: Record<string, string>, revision?: string, current: () => boolean = () => true): Promise<CompleteResult> {
      this.resetForAccount()
      const task = this.taskById(exerciseId)
      if (revision !== undefined) {
        if (!task || !hasTrustedContent(task) || task.content_revision !== revision) throw new ApiError(409, 'exercise_content_changed')
        if (task.is_completed) {
          return { daily_seconds_spent: this.accumulatedSeconds, daily_minutes_spent: Math.floor(this.accumulatedSeconds / 60), is_target_met: this.targetMet, quiz_result: task.quiz_result ?? null, targetMetChanged: false }
        }
        const qs = questionsOf(classifyContent(task.content_json, task.task_type))
        if (!userAnswers || Object.keys(userAnswers).length !== qs.length || qs.some(q => !Object.hasOwn(q.options, userAnswers[q.id] ?? ''))) throw new ApiError(400, 'invalid_request')
      }
      const scope = sessionScope()
      const key = `${scope}:${exerciseId}:${revision ?? ''}`
      let pending = pendingCompletion.get(this)
      if (!pending) { pending = new Map(); pendingCompletion.set(this, pending) }
      const existing = pending.get(key)
      if (existing) return existing
      const body: Record<string, unknown> = { exercise_id: exerciseId, duration_seconds: clampDuration(durationSeconds) }
      if (revision !== undefined) { body.content_revision = revision; body.user_answers = { ...userAnswers } }
      else if (userAnswers && Object.keys(userAnswers).length) body.user_answers = { ...userAnswers }
      const daily = this.daily
      const wasMet = this.daily?.is_target_met ?? false
      const request = (async () => {
        const res = await useApi().post<ProgressResponse>('/api/v1/quests/progress', body)
        if (scope !== sessionScope() || daily !== this.daily || !current() || (revision !== undefined && this.taskById(exerciseId)?.content_revision !== revision)) return { ...res, targetMetChanged: false }
        if (this.daily) {
          this.dailyMutation += 1
          this.daily.accumulated_seconds = res.daily_seconds_spent
          this.daily.is_target_met = res.is_target_met
          const t = this.daily.tasks.find(x => x.id === exerciseId)
          if (t && (revision === undefined || res.quiz_result)) { t.is_completed = true; t.quiz_result = res.quiz_result ?? null }
        }
        if (revision === undefined || res.quiz_result) {
          Reflect.deleteProperty(this.timers, exerciseId)
          writeTimers(this.timers)
        }
        return { ...res, targetMetChanged: res.is_target_met && !wasMet }
      })()
      pending.set(key, request)
      try { return await request } finally { if (pending.get(key) === request) pending.delete(key) }
    },
  },
})
