import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import AppHeader from '~/components/AppHeader.vue'
import GrowthChip from '~/components/plant/GrowthChip.vue'
import PlantSvg from '~/components/plant/PlantSvg.vue'
import SpeechBubble from '~/components/plant/SpeechBubble.vue'
import QuestRow from '~/components/quest/QuestRow.vue'
import HealthBar from '~/components/ui/HealthBar.vue'
import SegmentedProgress from '~/components/ui/SegmentedProgress.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'

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

function mountPage() {
  return mount(IndexPage, {
    global: {
      components: { AppHeader, AppCard, StateBlock, PlantSvg, HealthBar, SpeechBubble, SegmentedProgress, QuestRow, GrowthChip, AppButton },
      stubs: { NuxtLink: { template: '<a><slot /></a>' } },
      mocks: { navigateTo: vi.fn() },
    },
  })
}

describe('/ (wireframe 7.2) growth moment', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    api.get.mockReset()
    api.post.mockReset()
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
  })

  afterEach(() => {
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
    expect(w.find('[data-plant]').classes()).not.toContain('plant-grow')

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
    expect(w.find('[data-plant]').classes()).toContain('plant-grow')

    vi.advanceTimersByTime(1500)
    await nextTick()
    expect(w.find('[data-plant]').classes()).not.toContain('plant-grow')
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
    expect(w.text()).toContain('Streak: 6 ngày')
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
    expect(w.find('[data-plant]').classes()).not.toContain('plant-grow')
    expect(w.find('[data-streak-chip]').classes()).not.toContain('streak-pulse')
    expect(w.text()).toContain('Còn 20 phút nữa thôi!')
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
    expect(w.find('[data-plant]').classes()).not.toContain('plant-grow')
    expect(w.find('[data-streak-chip]').classes()).not.toContain('streak-pulse')
    expect(w.text()).toContain('Còn 20 phút nữa thôi!')
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
})
