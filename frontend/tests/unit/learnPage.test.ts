import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick, reactive } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ContentViewer from '~/components/learn/ContentViewer.vue'
import CountdownTimer from '~/components/learn/CountdownTimer.vue'
import AppCard from '~/components/ui/AppCard.vue'
import ItemPassage from '~/components/learn/ItemPassage.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { useQuestStore, type QuestTask, type QuizResult } from '~/stores/quest'
import { useAuthStore } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
import { clearOwnerDrafts } from '~/utils/learningSession'
const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const navigateTo = vi.fn()
const route = reactive({ params: { id: 'ex-2' } })
vi.stubGlobal('useRoute', () => route)
vi.stubGlobal('navigateTo', navigateTo)
const { default: LearnPage } = await import('~/pages/learn/[id].vue')
const wrappers: VueWrapper[] = []
const T0 = Date.parse('2026-09-23T13:00:00Z')
const question = { id: 'q1', prompt: 'What did they sign?', options: { A: 'A letter', B: 'A contract' }, answer: 'B', explanation: 'They signed a contract.' }
const reading = (): QuestTask => ({ id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu', duration_minutes: 10, is_completed: false, content_origin: 'legacy', content_revision: 'rev-1', quiz_result: null, content_json: { content_origin: 'legacy', content: { passage: 'A contract.', glossary: [{ term: 'contract', meaning_vi: 'hợp đồng', definition: 'agreement' }], questions: [question] } } })
const outline = (): QuestTask => ({ ...reading(), content_origin: 'outline', content_revision: null, content_json: { content: { passage: 'Draft only.' } } })
const daily = (task = reading()) => ({ date: '2026-09-23', day_number: 3, total_minutes_required: 30, accumulated_seconds: 0, is_target_met: false, tasks: [task] })
const grade = (duration = 47): QuizResult => ({ status: 'graded', answered_count: 1, question_count: 1, correct_count: 0, score_percent: 0, duration_seconds: duration, answers: [{ question_id: 'q1', selected_option: 'A', correct_option: 'B', is_correct: false, explanation: 'Server explanation.' }] })
const progress = (result = grade()) => ({ daily_seconds_spent: 1215, daily_minutes_spent: 20, is_target_met: false, pet_health: 100, streak_count: 5, quiz_result: result })
function mountPage() {
  const w = mount(LearnPage, { global: { components: { AppCard, ContentViewer, CountdownTimer, StateBlock }, mocks: { navigateTo }, stubs: { NuxtLink: { template: '<a><slot /></a>' } } } })
  wrappers.push(w); return w
}
const action = (w: VueWrapper, label: string) => w.findAll('button').find(b => b.text() === label)!
function deferred<T>() { let resolve!: (v: T) => void; let reject!: (e: unknown) => void; const promise = new Promise<T>((r, j) => { resolve = r; reject = j }); return { promise, resolve, reject } }
async function finish(w: VueWrapper) { await action(w, 'Đọc xong →').trigger('click'); await w.find('[role="radio"]').trigger('click'); await action(w, 'Trả lời').trigger('click'); await action(w, 'Tiếp tục').trigger('click') }
beforeEach(() => {
  route.params.id = 'ex-2'; setActivePinia(createPinia()); localStorage.clear(); clearOwnerDrafts('owner'); clearOwnerDrafts('other')
  useAuthStore().user = { id: 'owner', email: 'owner@example.test', full_name: 'Owner', cefr_current: 'B1' }
  useAuthStore().accessToken = 'token'; api.get.mockReset(); api.post.mockReset(); navigateTo.mockReset()
  vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(true)
})
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.useRealTimers(); vi.restoreAllMocks() })
describe('/learn/:id learning sessions', () => {
  it('clock snaps on visibility/focus and expiry never permits unanswered completion', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    api.get.mockResolvedValue(daily()); const w = mountPage(); await flushPromises()
    expect(w.text()).toContain('10:00'); expect(action(w, 'Hoàn thành')).toBeUndefined()
    vi.setSystemTime(T0 + 600_000); document.dispatchEvent(new Event('visibilitychange')); await nextTick()
    expect(w.text()).toContain('00:00'); expect(action(w, 'Hoàn thành')).toBeUndefined()
    await action(w, 'Đọc xong →').trigger('click'); await action(w, 'Trả lời').trigger('click')
    expect(w.findAll('[role="radio"]')).toHaveLength(2); expect(api.post).not.toHaveBeenCalled()
    vi.setSystemTime(T0 + 599_000); window.dispatchEvent(new Event('focus')); await nextTick(); expect(w.text()).toContain('00:01')
  })
  it('requires final continue and one explicit save with revision/full answers; authoritative summary persists', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    api.get.mockResolvedValue(daily()); const w = mountPage(); await flushPromises()
    await action(w, 'Đọc xong →').trigger('click'); await w.find('[role="radio"]').trigger('click'); await action(w, 'Trả lời').trigger('click')
    expect(action(w, 'Hoàn thành')).toBeUndefined(); expect(w.text()).toContain('They signed a contract.'); expect(api.post).not.toHaveBeenCalled()
    await action(w, 'Tiếp tục').trigger('click'); expect(w.text()).toContain('Chưa lưu kết quả'); expect(w.text()).not.toContain('0%')
    vi.setSystemTime(T0 + 615_000); const pending = deferred<ReturnType<typeof progress>>(); api.post.mockReturnValue(pending.promise)
    const save = action(w, 'Hoàn thành'); await save.trigger('click'); await save.trigger('click'); expect(api.post).toHaveBeenCalledTimes(1)
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', { exercise_id: 'ex-2', duration_seconds: 615, content_revision: 'rev-1', user_answers: { q1: 'A' } })
    pending.resolve(progress(grade(615))); await flushPromises()
    expect(w.text()).toContain('Đã lưu kết quả'); expect(w.text()).toContain('Đúng 0 / 1 · 0%'); expect(w.text()).toContain('Server explanation.')
    expect(w.text()).toContain('Thời gian học: 10 phút 15 giây'); expect(localStorage.getItem('aelp.timers')).toBe('{}'); expect(navigateTo).not.toHaveBeenCalled()
    w.unmount(); const reopened = mountPage(); await flushPromises(); expect(reopened.text()).toContain('Thời gian học: 10 phút 15 giây'); expect(api.post).toHaveBeenCalledTimes(1)
  })
  it('hydrates outlines with no JSON flash and starts the original clock only after trust', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    const pending = deferred<QuestTask>(); api.get.mockResolvedValueOnce(daily(outline())).mockReturnValueOnce(pending.promise)
    const w = mountPage(); await flushPromises(); expect(w.find('pre').exists()).toBe(false); expect(useQuestStore().timers['ex-2']).toBeUndefined()
    vi.setSystemTime(T0 + 30_000); pending.resolve(reading()); await flushPromises()
    expect(w.findComponent(ItemPassage).exists()).toBe(true); expect(useQuestStore().timers['ex-2']?.startedAt).toBe(T0 + 30_000)
  })
  it('offline trusted content/gloss study works, save is disabled and the unchanged anchor includes lookback time', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    const connected = vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); expect(api.get).not.toHaveBeenCalled()
    await w.find('[aria-label="Giải nghĩa contract"]').trigger('click')
    vi.setSystemTime(T0 + 45_000); window.dispatchEvent(new Event('focus')); await nextTick(); expect(w.text()).toContain('09:15'); expect(w.text()).toContain('hợp đồng')
    await finish(w); expect(action(w, 'Hoàn thành').attributes('disabled')).toBeDefined()
    connected.mockReturnValue(true); window.dispatchEvent(new Event('online')); await nextTick(); api.post.mockResolvedValue(progress(grade(45)))
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises(); expect(api.post.mock.calls[0]![1].duration_seconds).toBe(45)
  })
  it('offline outlines cannot render or start timers', async () => {
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false); useQuestStore().daily = daily(outline())
    const w = mountPage(); await flushPromises(); expect(w.text()).toContain('Cần kết nối để tải nội dung'); expect(w.find('pre').exists()).toBe(false)
    expect(w.findComponent(ContentViewer).exists()).toBe(false); expect(useQuestStore().timers['ex-2']).toBeUndefined(); expect(api.get).not.toHaveBeenCalled()
  })
  it('invalid stored generation requires explicit repair and no automatic regeneration', async () => {
    useQuestStore().daily = daily({ ...outline(), content_origin: 'generated', content_json: { content_schema: 1, content: { passage: 'bad' } } })
    const w = mountPage(); await flushPromises(); expect(w.text()).toContain('Nội dung này cần được tạo lại.'); expect(api.get).not.toHaveBeenCalled()
    api.get.mockRejectedValueOnce(new ApiError(502, 'ai_bad_output')).mockResolvedValueOnce(reading())
    await action(w, 'Tạo lại nội dung').trigger('click'); await flushPromises(); expect(api.get).toHaveBeenCalledWith('/api/v1/quests/exercises/ex-2?repair=true')
    expect(w.text()).toContain('Chưa tạo lại được nội dung'); expect(api.get).toHaveBeenCalledTimes(1)
    await action(w, 'Tạo lại nội dung').trigger('click'); await flushPromises(); expect(w.findComponent(ItemPassage).exists()).toBe(true)
  })
  it('restores pending selection and locked feedback/lookback across back/reload and token refresh', async () => {
    useQuestStore().daily = daily(); let w = mountPage(); await flushPromises(); await action(w, 'Đọc xong →').trigger('click')
    await action(w, 'Xem lại đoạn').trigger('click'); await w.find('[role="radio"]').trigger('click'); expect(w.text()).toContain('Đã trả lời 0 / 1')
    useAuthStore().accessToken = 'refreshed'; await flushPromises(); expect(w.find('[aria-checked="true"]').exists()).toBe(true)
    w.unmount(); w = mountPage(); await flushPromises(); expect(w.find('[aria-checked="true"]').exists()).toBe(true); expect(action(w, 'Ẩn đoạn văn')).toBeDefined()
    await action(w, 'Trả lời').trigger('click'); w.unmount(); w = mountPage(); await flushPromises()
    expect(w.text()).toContain('Trượt…'); expect(w.text()).toContain('Đã trả lời 1 / 1'); expect(w.find('[role="radio"]').attributes('disabled')).toBeDefined()
  })
  it('confirmed rejection retains answers/duration for explicit retry', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    useQuestStore().daily = daily(); const w = mountPage(); await flushPromises(); await finish(w); vi.setSystemTime(T0 + 47_000)
    api.post.mockRejectedValueOnce(new ApiError(400, 'invalid_request')).mockResolvedValueOnce(progress())
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises(); expect(w.text()).toContain('Chưa ghi được tiến độ')
    vi.setSystemTime(T0 + 100_000); await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    expect(api.post.mock.calls.map(c => c[1].duration_seconds)).toEqual([47, 47]); expect(api.post.mock.calls[1]![1].user_answers).toEqual({ q1: 'A' })
  })
  it('lost response reconciles matching saved result without a second POST', async () => {
    useQuestStore().daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    api.post.mockRejectedValue(new TypeError('lost response')); api.get.mockResolvedValue(daily({ ...reading(), is_completed: true, quiz_result: grade() }))
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    expect(w.text()).toContain('Đã lưu kết quả'); expect(w.text()).toContain('0 phút 47 giây'); expect(api.post).toHaveBeenCalledTimes(1); expect(action(w, 'Hoàn thành')).toBeUndefined()
  })
  it('unknown outcome offers read-only check; confirmed incomplete permits explicit retry', async () => {
    useQuestStore().daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    api.post.mockRejectedValue(new TypeError('lost response')); api.get.mockRejectedValueOnce(new TypeError('offline')).mockResolvedValueOnce(daily())
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises(); expect(w.text()).toContain('Chưa xác nhận được kết quả'); expect(action(w, 'Hoàn thành')).toBeUndefined()
    await action(w, 'Kiểm tra lại').trigger('click'); await flushPromises(); expect(api.post).toHaveBeenCalledTimes(1); expect(action(w, 'Hoàn thành')).toBeDefined()
  })
  it('completed legacy ungraded/historical results never invent score or duration and never POST', async () => {
    useQuestStore().daily = daily({ ...reading(), is_completed: true, quiz_result: null }); let w = mountPage(); await flushPromises()
    expect(w.text()).toContain('Không có kết quả câu hỏi đã lưu'); expect(w.text()).toContain('Thời gian học: chưa có dữ liệu'); expect(w.text()).not.toContain('0%'); expect(api.post).not.toHaveBeenCalled()
    w.unmount(); useQuestStore().daily = daily({ ...reading(), is_completed: true, quiz_result: { ...grade(), status: 'ungraded', correct_count: null, score_percent: null, answers: [] } }); w = mountPage(); await flushPromises()
    expect(w.text()).toContain('Không chấm điểm'); expect(w.text()).not.toContain('%'); expect(api.post).not.toHaveBeenCalled()
  })
  it('content revision replacement clears incompatible choices and late route/account responses are suppressed', async () => {
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); await action(w, 'Đọc xong →').trigger('click'); await w.find('[role="radio"]').trigger('click')
    q.daily.tasks[0] = { ...reading(), content_revision: 'rev-2' }; await flushPromises(); expect(w.text()).toContain('Nội dung đã thay đổi'); expect(w.findComponent(ItemPassage).exists()).toBe(true)
    expect(w.find('[aria-checked="true"]').exists()).toBe(false)
    q.daily = daily(outline()); await flushPromises(); const pending = deferred<QuestTask>(); api.get.mockReturnValue(pending.promise)
    await action(w, 'Thử lại').trigger('click'); await flushPromises(); route.params.id = 'other'; await flushPromises(); pending.resolve(reading()); await flushPromises()
    expect(w.findComponent(ContentViewer).exists()).toBe(false)
    useAuthStore().user = { ...useAuthStore().user!, id: 'other' }; api.get.mockResolvedValue(daily()); await flushPromises(); expect(q.accountScope).toBe('owner:other')
  })
})


