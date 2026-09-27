import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import { usePetStore, REVIVE_STORAGE_KEY } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { useRetroToast } from '~/composables/useRetroToast'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const { default: RevivePage } = await import('~/pages/revive.vue')

const DAILY = { date: '2026-09-23', day_number: 3, total_minutes_required: 30, accumulated_seconds: 0, is_target_met: false, tasks: [] }
const WILTED = { plant_name: 'My Green Buddy', health_points: 0, stage: 'wilted', current_streak: 0, last_practiced_at: '2026-09-20T13:00:00Z' }

/** Routes GET by path so /quests/daily can succeed while /pet/status fails. */
function routeGet(pet: () => Promise<unknown>, daily: unknown = DAILY) {
  api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? pet() : Promise.resolve(daily)))
}

function mountPage() {
  return mount(RevivePage, {
    global: { mocks: { navigateTo: vi.fn() } },
  })
}

function setOnline(value: boolean) {
  Object.defineProperty(window.navigator, 'onLine', { value, configurable: true })
}

describe('/revive (design harness/designs/retro-revive.md) on the kit', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
    setOnline(true)
    useRetroToast().queue.value = []
  })

  it('(1) failed load, nothing cached: unknown state, no band, no down sprite', async () => {
    routeGet(() => Promise.reject(new Error('offline')))
    const w = mountPage()
    await flushPromises()

    expect(w.find('[data-band]').exists()).toBe(false)
    expect(w.find('[data-react="down"]').exists()).toBe(false)
    expect(w.find('[role="status"]').text()).toContain('Không tải được trạng thái của Mầm Non. Chưa biết bạn ấy có gục không.')
    expect(w.text()).toContain('Thử lại')
  })

  it('(2) retry then real wilted data: band, down sprite, HP 0/100, one danger button', async () => {
    let fails = true
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve({ ...WILTED })))
    const w = mountPage()
    await flushPromises()

    fails = false
    await w.find('[role="status"] button').trigger('click')
    await flushPromises()

    expect(w.find('[data-band]').text()).toContain('My Green Buddy đã gục')
    expect(w.find('[data-react="down"]').exists()).toBe(true)
    expect(w.text()).toContain('HP 0/100')
    const buttons = w.findAllComponents(RetroButton).filter(b => b.text().includes('Hồi sinh (15 phút)'))
    expect(buttons).toHaveLength(1)
    expect(buttons[0].props('variant')).toBe('danger')
  })

  it('(3) missed-days line, and cleanly with no last practice', async () => {
    vi.setSystemTime(new Date('2026-09-23T13:00:00Z')) // 3 whole days after last_practiced_at
    try {
      routeGet(() => Promise.resolve({ ...WILTED }))
      const w = mountPage()
      await flushPromises()
      expect(w.text()).toContain('Cậu bỏ tớ 3 ngày liền. Học 15 phút để tớ đứng dậy nhé.')
    } finally {
      vi.useRealTimers()
    }

    setActivePinia(createPinia())
    routeGet(() => Promise.resolve({ ...WILTED, last_practiced_at: null }))
    const w2 = mountPage()
    await flushPromises()
    expect(w2.text()).toContain('Cậu bỏ tớ lâu quá. Học 15 phút để tớ đứng dậy nhé.')
  })

  it('(4) stale wilted data survives a failed reload', async () => {
    let fails = false
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve({ ...WILTED })))
    const w = mountPage()
    await flushPromises()
    expect(w.find('[data-react="down"]').exists()).toBe(true)

    fails = true
    await usePetStore().load()
    await flushPromises()

    expect(w.find('[data-react="down"]').exists()).toBe(true)
    expect(w.find('[role="status"]').exists()).toBe(false)
  })

  it('(5) a failed POST /pet/revive keeps state A and shows the ember strip above the button', async () => {
    routeGet(() => Promise.resolve({ ...WILTED }))
    api.post.mockRejectedValue(new Error('offline'))
    const w = mountPage()
    await flushPromises()

    const start = w.findAllComponents(RetroButton).find(b => b.text().includes('Hồi sinh (15 phút)'))
    if (!start) throw new Error('no start button')
    await start.trigger('click')
    await flushPromises()

    expect(w.find('[data-react="down"]').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('Chưa bắt đầu được nhiệm vụ hồi sinh. Thử lại.')
  })

  it('(6) state B: DayBar counter mm/15, caption, primary "Vào học ngay" navigates home', async () => {
    localStorage.setItem(REVIVE_STORAGE_KEY, JSON.stringify({ date: '2026-09-23', startSeconds: 0 }))
    routeGet(() => Promise.resolve({ ...WILTED }), { ...DAILY, accumulated_seconds: 300 }) // progress 300s -> 10 min left
    const w = mountPage()
    await flushPromises()

    expect(w.text()).toMatch(/\d+\/15/)
    expect(w.text()).toContain('Còn 10 phút học nữa')
    const button = w.findAllComponents(RetroButton).find(b => b.text().includes('Vào học ngay'))
    if (!button) throw new Error('no "Vào học ngay" button')
    expect(button.props('variant')).toBe('primary')
    await button.trigger('click')
    expect(api.post).not.toHaveBeenCalled()
  })

  it('(7) at progress >= 900 the caption and button change, and only then does a tap call revive again', async () => {
    localStorage.setItem(REVIVE_STORAGE_KEY, JSON.stringify({ date: '2026-09-23', startSeconds: 0 }))
    routeGet(() => Promise.resolve({ ...WILTED }), { ...DAILY, accumulated_seconds: 300 })
    api.post.mockResolvedValue({ revival_passed: false, pet_state: { health_points: 0, stage: 'wilted', current_streak: 0 } })
    const w = mountPage()
    await flushPromises()

    // Below target: no post from the "Vào học ngay" tap.
    let button = w.findAllComponents(RetroButton).find(b => b.text().includes('Vào học ngay'))
    if (!button) throw new Error('no button')
    await button.trigger('click')
    expect(api.post).not.toHaveBeenCalled()

    // Cross the target: the caption and label flip, and only now does the tap post.
    routeGet(() => Promise.resolve({ ...WILTED }), { ...DAILY, accumulated_seconds: 1000 })
    await useQuestStore().load()
    await flushPromises()

    expect(w.text()).toContain('Đủ 15 phút rồi')
    button = w.findAllComponents(RetroButton).find(b => b.text().trim() === 'Hồi sinh')
    if (!button) throw new Error('no full-bar button')
    await button.trigger('click')
    await flushPromises()
    expect(api.post).toHaveBeenCalledTimes(1)
  })

  it('(8) revival_passed: true turns the panel growth, plays the get-up once, HP 50/100, "Về trại"', async () => {
    routeGet(() => Promise.resolve({ ...WILTED }))
    api.post.mockResolvedValue({ revival_passed: true, pet_state: { health_points: 50, stage: 'sprout', current_streak: 0 } })
    const w = mountPage()
    await flushPromises()

    const start = w.findAllComponents(RetroButton).find(b => b.text().includes('Hồi sinh (15 phút)'))
    if (!start) throw new Error('no start button')
    await start.trigger('click')
    await flushPromises()

    expect(w.text()).toContain('Tớ dậy rồi. Cảm ơn cậu.')
    expect(w.text()).toContain('HP 50/100')
    const sprite = w.findComponent(CompanionSprite)
    expect(sprite.props('react')).toBe('levelup')
    expect(sprite.props('stage')).toBe('sprout')

    sprite.vm.$emit('reacted', 'levelup')
    await nextTick()
    expect(w.findComponent(CompanionSprite).props('react')).toBe('idle')

    const homeButtons = w.findAllComponents(RetroButton).filter(b => b.text().includes('Về trại'))
    expect(homeButtons).toHaveLength(1)
  })

  it('(9) not wilted: no band, "vẫn khoẻ", "Về trại"', async () => {
    routeGet(() => Promise.resolve({ ...WILTED, health_points: 80, stage: 'sprout' }))
    const w = mountPage()
    await flushPromises()

    expect(w.find('[data-band]').exists()).toBe(false)
    expect(w.text()).toContain('My Green Buddy vẫn khoẻ.')
    expect(w.text()).toContain('Về trại')
  })

  it('(10) loading: StateBlock loading, no band, no button', () => {
    routeGet(() => new Promise(() => {})) // never resolves
    const w = mountPage()

    expect(w.find('[aria-busy="true"]').exists()).toBe(true)
    expect(w.find('[data-band]').exists()).toBe(false)
    expect(w.findComponent(RetroButton).exists()).toBe(false)
  })

  it('(11) offline in state B: exactly one toast, tapping the button does not post', async () => {
    localStorage.setItem(REVIVE_STORAGE_KEY, JSON.stringify({ date: '2026-09-23', startSeconds: 0 }))
    routeGet(() => Promise.resolve({ ...WILTED }), { ...DAILY, accumulated_seconds: 1000 })
    setOnline(false)
    const w = mountPage()
    await flushPromises()

    const button = w.findAllComponents(RetroButton).find(b => b.text().trim() === 'Hồi sinh')
    if (!button) throw new Error('no full-bar button')
    await button.trigger('click')
    await flushPromises()

    expect(api.post).not.toHaveBeenCalled()
    expect(useRetroToast().queue.value).toHaveLength(1)
    expect(w.text()).toContain('Cần mạng để hồi sinh. Tớ vẫn đếm phút cho cậu.')
  })

  it('(12) no v1 component or token class in the rendered HTML', async () => {
    routeGet(() => Promise.resolve({ ...WILTED }))
    const w = mountPage()
    await flushPromises()
    expect(w.html()).not.toMatch(/rounded-card|bg-alert|text-mute|rounded-btn|AppCard|AppButton|PlantSvg|SegmentedProgress/)
  })
})
