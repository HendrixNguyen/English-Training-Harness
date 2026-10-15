import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import HpBar from '~/components/retro/HpBar.vue'
import DayBar from '~/components/retro/DayBar.vue'
import { useQuestStore } from '~/stores/quest'
import { useAuthStore } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
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
  api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? pet() : Promise.resolve({ ...DAILY })))
}

function mountPage() {
  return mount(RevivePage, {
    global: {
      components: { AppButton, AppCard, CompanionSprite, RetroPanel, RetroButton, HpBar, DayBar, SegmentedProgress, StateBlock },
      stubs: { NuxtLink: { template: '<a><slot /></a>' } },
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
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve({ ...WILTED })))
    const w = mountPage()
    await flushPromises()
    expect(w.text()).toContain('Thử lại')

    fails = false
    await w.find('[role="status"] button').trigger('click')
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('héo rũ')
    expect(w.find('[role="alert"]').text()).toContain('My Green Buddy đang bị héo rũ')
    expect(w.text()).toContain('Hồi sinh (15 phút)')
  })

  it('shows the wilted UI on real wilted data', async () => {
    routeGet(() => Promise.resolve({ ...WILTED }))
    const w = mountPage()
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="alert"]').text()).toContain('héo rũ')
    expect(w.find('[role="status"]').exists()).toBe(false)
  })

  it('renders the missed-days sentence with its space, and cleanly with no last practice', async () => {
    vi.setSystemTime(new Date('2026-09-23T13:00:00Z')) // 3 whole days after last_practiced_at; Date only, timers untouched
    try {
      routeGet(() => Promise.resolve({ ...WILTED }))
      const w = mountPage()
      await flushPromises()
      expect(w.text()).toContain('Cậu bỏ tớ 3 ngày liền. Học 15 phút')
      expect(w.text()).not.toContain('bỏ học3')
    } finally {
      vi.useRealTimers()
    }

    setActivePinia(createPinia())
    routeGet(() => Promise.resolve({ ...WILTED, last_practiced_at: null }))
    const w2 = mountPage()
    await flushPromises()
    expect(w2.text()).toContain('Cậu bỏ tớ một thời gian rồi. Học 15 phút')
    expect(w2.text()).not.toContain('bỏ học  ')
  })

  it('keeps the last-known wilted state when a later reload fails — stale data wins over the error card', async () => {
    let fails = false
    routeGet(() => (fails ? Promise.reject(new Error('offline')) : Promise.resolve({ ...WILTED })))
    const w = mountPage()
    await flushPromises()
    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)

    fails = true
    await usePetStore().load()
    await flushPromises()

    expect(w.find('[data-stage="wilted"]').exists()).toBe(true)
    expect(w.find('[role="status"]').exists()).toBe(false)
    expect(w.text()).toContain('Hồi sinh (15 phút)')
  })

  it('a failed POST /pet/revive keeps the wilted screen and shows only the inline message', async () => {
    routeGet(() => Promise.resolve({ ...WILTED }))
    api.post.mockRejectedValue(new Error('offline'))
    const w = mountPage()
    await flushPromises()

    const start = w.findAll('button').find(b => b.text().includes('Hồi sinh (15 phút)'))
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


describe('real revive challenge lifecycle', () => {
 beforeEach(() => { setActivePinia(createPinia()); localStorage.clear(); api.get.mockReset(); api.post.mockReset(); routeGet(() => Promise.resolve({ ...WILTED })); vi.stubGlobal('navigateTo', vi.fn()) })
 it('starts dated measured challenge, resumes and permits second check only after 900 seconds', async () => {
  api.post.mockResolvedValue({ revival_passed: false, pet_state: { health_points: 0, stage: 'wilted', current_streak: 0 } })
  const w = mountPage(); await flushPromises(); await w.findAll('button').find(b => b.text().includes('Hồi sinh'))!.trigger('click'); await flushPromises()
  expect(usePetStore().challenge).toEqual({ date: DAILY.date, startSeconds: 600 }); expect(w.text()).toContain('0/15'); expect(w.text()).toContain('Vào học ngay'); expect(api.post).toHaveBeenCalledTimes(1)
  w.unmount(); const resumed = mountPage(); await flushPromises(); expect(resumed.text()).toContain('Vào học ngay')
  useQuestStore().daily!.accumulated_seconds = 1499; await flushPromises(); expect(resumed.text()).toContain('Vào học ngay')
  useQuestStore().daily!.accumulated_seconds = 1500; await flushPromises(); expect(resumed.text()).not.toContain('Vào học ngay')
  api.post.mockResolvedValue({ revival_passed: true, pet_state: { health_points: 37, stage: 'sapling', current_streak: 2 } }); await resumed.findAll('button').find(b => b.text() === 'Hồi sinh')!.trigger('click'); await flushPromises()
  expect(resumed.text()).toContain('37/100'); expect(resumed.find('[data-stage="sapling"]').exists()).toBe(true); expect(resumed.text()).toContain('Về trại'); expect(usePetStore().challenge).toBeNull()
 })
 it('healthy state renders actual HP and never a revive action', async () => {
  routeGet(() => Promise.resolve({ ...WILTED, health_points: 61, stage: 'sapling' })); const w = mountPage(); await flushPromises(); expect(w.text()).toContain('61/100'); expect(w.text()).toContain('vẫn khỏe'); expect(w.findAll('button').map(b => b.text())).toEqual(['Về trại'])
 })
 it('pet_not_wilted is healthy without invented HP', async () => {
  api.post.mockRejectedValue(new ApiError(409, 'pet_not_wilted')); const w = mountPage(); await flushPromises(); await w.find('button').trigger('click'); await flushPromises(); expect(w.text()).toContain('vẫn khỏe'); expect(w.text()).not.toContain('100/100')
 })
 it('offline cannot POST and does not claim a queued save', async () => {
  const w = mountPage(); await flushPromises(); const online = vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false); window.dispatchEvent(new Event('offline')); await flushPromises(); expect(w.find('button').attributes('disabled')).toBeDefined(); expect(w.text()).toContain('Cần mạng'); expect(api.post).not.toHaveBeenCalled(); online.mockRestore(); w.unmount()
 })
 it('logout while revive is pending cannot show a passed result', async () => {
  let resolve!: (value: unknown) => void; api.post.mockImplementation(() => new Promise(r => { resolve = r })); const w = mountPage(); await flushPromises(); await w.find('button').trigger('click'); await useAuthStore().signOut(); resolve({ revival_passed: true, pet_state: { health_points: 50, stage: 'sprout', current_streak: 0 } }); await flushPromises(); expect(w.text()).not.toContain('đã hồi sinh'); expect(usePetStore().status).toBeNull()
 })
})
