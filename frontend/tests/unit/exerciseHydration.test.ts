import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useQuestStore, type QuestTask } from '~/stores/quest'
import { useAuthStore } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const typed = { content_origin: 'legacy', content: { passage: 'A contract.', questions: [] } }
const hydrated = (): QuestTask => ({ ...task(), content_origin: 'legacy', content_revision: 'rev', content_json: typed })
const task = (): QuestTask => ({ content_origin: 'outline', content_revision: null, id: 'reading', task_type: 'reading', title: 'Read', duration_minutes: 10, is_completed: false, content_json: { title: 'outline' } })
const daily = () => ({ date: '2026-10-11', day_number: 1, total_minutes_required: 30, accumulated_seconds: 0, is_target_met: false, tasks: [task()] })
function deferred<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(r => resolve = r); return { promise, resolve } }
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset() })
afterEach(() => vi.restoreAllMocks())
describe('typed exercise hydration', () => {
  it('deduplicates endpoint requests, replaces matching task and caches typed content for offline daily outlines', async () => {
    const q = useQuestStore(); q.daily = daily()
    const pending = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(pending.promise)
    const a = q.hydrateTask('reading'); const b = q.hydrateTask('reading')
    expect(api.get).toHaveBeenCalledTimes(1)
    pending.resolve(hydrated())
    await Promise.all([a, b])
    expect(q.taskById('reading')?.content_json).toEqual(typed)
    setActivePinia(createPinia()); const cached = useQuestStore()
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    api.get.mockResolvedValue(daily()); await cached.load()
    api.get.mockClear(); await cached.hydrateTask('reading')
    expect(cached.taskById('reading')?.content_json).toEqual(typed)
    expect(api.get).not.toHaveBeenCalled()
  })
  it('uses legacy cached content without a schema marker or endpoint', async () => {
    const q = useQuestStore(); q.daily = daily(); q.daily.tasks[0] = hydrated()
    expect(await q.hydrateTask('reading')).toEqual(q.daily.tasks[0])
    expect(api.get).not.toHaveBeenCalled()
  })
  it('allows retry after failure without changing anchors', async () => {
    const q = useQuestStore(); q.daily = daily(); q.startTimer('reading', 10, 1000)
    api.get.mockRejectedValueOnce(new ApiError(503, 'ai_unavailable')).mockResolvedValueOnce(hydrated())
    await expect(q.hydrateTask('reading')).rejects.toMatchObject({ code: 'ai_unavailable' })
    await q.hydrateTask('reading')
    expect(q.timers.reading?.startedAt).toBe(1000)
    expect(api.get).toHaveBeenCalledTimes(2)
  })
  it('ignores foreign IDs, route cancellations and account changes', async () => {
    const q = useQuestStore(); q.daily = daily()
    api.get.mockResolvedValueOnce({ ...hydrated(), id: 'foreign' })
    await expect(q.hydrateTask('reading')).rejects.toThrow()
    let current = true
    const pending = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(pending.promise)
    const request = q.hydrateTask('reading', () => current); current = false
    pending.resolve(hydrated()); await request
    expect(q.daily.tasks[0]!.content_json).toEqual({ title: 'outline' })
    const next = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(next.promise)
    const late = q.hydrateTask('reading'); useAuthStore().accessToken = 'new-session'
    next.resolve(hydrated()); await late
    expect(q.daily.tasks[0]!.content_json).toEqual({ title: 'outline' })
  })
  it('never overwrites completion with stale hydration', async () => {
    const q = useQuestStore(); q.daily = daily()
    const pending = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(pending.promise)
    const request = q.hydrateTask('reading'); q.daily.tasks[0]!.is_completed = true
    pending.resolve(hydrated()); await request
    expect(q.daily.tasks[0]!.is_completed).toBe(true)
  })
  it('discards late daily responses and lets a new account load independently', async () => {
    const q = useQuestStore()
    const old = deferred<ReturnType<typeof daily>>()
    const next = deferred<ReturnType<typeof daily>>()
    api.get.mockReturnValueOnce(old.promise).mockReturnValueOnce(next.promise)
    const a = q.load(); const duplicate = q.load()
    expect(api.get).toHaveBeenCalledTimes(1)
    useAuthStore().accessToken = 'next-session'
    const b = q.load()
    expect(api.get).toHaveBeenCalledTimes(2)
    next.resolve({ ...daily(), day_number: 2 }); await b
    old.resolve(daily()); await Promise.all([a, duplicate])
    expect(q.daily?.day_number).toBe(2)
    expect(q.loading).toBe(false)
  })
  it('keeps content cached for one account out of another account', async () => {
    const auth = useAuthStore()
    auth.user = { id: 'owner', email: 'owner@example.test', full_name: 'Owner', cefr_current: 'B1' }
    const q = useQuestStore(); q.daily = daily()
    api.get.mockResolvedValueOnce(hydrated()); await q.hydrateTask('reading')
    auth.user = { ...auth.user, id: 'other' }
    api.get.mockResolvedValueOnce(daily()); await q.load()
    expect(q.daily?.tasks[0]?.content_json).toEqual({ title: 'outline' })
  })
  it('does not apply late progress to the next account or delete its timer', async () => {
    const q = useQuestStore(); q.daily = daily()
    const pending = deferred<{ daily_seconds_spent: number, daily_minutes_spent: number, is_target_met: boolean }>()
    api.post.mockReturnValueOnce(pending.promise)
    const request = q.complete('reading', 60)
    useAuthStore().accessToken = 'next-account'
    q.resetForAccount(); q.daily = daily(); q.startTimer('reading', 10, 2000)
    pending.resolve({ daily_seconds_spent: 60, daily_minutes_spent: 1, is_target_met: false }); await request
    expect(q.daily.accumulated_seconds).toBe(0)
    expect(q.daily.tasks[0]?.is_completed).toBe(false)
    expect(q.timers.reading?.startedAt).toBe(2000)
  })
  it('ignores a hydration response after daily data has been replaced', async () => {
    const q = useQuestStore(); q.daily = daily()
    const pending = deferred<QuestTask>(); api.get.mockReturnValueOnce(pending.promise)
    const request = q.hydrateTask('reading'); q.daily = { ...daily(), date: '2026-10-12' }
    pending.resolve(hydrated()); await request
    expect(q.daily.tasks[0]?.content_json).toEqual({ title: 'outline' })
  })

})