describe('save response lifetime', () => {
  it('reconciles a successful response when a daily replacement prevented applying it locally', async () => {
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    const pending = deferred<ReturnType<typeof progress>>(); api.post.mockReturnValueOnce(pending.promise)
    await action(w, 'Hoàn thành').trigger('click')
    q.daily = daily()
    api.get.mockResolvedValueOnce(daily({ ...reading(), is_completed: true, quiz_result: grade() }))
    pending.resolve(progress()); await flushPromises()
    expect(w.text()).toContain('Đã lưu kết quả')
    expect(api.get).toHaveBeenCalledWith('/api/v1/quests/daily')
    expect(api.post).toHaveBeenCalledTimes(1)
  })
})

describe('study and response boundaries', () => {
  it('zero-question legacy material must be studied then saves an empty map with explicitly ungraded result', async () => {
    useQuestStore().daily = daily({ ...reading(), content_json: { words: [{ term: 'Hello', definition: 'A greeting', example: 'Hello, friend.' }, { term: 'Goodbye', definition: 'A farewell' }] }, task_type: 'vocabulary' })
    const w = mountPage(); await flushPromises(); expect(action(w, 'Hoàn thành')).toBeUndefined()
    await action(w, 'Tiếp tục').trigger('click'); expect(w.text()).toContain('Goodbye'); expect(action(w, 'Hoàn thành')).toBeUndefined()
    await action(w, 'Tiếp tục').trigger('click'); expect(w.text()).toContain('Không chấm điểm')
    api.post.mockResolvedValue(progress({ status: 'ungraded', answered_count: 0, question_count: 0, correct_count: null, score_percent: null, duration_seconds: 1, answers: [] }))
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    expect(api.post.mock.calls[0]![1]).toMatchObject({ content_revision: 'rev-1', user_answers: {} }); expect(w.text()).toContain('Đã lưu kết quả'); expect(w.text()).not.toContain('%')
  })
  it('denied storage keeps the page usable with a visible warning and server save still works', async () => {
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => { throw Error('denied') })
    useQuestStore().daily = daily(); const w = mountPage(); await flushPromises(); expect(w.text()).toContain('Tiến độ chưa lưu chỉ giữ trong lần mở này.')
    await finish(w); api.post.mockResolvedValue(progress()); await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    expect(w.text()).toContain('Đã lưu kết quả'); expect(api.post).toHaveBeenCalledTimes(1)
  })
  it('a pending save on another route cannot mutate task completion or navigate', async () => {
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    const pending = deferred<ReturnType<typeof progress>>(); api.post.mockReturnValueOnce(pending.promise)
    await action(w, 'Hoàn thành').trigger('click'); route.params.id = 'other'; await flushPromises(); pending.resolve(progress()); await flushPromises()
    expect(q.taskById('ex-2')?.is_completed).toBe(false); expect(w.text()).not.toContain('Đã lưu kết quả'); expect(navigateTo).not.toHaveBeenCalled()
  })
  it('account replacement clears the former owner draft and suppresses its late save', async () => {
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    const pending = deferred<ReturnType<typeof progress>>(); api.post.mockReturnValueOnce(pending.promise)
    await action(w, 'Hoàn thành').trigger('click'); api.get.mockResolvedValueOnce(daily())
    useAuthStore().user = { ...useAuthStore().user!, id: 'other' }; await flushPromises(); pending.resolve(progress()); await flushPromises()
    expect(q.taskById('ex-2')?.is_completed).toBe(false); expect(w.text()).not.toContain('Trượt…'); expect(navigateTo).not.toHaveBeenCalled()
    const keys = Array.from({ length: localStorage.length }, (_, i) => localStorage.key(i))
    expect(keys.some(k => k?.startsWith('aelp.learning-draft.v1:') && k.includes('owner'))).toBe(false)
  })
})

