import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { useQuestStore, TIMER_STORAGE_KEY } = await import('~/stores/quest')

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

const T0 = Date.parse('2026-09-23T13:00:00Z')

async function loaded() {
  api.get.mockResolvedValue(daily)
  const q = useQuestStore()
  await q.load()
  return q
}

describe('useQuestStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
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

  it('timers are anchored to the wall clock: elapsed is the real delta however often tick runs', async () => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(T0)
    const q = await loaded()
    q.startTimer('ex-2', 10)
    expect(q.timers['ex-2']).toEqual({ startedAt: T0, totalSeconds: 600, date: '2026-09-23' })
    expect(q.remainingSeconds('ex-2')).toBe(600)

    vi.setSystemTime(T0 + 18_000)
    q.tick()
    expect(q.elapsedSeconds('ex-2')).toBe(18)
    expect(q.remainingSeconds('ex-2')).toBe(582)

    for (let i = 0; i < 20; i++) q.tick()
    expect(q.elapsedSeconds('ex-2')).toBe(18)

    vi.setSystemTime(T0 + 700_000)
    expect(q.elapsedSeconds('ex-2', T0 + 700_000)).toBe(700)
    expect(q.remainingSeconds('ex-2', T0 + 700_000)).toBe(0)

    q.startTimer('ex-2', 10) // re-entry resumes, never resets
    expect(q.timers['ex-2']?.startedAt).toBe(T0)
  })

  it('a timer survives a reload: a fresh store hydrates aelp.timers and load keeps today\'s entry', async () => {
    const q = await loaded()
    q.startTimer('ex-2', 10, T0)
    expect(JSON.parse(localStorage.getItem(TIMER_STORAGE_KEY) ?? 'null')).toEqual({
      'ex-2': { startedAt: T0, totalSeconds: 600, date: '2026-09-23' },
    })

    setActivePinia(createPinia()) // simulated reload
    const q2 = await loaded()
    expect(q2.elapsedSeconds('ex-2', T0 + 300_000)).toBe(300)
    q2.startTimer('ex-2', 10, T0 + 300_000)
    expect(q2.timers['ex-2']?.startedAt).toBe(T0)
  })

  it('load drops a timer from another day', async () => {
    localStorage.setItem(TIMER_STORAGE_KEY, JSON.stringify({ 'ex-2': { startedAt: T0 - 86_400_000, totalSeconds: 600, date: '2026-09-22' } }))
    const q = await loaded()
    expect(q.timers['ex-2']).toBeUndefined()
    expect(localStorage.getItem(TIMER_STORAGE_KEY)).toBe('{}')

    q.startTimer('ex-2', 10, T0)
    expect(q.timers['ex-2']?.startedAt).toBe(T0)
  })

  it('load drops a timer whose task is already completed', async () => {
    localStorage.setItem(TIMER_STORAGE_KEY, JSON.stringify({
      'ex-1': { startedAt: T0, totalSeconds: 600, date: '2026-09-23' },
      'ex-2': { startedAt: T0, totalSeconds: 600, date: '2026-09-23' },
    }))
    const q = await loaded()
    expect(Object.keys(q.timers)).toEqual(['ex-2'])
    expect(JSON.parse(localStorage.getItem(TIMER_STORAGE_KEY) ?? 'null')).toEqual({
      'ex-2': { startedAt: T0, totalSeconds: 600, date: '2026-09-23' },
    })
  })

  it('the countdown floors at 0 for the button gate while the posted duration is the real elapsed clamped to 3600', async () => {
    const q = await loaded()
    api.post.mockResolvedValue({ daily_seconds_spent: 1200, daily_minutes_spent: 20, is_target_met: false, pet_health: 100, streak_count: 5 })
    q.startTimer('ex-2', 10, T0)
    const now = T0 + 5_000_000
    expect(q.remainingSeconds('ex-2', now)).toBe(0)
    expect(q.elapsedSeconds('ex-2', now)).toBe(5000)
    await q.complete('ex-2', q.elapsedSeconds('ex-2', now))
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', {
      exercise_id: 'ex-2',
      duration_seconds: 3600,
    })
  })

  it('the store works without localStorage and when it throws', async () => {
    vi.stubGlobal('localStorage', undefined)
    setActivePinia(createPinia())
    let q = await loaded()
    q.startTimer('ex-2', 10, T0)
    expect(q.elapsedSeconds('ex-2', T0 + 60_000)).toBe(60)

    vi.stubGlobal('localStorage', {
      getItem() { throw new Error('denied') },
      setItem() { throw new Error('denied') },
      removeItem() { throw new Error('denied') },
    })
    setActivePinia(createPinia())
    q = await loaded()
    q.startTimer('ex-2', 10, T0)
    expect(q.elapsedSeconds('ex-2', T0 + 60_000)).toBe(60)
  })

  it('complete posts the §6.2 body with a clamped duration and applies the response', async () => {
    api.get.mockResolvedValue(daily)
    api.post.mockResolvedValue({ daily_seconds_spent: 1200, daily_minutes_spent: 20, is_target_met: false, pet_health: 100, streak_count: 5 })
    const q = useQuestStore()
    await q.load()
    q.startTimer('ex-2', 10, T0)
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
    expect(localStorage.getItem(TIMER_STORAGE_KEY)).toBe('{}')
  })

  it('complete omits user_answers when none were collected', async () => {
    api.post.mockResolvedValue({ daily_seconds_spent: 60, daily_minutes_spent: 1, is_target_met: false, pet_health: 100, streak_count: 0 })
    const q = useQuestStore()
    await q.complete('ex-2', 60)
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', { exercise_id: 'ex-2', duration_seconds: 60 })
  })
})