describe('cache provenance and revision', () => {
  it('old body-only cache cannot masquerade as validated content and token renewal preserves stable-owner state', async () => {
    const auth = useAuthStore(); auth.user = { id: 'owner', email: 'owner@test.example', full_name: 'Owner', cefr_current: 'B1' }; auth.accessToken = 'first'
    localStorage.setItem('aelp.exercise-content', JSON.stringify({ owner: 'owner', tasks: { reading: typed } }))
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    const q = useQuestStore(); api.get.mockResolvedValueOnce(daily()); await q.load()
    expect(q.taskById('reading')?.content_origin).toBe('outline')
    q.daily = { ...daily(), tasks: [hydrated()] }; q.startTimer('reading', 10, 1000)
    auth.accessToken = 'renewed'; q.resetForAccount()
    expect(q.taskById('reading')?.content_revision).toBe('rev'); expect(q.timers.reading?.startedAt).toBe(1000)
  })
  it('never substitutes a cached previous revision for a changed authoritative task', async () => {
    const q = useQuestStore(); q.daily = { ...daily(), tasks: [hydrated()] }
    api.get.mockResolvedValueOnce({ ...daily(), tasks: [{ ...task(), content_revision: 'changed' }] }); await q.load()
    expect(q.taskById('reading')?.content_json).toEqual({ title: 'outline' }); expect(q.taskById('reading')?.content_revision).toBe('changed')
  })
  it('revised complete guards partial answers, sends full empty maps for study, and deduplicates in-flight posts', async () => {
    const q = useQuestStore(); q.daily = { ...daily(), tasks: [{ ...hydrated(), content_json: { questions: [{ id: 'q', prompt: 'Which item comes next?', options: { A: 'First', B: 'Second' } }] } }] }
    await expect(q.complete('reading', 60, {}, 'rev')).rejects.toMatchObject({ code: 'invalid_request' }); expect(api.post).not.toHaveBeenCalled()
    await expect(q.complete('reading', 60, { q: 'A' }, 'wrong')).rejects.toMatchObject({ code: 'exercise_content_changed' }); expect(api.post).not.toHaveBeenCalled()
    const pending = deferred<{ daily_seconds_spent: number, daily_minutes_spent: number, is_target_met: boolean }>(); api.post.mockReturnValueOnce(pending.promise)
    const a = q.complete('reading', 60, { q: 'A' }, 'rev'); const b = q.complete('reading', 60, { q: 'A' }, 'rev'); expect(api.post).toHaveBeenCalledTimes(1)
    expect(api.post.mock.calls[0]![1]).toEqual({ exercise_id: 'reading', duration_seconds: 60, content_revision: 'rev', user_answers: { q: 'A' } })
    pending.resolve({ daily_seconds_spent: 60, daily_minutes_spent: 1, is_target_met: false }); await Promise.all([a, b])
  })
})
