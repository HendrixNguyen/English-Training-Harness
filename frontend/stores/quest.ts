import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'
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

interface Timer {
  totalSeconds: number
  remainingSeconds: number
}

function rank(type: string): number {
  const i = (TASK_ORDER as readonly string[]).indexOf(type)
  return i === -1 ? TASK_ORDER.length : i
}

export const useQuestStore = defineStore('quest', {
  state: () => ({
    daily: null as DailyQuests | null,
    noRoadmap: false,
    loading: false,
    error: null as string | null,
    timers: {} as Record<string, Timer>,
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
    async load() {
      this.loading = true
      this.error = null
      try {
        this.daily = await useApi().get<DailyQuests>('/api/v1/quests/daily')
        this.noRoadmap = false
      } catch (e) {
        if (e instanceof ApiError && e.code === 'no_active_roadmap') {
          this.daily = null
          this.noRoadmap = true
        } else {
          this.error = e instanceof ApiError ? e.code : 'network_error'
        }
      } finally {
        this.loading = false
      }
    },
    taskById(id: string): QuestTask | null {
      return this.daily?.tasks.find(t => t.id === id) ?? null
    },
    /** Idempotent: re-entering a task resumes its timer (design §2.4). */
    startTimer(taskId: string, durationMinutes: number) {
      if (this.timers[taskId]) return
      const total = Math.max(60, Math.floor(durationMinutes * 60))
      this.timers[taskId] = { totalSeconds: total, remainingSeconds: total }
    },
    tick(taskId: string, seconds = 1) {
      const t = this.timers[taskId]
      if (t) t.remainingSeconds = Math.max(0, t.remainingSeconds - seconds)
    },
    elapsedSeconds(taskId: string): number {
      const t = this.timers[taskId]
      return t ? t.totalSeconds - t.remainingSeconds : 0
    },
    async complete(exerciseId: string, durationSeconds: number, userAnswers?: Record<string, string>): Promise<ProgressResponse> {
      const body: Record<string, unknown> = { exercise_id: exerciseId, duration_seconds: clampDuration(durationSeconds) }
      if (userAnswers && Object.keys(userAnswers).length > 0) body.user_answers = userAnswers
      const res = await useApi().post<ProgressResponse>('/api/v1/quests/progress', body)
      if (this.daily) {
        this.daily.accumulated_seconds = res.daily_seconds_spent
        this.daily.is_target_met = res.is_target_met
        const t = this.daily.tasks.find(x => x.id === exerciseId)
        if (t) t.is_completed = true
      }
      Reflect.deleteProperty(this.timers, exerciseId)
      return res
    },
  },
})
