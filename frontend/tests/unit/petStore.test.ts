import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { REVIVE_SECONDS, REVIVE_STORAGE_KEY, usePetStore } = await import('~/stores/pet')

const status = { plant_name: 'My Green Buddy', health_points: 80, stage: 'sprout', current_streak: 5, last_practiced_at: '2026-09-21T20:15:00Z', shields: 0, last_shield_used_on: null }

describe('usePetStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
  })

  it('load fetches GET /api/v1/pet/status', async () => {
    api.get.mockResolvedValue(status)
    const pet = usePetStore()
    await pet.load()
    expect(api.get).toHaveBeenCalledWith('/api/v1/pet/status')
    expect(pet.status?.health_points).toBe(80)
    expect(pet.isWilted).toBe(false)
  })

  it('load exposes shields and last_shield_used_on', async () => {
    api.get.mockResolvedValue({ ...status, shields: 1, last_shield_used_on: '2026-09-24' })
    const pet = usePetStore()
    await pet.load()
    expect(pet.status?.shields).toBe(1)
    expect(pet.status?.last_shield_used_on).toBe('2026-09-24')
  })

  it('isWilted when health is 0 or stage is wilted', async () => {
    api.get.mockResolvedValue({ ...status, health_points: 0, stage: 'wilted' })
    const pet = usePetStore()
    await pet.load()
    expect(pet.isWilted).toBe(true)
  })

  it('applyProgress updates health and streak from a §6.2 progress response', async () => {
    api.get.mockResolvedValue(status)
    const pet = usePetStore()
    await pet.load()
    pet.applyProgress({ pet_health: 100, streak_count: 6 })
    expect(pet.status).toMatchObject({ health_points: 100, current_streak: 6 })
  })

  it('revive: first call starts a challenge anchored to the current daily seconds', async () => {
    api.get.mockResolvedValue({ ...status, health_points: 0, stage: 'wilted' })
    api.post.mockResolvedValue({ revival_passed: false, pet_state: { health_points: 0, stage: 'wilted', current_streak: 0 } })
    const pet = usePetStore()
    await pet.load()
    const res = await pet.revive('2026-09-23', 120)
    expect(api.post).toHaveBeenCalledWith('/api/v1/pet/revive', { answers: {} })
    expect(res?.revival_passed).toBe(false)
    expect(pet.challenge).toEqual({ date: '2026-09-23', startSeconds: 120 })
    expect(JSON.parse(localStorage.getItem(REVIVE_STORAGE_KEY) ?? 'null')).toEqual({ date: '2026-09-23', startSeconds: 120 })
    expect(pet.challengeProgress(120 + 450)).toBe(450)
    expect(REVIVE_SECONDS).toBe(900)
  })

  it('revive: a pass applies pet_state and clears the challenge', async () => {
    localStorage.setItem(REVIVE_STORAGE_KEY, JSON.stringify({ date: '2026-09-23', startSeconds: 0 }))
    api.get.mockResolvedValue({ ...status, health_points: 0, stage: 'wilted' })
    api.post.mockResolvedValue({ revival_passed: true, pet_state: { health_points: 50, stage: 'sprout', current_streak: 0 } })
    const pet = usePetStore()
    pet.hydrateChallenge()
    await pet.load()
    const res = await pet.revive('2026-09-23', 1000)
    expect(res?.revival_passed).toBe(true)
    expect(pet.status).toMatchObject({ health_points: 50, stage: 'sprout', current_streak: 0 })
    expect(pet.challenge).toBeNull()
    expect(localStorage.getItem(REVIVE_STORAGE_KEY)).toBeNull()
  })

  it('revive: 409 pet_not_wilted sets notWilted and returns null', async () => {
    api.post.mockRejectedValue(new ApiError(409, 'pet_not_wilted'))
    const pet = usePetStore()
    await expect(pet.revive('2026-09-23', 0)).resolves.toBeNull()
    expect(pet.notWilted).toBe(true)
  })
})
