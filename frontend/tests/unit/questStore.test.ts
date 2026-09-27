import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { useQuestStore } = await import('~/stores/quest')

const daily = {
  date: '2026-09-23',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 600,
  is_target_met: false,
  tasks: [
    { id: 'ex-3', task_type: 'practice', title: 'Viết phản hồi', duration_minutes: 10, is_completed: false, content_json: {} },
    { id: 'ex-1', task_type: 'vocabulary', title: 'Từ vựng', duration_minutes: 10, is_completed: true, content_json: {} },
    { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu', duration_minutes: 10, is_completed: false, content_json: {} },
  ],
}

describe('useQuestStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    api.get.mockReset()
    api.post.mockReset()
  })

  it('load fetches GET /api/v1/quests/daily and orders tasks vocabulary → reading → practice', async () => {
    api.get.mockResolvedValue(daily)
    const q = useQuestStore()
    await q.load()
    expect(api.get).toHaveBeenCalledWith('/api/v1/quests/daily')
    expect(q.sortedTasks.map(t => t.id)).toEqual(['ex-1', 'ex-2', 'ex-3'])
    expect(q.nextTaskId).toBe('ex-2')
    expect(q.accumulatedSeconds).toBe(600)
    expect(q.noRoadmap).toBe(false)
    expect(q.loading).toBe(false)
  })

  it('load maps 404 no_active_roadmap to the empty state, other errors to error', async () => {
    api.get.mockRejectedValue(new ApiError(404, 'no_active_roadmap'))
    const q = useQuestStore()
    await q.load()
    expect(q.noRoadmap).toBe(true)
    expect(q.error).toBeNull()

    api.get.mockRejectedValue(new TypeError('Failed to fetch'))
    await q.load()
    expect(q.error).toBe('network_error')
  })

  it('timers count down from duration_minutes and report elapsed seconds', () => {
    const q = useQuestStore()
    q.startTimer('ex-2', 10)
    expect(q.timers['ex-2']?.remainingSeconds).toBe(600)
    q.tick('ex-2', 18)
    expect(q.timers['ex-2']?.remainingSeconds).toBe(582)
    expect(q.elapsedSeconds('ex-2')).toBe(18)
    q.tick('ex-2', 10_000)
    expect(q.timers['ex-2']?.remainingSeconds).toBe(0)
    q.startTimer('ex-2', 10) // re-entry resumes, never resets
    expect(q.elapsedSeconds('ex-2')).toBe(600)
  })

  it('complete posts the §6.2 body with a clamped duration and applies the response', async () => {
    api.get.mockResolvedValue(daily)
    api.post.mockResolvedValue({ daily_seconds_spent: 1200, daily_minutes_spent: 20, is_target_met: false, pet_health: 100, streak_count: 5 })
    const q = useQuestStore()
    await q.load()
    q.startTimer('ex-2', 10)
    const res = await q.complete('ex-2', 0, { q1: 'A' })
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', {
      exercise_id: 'ex-2',
      duration_seconds: 1,
      user_answers: { q1: 'A' },
    })
    expect(res.pet_health).toBe(100)
    expect(q.accumulatedSeconds).toBe(1200)
    expect(q.taskById('ex-2')?.is_completed).toBe(true)
    expect(q.nextTaskId).toBe('ex-3')
    expect(q.timers['ex-2']).toBeUndefined()
  })

  it('complete omits user_answers when none were collected', async () => {
    api.post.mockResolvedValue({ daily_seconds_spent: 60, daily_minutes_spent: 1, is_target_met: false, pet_health: 100, streak_count: 0 })
    const q = useQuestStore()
    await q.complete('ex-2', 60)
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', { exercise_id: 'ex-2', duration_seconds: 60 })
  })

  it('complete reports targetMetChanged only when the response newly meets the target', async () => {
    api.get.mockResolvedValue(daily)
    const q = useQuestStore()
    await q.load()
    expect(q.targetMet).toBe(false)

    api.post.mockResolvedValue({ daily_seconds_spent: 1800, daily_minutes_spent: 30, is_target_met: true, pet_health: 100, streak_count: 6 })
    const first = await q.complete('ex-2', 600)
    expect(first.targetMetChanged).toBe(true)
    expect(q.targetMet).toBe(true)
    expect(first.pet_health).toBe(100)

    const second = await q.complete('ex-3', 600)
    expect(second.targetMetChanged).toBe(false)

    setActivePinia(createPinia())
    api.get.mockResolvedValue(daily)
    const q2 = useQuestStore()
    await q2.load()
    api.post.mockResolvedValue({ daily_seconds_spent: 600, daily_minutes_spent: 10, is_target_met: false, pet_health: 80, streak_count: 5 })
    const third = await q2.complete('ex-1', 600)
    expect(third.targetMetChanged).toBe(false)
  })
})