describe('fresh reconciliation reads', () => {
  it('never permits retry from a daily request started before the ambiguous save', async () => {
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    const old = deferred<ReturnType<typeof daily>>(); api.get.mockReturnValueOnce(old.promise).mockResolvedValueOnce(daily({ ...reading(), is_completed: true, quiz_result: grade() }))
    const startedBeforeSave = q.load()
    api.post.mockRejectedValueOnce(new TypeError('lost response'))
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises(); old.resolve(daily()); await startedBeforeSave; await flushPromises()
    expect(w.text()).toContain('Đã lưu kết quả'); expect(api.get).toHaveBeenCalledTimes(2); expect(api.post).toHaveBeenCalledTimes(1)
  })
})

describe('reviewed successful-save and recovery races', () => {
  it('keeps a successful POST result and progress when an older incomplete daily GET arrives afterward', async () => {
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    const old = deferred<ReturnType<typeof daily>>(); api.get.mockReturnValueOnce(old.promise)
    const preSaveRead = q.load()
    api.post.mockResolvedValueOnce(progress())
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    expect(w.text()).toContain('Đã lưu kết quả'); expect(q.taskById('ex-2')?.quiz_result).toEqual(grade())
    old.resolve(daily()); await preSaveRead; await flushPromises()
    expect(w.text()).toContain('Đã lưu kết quả'); expect(w.text()).toContain('Server explanation.')
    expect(q.taskById('ex-2')?.quiz_result).toEqual(grade()); expect(q.accumulatedSeconds).toBe(1215)
    expect(action(w, 'Hoàn thành')).toBeUndefined()
    await q.complete('ex-2', 47, { q1: 'A' }, 'rev-1'); expect(api.post).toHaveBeenCalledTimes(1)
    api.get.mockResolvedValueOnce({ ...daily({ ...reading(), is_completed: true, quiz_result: grade() }), accumulated_seconds: 1300 })
    expect(await q.load(true)).toBe(true); expect(q.accumulatedSeconds).toBe(1300)
  })
  it('does not reset another account session when the prior account conflict refresh resolves false', async () => {
    const q = useQuestStore(); q.daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    const refresh = deferred<ReturnType<typeof daily>>(); api.get.mockReturnValueOnce(refresh.promise).mockResolvedValueOnce(daily())
    api.post.mockRejectedValueOnce(new ApiError(409, 'exercise_content_changed'))
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    useAuthStore().user = { ...useAuthStore().user!, id: 'other' }; await flushPromises()
    await action(w, 'Đọc xong →').trigger('click'); await w.find('[role="radio"]').trigger('click')
    expect(w.find('[aria-checked="true"]').exists()).toBe(true)
    refresh.resolve(daily()); await flushPromises()
    expect(w.findComponent(ContentViewer).exists()).toBe(true); expect(w.find('[aria-checked="true"]').exists()).toBe(true)
    expect(w.text()).not.toContain('Chưa tải được nội dung học. Thử lại.'); expect(api.get).toHaveBeenCalledTimes(2)
  })
  it('does not reopen a newly selected route when the prior task conflict refresh resolves true', async () => {
    const second = { ...reading(), id: 'ex-other', title: 'Another lesson', content_revision: 'other-rev' }
    const twoTasks = () => ({ ...daily(), tasks: [reading(), second] })
    const q = useQuestStore(); q.daily = twoTasks(); const w = mountPage(); await flushPromises(); await finish(w)
    const refresh = deferred<ReturnType<typeof twoTasks>>(); api.get.mockReturnValueOnce(refresh.promise)
    api.post.mockRejectedValueOnce(new ApiError(409, 'exercise_content_invalid'))
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    route.params.id = 'ex-other'; await flushPromises(); await action(w, 'Đọc xong →').trigger('click'); await w.find('[role="radio"]').trigger('click')
    const activeViewer = w.findComponent(ContentViewer).element
    refresh.resolve(twoTasks()); await flushPromises()
    expect(w.findComponent(ContentViewer).element).toBe(activeViewer); expect(w.find('[aria-checked="true"]').exists()).toBe(true)
    expect(w.text()).not.toContain('Chưa tải được nội dung học. Thử lại.')
  })
  it('still reopens the current task after the expected replacement revision arrives', async () => {
    useQuestStore().daily = daily(); const w = mountPage(); await flushPromises(); await finish(w)
    const refresh = deferred<ReturnType<typeof daily>>(); api.get.mockReturnValueOnce(refresh.promise)
    api.post.mockRejectedValueOnce(new ApiError(409, 'exercise_content_changed'))
    await action(w, 'Hoàn thành').trigger('click'); await flushPromises()
    refresh.resolve(daily({ ...reading(), content_revision: 'replacement' })); await flushPromises()
    expect(w.findComponent(ItemPassage).exists()).toBe(true); expect(w.find('[aria-checked="true"]').exists()).toBe(false)
    expect(w.text()).toContain('Nội dung đã thay đổi. Học lại từ đầu.')
  })
})
