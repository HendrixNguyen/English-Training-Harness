import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppHeader from '~/components/AppHeader.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'

import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import HpBar from '~/components/retro/HpBar.vue'
import DayBar from '~/components/retro/DayBar.vue'
import QuestNode from '~/components/retro/QuestNode.vue'
import SpeechBox from '~/components/retro/SpeechBox.vue'
import { useRetroToast } from '~/composables/useRetroToast'

const reduced = ref(false)
vi.mock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => reduced }))

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { default: IndexPage } = await import('~/pages/index.vue')

const STATUS_BEFORE = { plant_name: 'My Green Buddy', health_points: 80, stage: 'sprout', current_streak: 5, last_practiced_at: '2026-09-24T20:00:00Z' }
const STATUS_AFTER = { ...STATUS_BEFORE, health_points: 100, stage: 'sapling', current_streak: 6 }
const DAILY_MET = {
  date: '2026-09-25',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 1800,
  is_target_met: true,
  tasks: [
    { id: 'ex-1', task_type: 'vocabulary', title: 'Từ vựng', duration_minutes: 10, is_completed: true, content_json: {} },
    { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu', duration_minutes: 10, is_completed: true, content_json: {} },
    { id: 'ex-3', task_type: 'practice', title: 'Viết phản hồi', duration_minutes: 10, is_completed: true, content_json: {} },
  ],
}
const DAILY_PARTWAY = { ...DAILY_MET, accumulated_seconds: 600, is_target_met: false, tasks: DAILY_MET.tasks.map((t, i) => ({ ...t, is_completed: i === 0 })) }

function routeGet(status: unknown, daily: unknown) {
  api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? Promise.resolve(status) : Promise.resolve(daily)))
}

const navigate = vi.fn()
const mounted: ReturnType<typeof mount>[] = []
function mountPage() {
  const wrapper = mount(IndexPage, {
    global: {
      components: { AppHeader, StateBlock, RetroPanel, RetroButton, CompanionSprite, HpBar, DayBar, QuestNode, SpeechBox },
      stubs: { NuxtLink: { template: '<a><slot /></a>' } },
      mocks: { navigateTo: navigate },
    },
  })
  mounted.push(wrapper)
  return wrapper
}

