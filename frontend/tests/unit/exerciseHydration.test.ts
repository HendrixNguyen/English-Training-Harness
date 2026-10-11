import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useQuestStore, type QuestTask } from '~/stores/quest'
import { useAuthStore } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const typed = { content: { passage: 'A contract.', questions: [] } }
const task = (): QuestTask => ({ id: 'reading', task_type: 'reading', title: 'Read', duration_minutes: 10, is_completed: false, content_json: { title: 'outline' } })
const daily = () => ({ date: '2026-10-11', day_number: 1, total_minutes_required: 30, accumulated_seconds: 0, is_target_met: false, tasks: [task()] })
function deferred<T>() { let resolve!: (value: T) => void; const promise = new Promise<T>(r => resolve = r); return { promise, resolve } }
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset() })
describe('typed exercise hydration', () => {
  it('deduplicates endpoint requests, replaces matching task and caches typed content for offline daily outlines', async () => {
    const q = useQuestStore(); q.daily = daily()
    const pending = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(pending.promise)
    const a = q.hydrateTask('reading'); const b = q.hydrateTask('reading')
    expect(api.get).toHaveBeenCalledTimes(1)
    pending.resolve({ ...task(), content_json: typed })
    await Promise.all([a, b])
    expect(q.taskById('reading')?.content_json).toEqual(typed)
    setActivePinia(createPinia()); const cached = useQuestStore()
    api.get.mockResolvedValue(daily()); await cached.load()
    api.get.mockClear(); await cached.hydrateTask('reading')
    expect(cached.taskById('reading')?.content_json).toEqual(typed)
    expect(api.get).not.toHaveBeenCalled()
  })
  it('uses legacy cached content without a schema marker or endpoint', async () => {
    const q = useQuestStore(); q.daily = daily(); q.daily.tasks[0]!.content_json = typed
    expect(await q.hydrateTask('reading')).toEqual(q.daily.tasks[0])
    expect(api.get).not.toHaveBeenCalled()
  })
  it('allows retry after failure without changing anchors', async () => {
    const q = useQuestStore(); q.daily = daily(); q.startTimer('reading', 10, 1000)
    api.get.mockRejectedValueOnce(new ApiError(503, 'ai_unavailable')).mockResolvedValueOnce({ ...task(), content_json: typed })
    await expect(q.hydrateTask('reading')).rejects.toMatchObject({ code: 'ai_unavailable' })
    await q.hydrateTask('reading')
    expect(q.timers.reading?.startedAt).toBe(1000)
    expect(api.get).toHaveBeenCalledTimes(2)
  })
  it('ignores foreign IDs, route cancellations and account changes', async () => {
    const q = useQuestStore(); q.daily = daily()
    api.get.mockResolvedValueOnce({ ...task(), id: 'foreign', content_json: typed })
    await expect(q.hydrateTask('reading')).rejects.toThrow()
    let current = true
    const pending = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(pending.promise)
    const request = q.hydrateTask('reading', () => current); current = false
    pending.resolve({ ...task(), content_json: typed }); await request
    expect(q.daily.tasks[0]!.content_json).toEqual({ title: 'outline' })
    const next = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(next.promise)
    const late = q.hydrateTask('reading'); useAuthStore().accessToken = 'new-session'
    next.resolve({ ...task(), content_json: typed }); await late
    expect(q.daily.tasks[0]!.content_json).toEqual({ title: 'outline' })
  })
  it('never overwrites completion with stale hydration', async () => {
    const q = useQuestStore(); q.daily = daily()
    const pending = deferred<ReturnType<typeof task>>(); api.get.mockReturnValue(pending.promise)
    const request = q.hydrateTask('reading'); q.daily.tasks[0]!.is_completed = true
    pending.resolve({ ...task(), content_json: typed }); await request
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
    api.get.mockResolvedValueOnce({ ...task(), content_json: typed }); await q.hydrateTask('reading')
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
    pending.resolve({ ...task(), content_json: typed }); await request
    expect(q.daily.tasks[0]?.content_json).toEqual({ title: 'outline' })
  })

})
