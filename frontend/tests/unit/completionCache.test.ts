import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useQuestStore, type DailyQuests, type QuizResult } from '~/stores/quest'
import { useAuthStore } from '~/stores/auth'
const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const user = { id: 'owner', email: 'owner@test.example', full_name: 'Owner', cefr_current: 'B1' }
const result: QuizResult = { status: 'ungraded', answered_count: 0, question_count: 0, correct_count: null, score_percent: null, duration_seconds: 65, answers: [] }
const daily = (): DailyQuests => ({ date: '2026-10-12', day_number: 1, total_minutes_required: 30, accumulated_seconds: 0, is_target_met: false, tasks: [{ id: 'reading', task_type: 'reading', title: 'Read', duration_minutes: 10, is_completed: false, content_origin: 'legacy', content_revision: 'rev', quiz_result: null, content_json: { content_origin: 'legacy', content: { passage: 'A contract.', questions: [] } } }] })
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset(); useAuthStore().user = user })
async function save() {
  const q = useQuestStore(); api.get.mockResolvedValueOnce(daily()); await q.load()
  api.post.mockResolvedValueOnce({ daily_seconds_spent: 65, daily_minutes_spent: 1, is_target_met: false, quiz_result: result })
  await q.complete('reading', 65, {}, 'rev'); return q
}
describe('accepted completion cache', () => {
  it('keeps saved result and progress after a later ordinary cached read, hub reopen and explicit second complete', async () => {
    const q = await save(); api.get.mockResolvedValueOnce(daily()); await q.load()
    expect(q.taskById('reading')).toMatchObject({ is_completed: true, quiz_result: result })
    expect(q.accumulatedSeconds).toBe(65)
    expect(JSON.parse(localStorage.getItem('aelp.exercise-content')!).tasks.reading).toMatchObject({ is_completed: true, quiz_result: result })
    await q.complete('reading', 65, {}, 'rev'); expect(api.post).toHaveBeenCalledTimes(1)
  })
  it('restores accepted completion from the durable cache in a re-created store against an offline initial outline', async () => {
    await save(); setActivePinia(createPinia()); useAuthStore().user = user
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    const stale = daily(); stale.tasks[0]!.content_origin = 'outline'; stale.tasks[0]!.content_revision = null; stale.tasks[0]!.content_json = { content: {} }
    const q = useQuestStore(); api.get.mockResolvedValueOnce(stale); await q.load()
    expect(q.taskById('reading')).toMatchObject({ is_completed: true, content_revision: 'rev', quiz_result: result })
    expect(q.accumulatedSeconds).toBe(65)
    await q.complete('reading', 65, {}, 'rev'); expect(api.post).toHaveBeenCalledTimes(1)
    vi.restoreAllMocks()
  })
  it.each(['owner', 'day', 'day_number', 'revision', 'type', 'task', 'fresh'])('does not carry accepted state across %s replacement', async change => {
    const q = await save(); const next = daily()
    if (change === 'owner') useAuthStore().user = { ...user, id: 'other' }
    if (change === 'day') next.date = '2026-10-13'
    if (change === 'day_number') next.day_number = 2
    if (change === 'revision') next.tasks[0]!.content_revision = 'replacement'
    if (change === 'type') next.tasks[0]!.task_type = 'practice'
    if (change === 'task') next.tasks[0]!.id = 'other'
    api.get.mockResolvedValueOnce(next); await q.load(change === 'fresh')
    expect(q.daily?.tasks[0]?.is_completed).toBe(false); expect(q.accumulatedSeconds).toBe(0)
  })
  it('does not resurrect accepted content after a real revision replacement or malformed checkpoint', async () => {
    const q = await save(); const next = daily(); next.tasks[0]!.content_revision = 'replacement'
    api.get.mockResolvedValueOnce(next); await q.load(); api.get.mockResolvedValueOnce(daily()); await q.load()
    expect(q.taskById('reading')?.is_completed).toBe(false)
    setActivePinia(createPinia()); useAuthStore().user = user
    localStorage.setItem('aelp.accepted-daily.v1', JSON.stringify({ owner: user.id, daily: { ...daily(), tasks: [null] } }))
    const recreated = useQuestStore(); api.get.mockResolvedValueOnce(daily()); expect(await recreated.load()).toBe(true)
  })
  it('keeps the in-memory accepted state when storage is denied', async () => {
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => { throw new Error('denied') })
    const q = await save(); api.get.mockResolvedValueOnce(daily()); await q.load()
    expect(q.taskById('reading')?.quiz_result).toEqual(result); expect(q.accumulatedSeconds).toBe(65)
    vi.restoreAllMocks()
  })
})