describe('/ (wireframe 7.2) growth moment', () => {
  beforeEach(() => {
    reduced.value = false
    setActivePinia(createPinia())
    api.get.mockReset()
    api.post.mockReset()
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    navigate.mockReset()
    vi.stubGlobal('navigateTo', navigate)
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
  })

  afterEach(() => {
    mounted.splice(0).forEach(wrapper => wrapper.unmount())
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('shows the growth moment after a task that met the target', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet(STATUS_AFTER, DAILY_MET)

    const w = mountPage()
    await flushPromises()
    vi.advanceTimersByTime(200)
    await nextTick()

    expect(w.find('[data-growth-chip="growth"]').text()).toBe('+20 máu')
    expect(w.find('[data-growth-chip="streak"]').text()).toBe('🔥 6 ngày')
    expect(w.find('[data-streak-chip]').classes()).toContain('streak-pulse')
    expect(w.text()).toContain('đủ nước')

    vi.advanceTimersByTime(300) // t = 500
    await nextTick()
    expect(w.find('[data-react]').attributes('data-react')).toBe('levelup')

    vi.advanceTimersByTime(1500)
    await nextTick()
    expect(w.find('[data-growth-chip]').exists()).toBe(false)
  })

  it('takes the grow breath when the stage did not change', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE, current_streak: 3, stage: 'sapling' }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet({ ...STATUS_AFTER, current_streak: 6, stage: 'sapling' }, DAILY_MET)

    const w = mountPage()
    await flushPromises()
    vi.advanceTimersByTime(500)
    await nextTick()
    expect(w.find('[data-react]').attributes('data-react')).toBe('hit')

    vi.advanceTimersByTime(1500)
    await nextTick()
    expect(w.find('[data-react]').attributes('data-react')).toBe('idle')
  })

  it('renders the final values after the flip — what a reduced-motion user sees', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet(STATUS_AFTER, DAILY_MET)

    const w = mountPage()
    await flushPromises()

    expect(w.find('[role="meter"]').attributes('aria-valuenow')).toBe('100')
    expect(w.find('[data-stage="sapling"]').exists()).toBe(true)
    expect(w.text()).toContain('Chuỗi ngày')
    expect(w.text()).toContain('đủ nước')
  })

  it('shows nothing on a plain load', async () => {
    const quest = useQuestStore()
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()
    vi.advanceTimersByTime(200)
    await nextTick()

    expect(w.find('[data-growth-chip]').exists()).toBe(false)
    expect(w.find('[data-react]').attributes('data-react')).toBe('idle')
    expect(w.find('[data-streak-chip]').classes()).not.toContain('streak-pulse')
    expect(w.text()).toContain('Tưới cho tớ')
  })

  it('a task that did not meet the target celebrates nothing', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 80, streak_count: 5, targetMetChanged: false })
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()
    vi.advanceTimersByTime(200)
    await nextTick()

    expect(w.find('[data-growth-chip]').exists()).toBe(false)
    expect(w.find('[data-react]').attributes('data-react')).toBe('idle')
    expect(w.find('[data-streak-chip]').classes()).not.toContain('streak-pulse')
    expect(w.text()).toContain('Tưới cho tớ')
  })

  it('the moment is consumed once', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet(STATUS_AFTER, DAILY_MET)

    const w = mountPage()
    await flushPromises()
    w.unmount()

    const w2 = mountPage()
    await flushPromises()
    vi.advanceTimersByTime(200)
    await nextTick()
    expect(w2.find('[data-growth-chip]').exists()).toBe(false)
  })
  it('opens current task, keeps locked tiles inert and permits extra study', async () => {
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)
    const w = mountPage()
    await flushPromises()
    const tiles = w.findAllComponents(QuestNode)
    expect(tiles.map(t => t.props('state'))).toEqual(['done', 'current', 'locked'])
    await tiles[2]!.find('button').trigger('click')
    expect(navigateTo).not.toHaveBeenCalled()
    await w.find('[data-next-task] button').trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-2')
    useQuestStore().daily = { ...DAILY_PARTWAY, is_target_met: true }
    await nextTick()
    expect(w.findAllComponents(QuestNode)[2]!.props('state')).toBe('open')
    await w.findAllComponents(QuestNode)[2]!.find('button').trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-3')
    w.unmount()
  })
  it('hides unknown navigation and sends all-done to roadmap', async () => {
    api.get.mockImplementation(() => new Promise(() => {}))
    const w = mountPage()
    expect(w.find('[data-next-task]').exists()).toBe(false)
    useQuestStore().daily = DAILY_MET
    useQuestStore().loading = false
    await nextTick()
    expect(w.find('[data-next-task]').text()).toContain('Xem hành trình')
    await w.find('[data-next-task] button').trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/roadmap')
    w.unmount()
  })
  it('keeps quests usable with wilted companion and names recovery', async () => {
    routeGet({ ...STATUS_BEFORE, health_points: 0, stage: 'wilted' }, DAILY_PARTWAY)
    const w = mountPage()
    await flushPromises()
    expect(w.find('[data-react="down"]').exists()).toBe(true)
    expect(w.text()).toContain('Hồi sinh My Green Buddy')
    await w.find('[data-next-task] button').trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-2')
    w.unmount()
  })
  it('invites onboarding without inventing progress for a missing roadmap', async () => {
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)
    const w = mountPage()
    await flushPromises()
    useQuestStore().daily = null
    useQuestStore().noRoadmap = true
    await nextTick()
    expect(w.text()).toContain('Cậu chưa có hành trình')
    expect(w.findComponent(DayBar).exists()).toBe(false)
    await w.findAll('button').find(b => b.text().includes('Bắt đầu hành trình'))!.trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/onboarding')
    w.unmount()
  })

  it('retries only the failed region and leaves companion visible', async () => {
    api.get.mockImplementation((path: string) => path.endsWith('/pet/status') ? Promise.resolve(STATUS_BEFORE) : Promise.reject(new Error('unavailable')))
    const w = mountPage()
    await flushPromises()
    expect(w.find('[role="meter"]').exists()).toBe(true)
    expect(w.find('[data-next-task]').exists()).toBe(false)
    expect(w.text()).toContain('Không tải được nhiệm vụ')
    api.get.mockClear()
    await w.findAll('button').find(b => b.text() === 'Thử lại')!.trigger('click')
    await flushPromises()
    expect(api.get).toHaveBeenCalledTimes(1)
    expect(api.get.mock.calls[0]![0]).toContain('/quests/daily')
    w.unmount()
  })
  it('greets day one with the actual companion name and actual target minutes', async () => {
    routeGet(STATUS_BEFORE, { ...DAILY_PARTWAY, day_number: 1, accumulated_seconds: 0, total_minutes_required: 45, tasks: DAILY_MET.tasks.map(t => ({ ...t, is_completed: false })) })
    const w = mountPage()
    await flushPromises()
    expect(w.text()).toContain('Tớ là My Green Buddy')
    expect(w.findComponent(DayBar).text()).toContain('0/45')
    expect(w.findAllComponents(QuestNode)[0]!.props('state')).toBe('current')
    w.unmount()
  })

  it('keeps cached truth offline and queues only one notice', async () => {
    const toast = useRetroToast()
    while (toast.queue.value.length) toast.dismiss()
    vi.spyOn(window.navigator, 'onLine', 'get').mockReturnValue(false)
    usePetStore().status = STATUS_BEFORE
    useQuestStore().daily = DAILY_PARTWAY
    api.get.mockRejectedValue(new Error('offline'))
    const w = mountPage()
    await flushPromises()
    window.dispatchEvent(new Event('offline'))
    window.dispatchEvent(new Event('offline'))
    expect(toast.queue.value).toHaveLength(1)
    expect(w.findComponent(DayBar).text()).toContain('10/30')
    await w.find('[data-next-task] button').trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-2')
    w.unmount()
    toast.dismiss()
  })

  it('shows final HP and a static earned reward for two seconds with reduced motion', async () => {
    reduced.value = true
    vi.stubGlobal('requestAnimationFrame', () => 1)
    const pet = usePetStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    routeGet(STATUS_AFTER, DAILY_MET)
    const w = mountPage()
    expect(w.find('[role="meter"]').attributes('aria-valuenow')).toBe('100')
    expect(w.find('[data-stage="sapling"]').exists()).toBe(true)
    expect(w.find('[data-growth-chip="growth"]').text()).toBe('+20 máu')
    expect(w.find('.return-connector').exists()).toBe(false)
    expect(w.find('.return-chip').exists()).toBe(false)
    expect(w.find('[data-react]').classes().some(c => c.startsWith('retro-'))).toBe(false)
    vi.advanceTimersByTime(1999)
    await nextTick()
    expect(w.find('[data-growth-chip]').exists()).toBe(true)
    vi.advanceTimersByTime(1)
    await nextTick()
    expect(w.find('[data-growth-chip]').exists()).toBe(false)
    w.unmount()
  })

  it('animates only the earned return connector and chips and never replays on plain entry', async () => {
    const pet = usePetStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    useQuestStore().daily = { ...DAILY_PARTWAY, is_target_met: true }
    routeGet(STATUS_AFTER, { ...DAILY_PARTWAY, is_target_met: true })
    const w = mountPage()
    await flushPromises()
    expect(w.findAll('.return-connector')).toHaveLength(1)
    expect(w.find('.return-connector').find('[data-connector]').classes()).toContain('bg-growth')
    vi.advanceTimersByTime(150)
    await nextTick()
    expect(w.find('[data-growth-chip]').classes()).toContain('return-chip')
    const announced = w.find('[data-hub-announcement]').text()
    expect(announced).toContain('+20 máu')
    vi.advanceTimersByTime(1850)
    await nextTick()
    expect(w.find('[data-hub-announcement]').text()).toBe(announced)
    expect(w.find('.return-connector').exists()).toBe(false)
    w.unmount()
    const again = mountPage()
    await flushPromises()
    expect(again.find('.return-connector').exists()).toBe(false)
    expect(again.find('.return-chip').exists()).toBe(false)
  })
  it('has one intentional atomic announcement boundary without live visual ancestors', async () => {
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)
    const w = mountPage()
    await flushPromises()
    const live = w.findAll('[aria-live="polite"]')
    expect(live).toHaveLength(1)
    expect(live[0]!.attributes('data-hub-announcement')).toBeDefined()
    expect(live[0]!.attributes('aria-atomic')).toBe('true')
    expect(live[0]!.text()).toContain('Tưới cho tớ')
    expect(w.findComponent(SpeechBox).find('[data-live-line]').attributes('aria-live')).toBeUndefined()
    expect(w.findAllComponents(RetroPanel).every(panel => panel.attributes('aria-live') === undefined)).toBe(true)
    await w.find('[data-next-task] button').trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-2')
  })

})
