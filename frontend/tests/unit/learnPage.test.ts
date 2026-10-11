import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick, reactive } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ContentViewer from '~/components/learn/ContentViewer.vue'
import CountdownTimer from '~/components/learn/CountdownTimer.vue'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import ItemPassage from '~/components/learn/ItemPassage.vue'
import { useQuestStore } from '~/stores/quest'
import { useAuthStore } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
import StateBlock from '~/components/ui/StateBlock.vue'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const navigateTo = vi.fn()
const route = reactive({ params: { id: 'ex-2' } })
vi.stubGlobal('useRoute', () => route)
vi.stubGlobal('navigateTo', navigateTo)

const { default: LearnPage } = await import('~/pages/learn/[id].vue')

const wrappers: VueWrapper[] = []
const T0 = Date.parse('2026-09-23T13:00:00Z')

const daily = {
  date: '2026-09-23',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 0,
  is_target_met: false,
  tasks: [
    { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu', duration_minutes: 10, is_completed: false, content_json: { questions: [{ id: 'q1', prompt: 'Choose', options: { A: 'Yes', B: 'No' } }] } },
  ],
}

function mountPage() {
  const w = mount(LearnPage, {
    global: {
      components: { AppButton, AppCard, ContentViewer, CountdownTimer, StateBlock },
      mocks: { navigateTo },
      stubs: { NuxtLink: { template: '<a><slot /></a>' } },
    },
  })
  wrappers.push(w)
  return w
}

function deferred<T>() { let resolve!: (v: T) => void; let reject!: (e: unknown) => void; const promise = new Promise<T>((r, j) => { resolve = r; reject = j }); return { promise, resolve, reject } }
const reading = { ...daily.tasks[0]!, content_json: { content: { passage: 'A contract.', glossary: [{ term: 'contract', meaning_vi: 'hợp đồng', definition: 'agreement' }], questions: [{ id: 'q1', prompt: 'Choose', options: { A: 'Yes' } }] } } }
const outline = () => ({ ...daily, tasks: [{ ...daily.tasks[0]!, content_json: { title: 'outline' } }] })

describe('/learn/:id (wireframe 7.3)', () => {
  beforeEach(() => {
    daily.tasks[0]!.is_completed = false
    route.params.id = 'ex-2'
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
    navigateTo.mockReset()
  })

  afterEach(() => {
    wrappers.splice(0).forEach(w => w.unmount())
    vi.useRealTimers()
  })

  it('the countdown and button follow the wall clock on visibilitychange, not the interval', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] })
    vi.setSystemTime(T0)
    api.get.mockResolvedValue(daily)
    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('10:00')
    const button = w.find('button.mt-4')
    if (!button.exists()) throw new Error('no button')
    expect(button.text()).toBe('Hoàn thành')
    expect(button.attributes('disabled')).toBeDefined()

    vi.setSystemTime(T0 + 600_000)
    document.dispatchEvent(new Event('visibilitychange'))
    await nextTick()

    expect(w.text()).toContain('00:00')
    expect(button.text()).toBe('Hết giờ — Hoàn thành')
    expect(button.attributes('disabled')).toBeUndefined()

    vi.setSystemTime(T0 + 599_000)
    window.dispatchEvent(new Event('focus'))
    await nextTick()
    expect(button.text()).toBe('Hoàn thành')

    vi.setSystemTime(T0 + 600_000)
    window.dispatchEvent(new Event('focus'))
    await nextTick()
    expect(button.text()).toBe('Hết giờ — Hoàn thành')
  })

  it('completing posts the wall-clock elapsed', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] })
    vi.setSystemTime(T0)
    api.get.mockResolvedValue(daily)
    const w = mountPage()
    await flushPromises()

    vi.setSystemTime(T0 + 615_000)
    document.dispatchEvent(new Event('visibilitychange'))
    await nextTick()

    api.post.mockResolvedValue({ daily_seconds_spent: 1215, daily_minutes_spent: 20, is_target_met: false, pet_health: 100, streak_count: 5 })
    const button = w.find('button.mt-4')
    if (!button.exists()) throw new Error('no button')
    await button.trigger('click')
    await flushPromises()

    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', { exercise_id: 'ex-2', duration_seconds: 615 })
    expect(localStorage.getItem('aelp.timers')).toBe('{}')
  })
  it('deep links load daily then hydrate without raw flash; the timer starts only when ready', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    const pending = deferred<typeof reading>()
    api.get.mockResolvedValueOnce(outline()).mockReturnValueOnce(pending.promise)
    const w = mountPage(); await flushPromises()
    expect(api.get.mock.calls.map(c => c[0])).toEqual(['/api/v1/quests/daily', '/api/v1/quests/exercises/ex-2'])
    expect(w.find('pre').exists()).toBe(false)
    expect(w.findComponent(ContentViewer).exists()).toBe(false)
    expect(useQuestStore().timers['ex-2']).toBeUndefined()
    expect(w.find('button.mt-4').attributes('disabled')).toBeDefined()
    vi.setSystemTime(T0 + 30_000); pending.resolve(reading); await flushPromises()
    expect(w.findComponent(ItemPassage).exists()).toBe(true)
    expect(useQuestStore().timers['ex-2']?.startedAt).toBe(T0 + 30_000)
  })

  it('failed generation keeps expired completion disabled and retry preserves the anchor', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    const q = useQuestStore(); q.daily = outline(); q.startTimer('ex-2', 10, T0 - 700_000)
    api.get.mockRejectedValueOnce(new ApiError(503, 'ai_unavailable')).mockResolvedValueOnce(reading)
    const w = mountPage(); await flushPromises()
    expect(w.find('[role="status"]').text()).toContain('Thử lại')
    expect(w.find('button.mt-4').attributes('disabled')).toBeDefined()
    await w.find('[role="status"] button').trigger('click'); await flushPromises()
    expect(q.timers['ex-2']?.startedAt).toBe(T0 - 700_000)
    expect(w.find('button.mt-4').attributes('disabled')).toBeUndefined()
    expect(api.get).toHaveBeenCalledTimes(2)
  })

  it('cached typed passage works offline with no request and glossary time contributes to unchanged answers/duration', async () => {
    vi.useFakeTimers({ toFake: ['Date', 'setInterval', 'clearInterval'] }); vi.setSystemTime(T0)
    const connected = vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    const q = useQuestStore(); q.daily = { ...daily, tasks: [{ ...reading }] }
    const w = mountPage(); await flushPromises()
    expect(w.find('button.mt-4').attributes('disabled')).toBeDefined()
    expect(api.get).not.toHaveBeenCalled()
    await w.find('[aria-label="Giải nghĩa contract"]').trigger('click')
    vi.setSystemTime(T0 + 45_000); window.dispatchEvent(new Event('focus')); await nextTick()
    expect(w.text()).toContain('09:15')
    expect(w.text()).toContain('hợp đồng')
    await w.findComponent(ContentViewer).findComponent(AppButton).trigger('click')
    await w.find('[role="radio"]').trigger('click')
    await w.findComponent(ContentViewer).findComponent(AppButton).trigger('click')
    connected.mockReturnValue(true); window.dispatchEvent(new Event('online')); await nextTick()
    api.post.mockResolvedValueOnce({ daily_seconds_spent: 45, is_target_met: false })
    await w.find('button.mt-4').trigger('click'); await flushPromises()
    expect(api.post).toHaveBeenCalledWith('/api/v1/quests/progress', { exercise_id: 'ex-2', duration_seconds: 45, user_answers: { q1: 'A' } })
  })

  it('late route/account responses never show the previous task or mutate its content', async () => {
    const q = useQuestStore(); q.daily = outline()
    const pending = deferred<typeof reading>(); api.get.mockReturnValueOnce(pending.promise)
    const w = mountPage(); await flushPromises(); route.params.id = 'other'; await flushPromises()
    pending.resolve(reading); await flushPromises()
    expect(w.findComponent(ContentViewer).exists()).toBe(false)
    expect(q.taskById('ex-2')?.content_json).toEqual({ title: 'outline' })
    daily.tasks[0]!.is_completed = false
    route.params.id = 'ex-2'; const late = deferred<typeof reading>(); api.get.mockReturnValueOnce(late.promise); await flushPromises()
    useAuthStore().accessToken = 'new-account'; await flushPromises()
    late.resolve(reading); await flushPromises()
    expect(w.findComponent(ContentViewer).exists()).toBe(false)
    expect(q.daily).toBeNull()
  })

  it('failed progress retains answers and time for retry', async () => {
    const q = useQuestStore(); q.daily = { ...daily, tasks: [{ ...reading }] }
    const w = mountPage(); await flushPromises()
    await w.findComponent(ContentViewer).findComponent(AppButton).trigger('click')
    await w.find('[role="radio"]').trigger('click')
    await w.findComponent(ContentViewer).findComponent(AppButton).trigger('click')
    const anchor = q.timers['ex-2']?.startedAt
    api.post.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ daily_seconds_spent: 10, is_target_met: false })
    await w.find('button.mt-4').trigger('click'); await flushPromises()
    expect(q.timers['ex-2']?.startedAt).toBe(anchor)
    expect(w.text()).toContain('Chưa ghi được tiến độ')
    await w.find('button.mt-4').trigger('click'); await flushPromises()
    expect(api.post.mock.calls[1]![1].user_answers).toEqual({ q1: 'A' })
  })

  it('offline outlines offer connection/retry instead of JSON and cannot complete', async () => {
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    const q = useQuestStore(); q.daily = outline()
    const w = mountPage(); await flushPromises()
    expect(w.find('[role="status"]').text()).toContain('Cần kết nối để tải nội dung')
    expect(w.findComponent(ContentViewer).exists()).toBe(false)
    expect(w.find('pre').exists()).toBe(false)
    expect(w.find('button.mt-4').attributes('disabled')).toBeDefined()
    expect(api.get).not.toHaveBeenCalled()
    expect(q.timers['ex-2']).toBeUndefined()
  })

  it('offers retry if daily refresh invalidates an in-flight hydration result', async () => {
    const q = useQuestStore(); q.daily = outline()
    const pending = deferred<typeof reading>(); api.get.mockReturnValueOnce(pending.promise)
    const w = mountPage(); await flushPromises()
    q.daily = outline(); pending.resolve(reading); await flushPromises()
    expect(w.find('[role="status"]').text()).toContain('Thử lại')
    expect(w.text()).not.toContain('Bạn đã xem hết nội dung')
    expect(w.find('button.mt-4').attributes('disabled')).toBeDefined()
  })

})
