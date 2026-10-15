import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '~/utils/apiClient'
import { useAuthStore } from '~/stores/auth'
import { useRoadmapStore } from '~/stores/roadmap'
import { dayState, calendarLabel } from '~/utils/roadmap'
const api = { get: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const day = { day_number: 1, date: '2026-10-01', title: 'Greetings', tasks: [], minutes_spent: 0, is_target_met: false }
const outline = { roadmap_id: 'one', title: 'Real journey', cefr_level: 'B1', today: '2026-10-03', day_number: 3, modules: [{ week: 1, title: 'Small talk', focus: 'Introductions', days: [day] }] }
beforeEach(() => { setActivePinia(createPinia()); localStorage.clear(); api.get.mockReset() })
describe('owned roadmap read and calendar truth', () => {
  it('never awards skipped or partial days a star, even at 30 minutes', () => {
    expect(dayState(day, outline.today)).toBe('missed')
    expect(dayState({ ...day, minutes_spent: 30 }, outline.today)).toBe('partial')
    expect(dayState({ ...day, is_target_met: true }, outline.today)).toBe('cleared')
    expect(dayState(day, day.date)).toBe('today')
    expect(dayState(day, '2026-09-30')).toBe('locked')
    expect(dayState({ ...day, date: '2026-10-28' }, '2026-10-29')).toBe('missed')
    expect(calendarLabel('2026-10-01')).toBe('T5 1/10')
  })
  it('always rereads replacement and retains owned data under transient error', async () => {
    const store = useRoadmapStore()
    api.get.mockResolvedValueOnce(outline).mockResolvedValueOnce({ ...outline, roadmap_id: 'two' }).mockRejectedValueOnce(new Error('offline'))
    await store.load(); await store.load(); await store.load()
    expect(api.get).toHaveBeenCalledTimes(3); expect(store.outline?.roadmap_id).toBe('two'); expect(store.error).toBe('network_error')
  })
  it('clears absence and never adopts an older overlapping read', async () => {
    let resolve!: (v: typeof outline) => void
    api.get.mockImplementationOnce(() => new Promise(r => { resolve = r })).mockResolvedValueOnce({ ...outline, roadmap_id: 'new' })
    const store = useRoadmapStore(); const old = store.load(); await store.load(); resolve(outline); await old
    expect(store.outline?.roadmap_id).toBe('new')
    api.get.mockRejectedValueOnce(new ApiError(404, 'no_active_roadmap')); await store.load()
    expect(store.outline).toBeNull(); expect(store.noRoadmap).toBe(true)
  })
  it('hides old owner immediately and rejects late session reads', async () => {
    const auth = useAuthStore(); auth.user = { id: 'a', email: '', full_name: '', cefr_current: null }
    const store = useRoadmapStore(); api.get.mockResolvedValueOnce(outline); await store.load()
    let resolve!: (v: typeof outline) => void
    api.get.mockImplementationOnce(() => new Promise(r => { resolve = r })); const pending = store.load()
    auth.user = { ...auth.user, id: 'b' }; auth.sessionEpoch++
    expect(store.outline).toBeNull(); resolve(outline); await pending; expect(store.outline).toBeNull()
  })
  it('central owner replacement clears memory while token renewal keeps the same journey', async () => {
    const auth = useAuthStore()
    const user = { id: 'a', email: '', full_name: '', cefr_current: null }
    await auth.signIn({ access_token: 'one', token_type: 'Bearer', expires_in: 60, user })
    const store = useRoadmapStore(); api.get.mockResolvedValueOnce(outline); await store.load()
    await auth.signIn({ access_token: 'renewed', token_type: 'Bearer', expires_in: 60, user })
    expect(store.outline?.roadmap_id).toBe('one')
    await auth.signIn({ access_token: 'new', token_type: 'Bearer', expires_in: 60, user: { ...user, id: 'b' } })
    expect(store.data).toBeNull(); expect(store.outline).toBeNull()
  })
  it('page lifetime cancellation retires late reads while retaining accepted memory', async () => {
    const store = useRoadmapStore(); api.get.mockResolvedValueOnce(outline); await store.load()
    let resolve!: (v: typeof outline) => void
    api.get.mockImplementationOnce(() => new Promise(r => { resolve = r })); const pending = store.load()
    store.cancelPending(); resolve({ ...outline, roadmap_id: 'late' }); await pending
    expect(store.outline?.roadmap_id).toBe('one'); expect(store.loading).toBe(false)
  })
  it('invalidation retires regeneration reads and sign-out clears memory', async () => {
    const store = useRoadmapStore(); let resolve!: (v: typeof outline) => void
    api.get.mockImplementationOnce(() => new Promise(r => { resolve = r })); const pending = store.load()
    store.invalidate(); resolve(outline); await pending; expect(store.outline).toBeNull()
    api.get.mockResolvedValueOnce(outline); await store.load(); await useAuthStore().signOut(); expect(store.outline).toBeNull()
  })
})
