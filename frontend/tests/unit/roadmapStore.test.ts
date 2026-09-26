import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { useRoadmapStore } = await import('~/stores/roadmap')

function dayFixture(dayNumber: number, met: boolean) {
  return {
    day_number: dayNumber,
    date: `2026-09-${String(dayNumber).padStart(2, '0')}`,
    title: `Day ${dayNumber}`,
    tasks: [{ task_type: 'vocabulary', title: 'x', duration_minutes: 10 }],
    minutes_spent: met ? 30 : 0,
    is_target_met: met,
  }
}

const outline = {
  roadmap_id: 'rm-1',
  title: 'Business English',
  cefr_level: 'B1',
  created_at: '2026-09-01T00:00:00Z',
  day_number: 3,
  modules: [
    { week: 1, title: 'Week 1', focus: 'intro', days: [dayFixture(1, true), dayFixture(2, true), dayFixture(3, false)] },
  ],
}

describe('useRoadmapStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    api.get.mockReset()
  })

  it('load fetches GET /api/v1/roadmap and stores the outline', async () => {
    api.get.mockResolvedValue(outline)
    const r = useRoadmapStore()
    await r.load()
    expect(api.get).toHaveBeenCalledWith('/api/v1/roadmap')
    expect(r.outline).toEqual(outline)
    expect(r.noRoadmap).toBe(false)
    expect(r.loading).toBe(false)
    expect(r.error).toBeNull()
  })

  it('maps 404 no_active_roadmap to the empty state without an error', async () => {
    api.get.mockRejectedValue(new ApiError(404, 'no_active_roadmap'))
    const r = useRoadmapStore()
    await r.load()
    expect(r.noRoadmap).toBe(true)
    expect(r.outline).toBeNull()
    expect(r.error).toBeNull()
  })

  it('maps any other ApiError to its code', async () => {
    api.get.mockRejectedValue(new ApiError(500, 'internal_error'))
    const r = useRoadmapStore()
    await r.load()
    expect(r.error).toBe('internal_error')
    expect(r.noRoadmap).toBe(false)
  })

  it('maps a non-ApiError rejection to network_error', async () => {
    api.get.mockRejectedValue(new TypeError('Failed to fetch'))
    const r = useRoadmapStore()
    await r.load()
    expect(r.error).toBe('network_error')
  })

  it('completedDays counts days with is_target_met across all modules', async () => {
    api.get.mockResolvedValue(outline)
    const r = useRoadmapStore()
    await r.load()
    expect(r.completedDays).toBe(2)
  })
})
