import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PlantSvg from '~/components/plant/PlantSvg.vue'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import SegmentedProgress from '~/components/ui/SegmentedProgress.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { usePetStore } from '~/stores/pet'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { default: RevivePage } = await import('~/pages/revive.vue')

const DAILY = { date: '2026-09-23', day_number: 3, total_minutes_required: 30, accumulated_seconds: 600, is_target_met: false, tasks: [] }
const WILTED = { plant_name: 'My Green Buddy', health_points: 0, stage: 'wilted', current_streak: 0, last_practiced_at: '2026-09-20T13:00:00Z' }

/** Routes GET by path so /quests/daily can succeed while /pet/status fails. */
function routeGet(pet: () => Promise<unknown>) {
  api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? pet() : Promise.resolve(DAILY)))
}

function mountPage() {
  return mount(RevivePage, {
    global: {
      components: { AppButton, AppCard, PlantSvg, SegmentedProgress, StateBlock },
      mocks: { navigateTo: vi.fn() },
    },
  })
}

describe('/revive (wireframe 7.5) when GET /pet/status fails', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
  })

  it('renders an error state with retry — never the wilted alarm', async () => {
    routeGet(() => Promise.reject(new Error('offline')))
    const w = mountPage()
    await flushPromises()

    expect(w.text()).not.toContain('héo rũ')
    expect(w.find('[role="alert"]').exists()).toBe(false)
    expect(w.find('[data-stage="wilted"]').exists()).toBe(false)
    expect(w.text()).toContain('Thử lại')
    expect(w.find('[role="status"]').text()).toContain('Không tải được')
  })

  it('retry reloads the status and then shows the real wilted state', async () => {
    let fails = true
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve(WILTED)))
    const w = mountPage()
    await flushPromises()
    expect(w.text()).toContain('Thử lại')

    fails = false
    await w.find('[role="status"] button').trigger('click')
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('héo rũ')
    expect(w.text()).toContain('Cứu cây ngay')
  })

  it('shows the wilted UI on real wilted data', async () => {
    routeGet(() => Promise.resolve(WILTED))
    const w = mountPage()
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('héo rũ')
    expect(w.find('[role="status"]').exists()).toBe(false)
  })

  it('renders the missed-days sentence with its space, and cleanly with no last practice', async () => {
    vi.setSystemTime(new Date('2026-09-23T13:00:00Z')) // 3 whole days after last_practiced_at; Date only, timers untouched
    try {
      routeGet(() => Promise.resolve(WILTED))
      const w = mountPage()
      await flushPromises()
      expect(w.text()).toContain('Bạn đã bỏ học 3 ngày liên tiếp. Hãy hoàn thành')
      expect(w.text()).not.toContain('bỏ học3')
    } finally {
      vi.useRealTimers()
    }

    setActivePinia(createPinia())
    routeGet(() => Promise.resolve({ ...WILTED, last_practiced_at: null }))
    const w2 = mountPage()
    await flushPromises()
    expect(w2.text()).toContain('Bạn đã bỏ học. Hãy hoàn thành')
    expect(w2.text()).not.toContain('bỏ học  ')
  })

  it('keeps the last-known wilted state when a later reload fails — stale data wins over the error card', async () => {
    let fails = false
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve(WILTED)))
    const w = mountPage()
    await flushPromises()
    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)

    fails = true
    await usePetStore().load()
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="status"]').exists()).toBe(false)
    expect(w.text()).toContain('Cứu cây ngay')
  })

  it('a failed POST /pet/revive keeps the wilted screen and shows only the inline message', async () => {
    routeGet(() => Promise.resolve(WILTED))
    api.post.mockRejectedValue(new Error('offline'))
    const w = mountPage()
    await flushPromises()

    const start = w.findAll('button').find(b => b.text().includes('Cứu cây ngay'))
    if (!start) throw new Error('no revive button')
    await start.trigger('click')
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="status"]').exists()).toBe(false)
    const alerts = w.findAll('[role="alert"]').map(a => a.text())
    expect(alerts.some(t => t.includes('héo rũ'))).toBe(true)
    expect(alerts.some(t => t.includes('Không bắt đầu được thử thách'))).toBe(true)
  })
})
