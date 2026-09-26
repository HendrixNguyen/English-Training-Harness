import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RoadmapMarker from '~/components/roadmap/RoadmapMarker.vue'
import RoadmapModuleHeader from '~/components/roadmap/RoadmapModuleHeader.vue'
import RoadmapNode from '~/components/roadmap/RoadmapNode.vue'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const navigateTo = vi.fn()
vi.stubGlobal('navigateTo', navigateTo)

const { default: RoadmapPage } = await import('~/pages/roadmap.vue')

const PET_STATUS = { plant_name: 'My Green Buddy', health_points: 80, stage: 'sprout', current_streak: 5, last_practiced_at: '2026-09-21T20:15:00Z' }

/** 4 modules x 7 days x 3 tasks. day_number 9 is today. Days 1 and 4 are
 * met (module 1 = 2/7 met, the rest 0/7); day 2 is missed, day 3 partial. */
function buildOutline() {
  const modules = []
  for (let w = 1; w <= 4; w++) {
    const days = []
    for (let i = 0; i < 7; i++) {
      const n = (w - 1) * 7 + i + 1
      const met = n === 1 || n === 4
      const minutes = met ? 32 : n === 3 ? 12 : n === 9 ? 10 : 0
      days.push({
        day_number: n,
        date: `2026-09-${String(n).padStart(2, '0')}`,
        title: `Day ${n}`,
        tasks: [
          { task_type: 'vocabulary', title: `Vocab ${n}`, duration_minutes: 10 },
          { task_type: 'reading', title: `Reading ${n}`, duration_minutes: 10 },
          { task_type: 'practice', title: `Practice ${n}`, duration_minutes: 10 },
        ],
        minutes_spent: minutes,
        is_target_met: met,
      })
    }
    modules.push({ week: w, title: `Module ${w}`, focus: `Focus ${w}`, days })
  }
  return {
    roadmap_id: 'rm-1',
    title: 'Business English for meetings',
    cefr_level: 'B1',
    created_at: '2026-09-01T00:00:00Z',
    day_number: 9,
    modules,
  }
}

/** Routes GET by path so /pet/status and /api/v1/roadmap resolve independently. */
function routeGet(roadmap: () => Promise<unknown>, pet: () => Promise<unknown> = () => Promise.resolve(PET_STATUS)) {
  api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? pet() : roadmap()))
}

function mountPage() {
  return mount(RoadmapPage, {
    attachTo: document.body,
    global: {
      components: { AppButton, AppCard, StateBlock, RoadmapNode, RoadmapMarker, RoadmapModuleHeader },
      stubs: { AppHeader: true, NuxtLink: { template: '<a><slot /></a>' } },
      mocks: { navigateTo },
    },
  })
}

describe('/roadmap (design harness/designs/roadmap-tree.md)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Element.prototype.scrollIntoView = vi.fn()
    api.get.mockReset()
    navigateTo.mockReset()
  })

  it('renders the roadmap title, level and completed-day count in the header', async () => {
    routeGet(() => Promise.resolve(buildOutline()))
    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('Business English for meetings')
    expect(w.text()).toContain('Trình độ B1')
    expect(w.text()).toContain('Đã hoàn thành 2/28 ngày')
  })

  it('renders four module headers with week labels, titles, focus and per-module counts', async () => {
    routeGet(() => Promise.resolve(buildOutline()))
    const w = mountPage()
    await flushPromises()

    for (let week = 1; week <= 4; week++) {
      expect(w.text()).toContain(`Tuần ${week}`)
      expect(w.text()).toContain(`Module ${week}`)
      expect(w.text()).toContain(`Focus ${week}`)
    }
    expect(w.text()).toContain('2/7')
    expect(w.text()).toContain('0/7')
  })

  it('renders 28 rows with the truthful per-day status', async () => {
    routeGet(() => Promise.resolve(buildOutline()))
    const w = mountPage()
    await flushPromises()

    const rows = w.findAll('li[id^="day-"]')
    expect(rows).toHaveLength(28)
    expect(rows[0].text()).toContain('Đã hoàn thành')
    expect(rows[0].text()).toContain('Day 1')
    expect(rows[1].text()).toContain('Bỏ lỡ')
    expect(rows[2].text()).toContain('12/30 phút')
    expect(rows[9].text()).toContain('Chưa mở khóa')
  })

  it('expands today\'s row by default with its tasks and a Học ngay link, others stay collapsed', async () => {
    routeGet(() => Promise.resolve(buildOutline()))
    const w = mountPage()
    await flushPromises()

    const today = w.find('#day-9')
    expect(today.find('button').attributes('aria-current')).toBe('step')
    expect(today.find('button').attributes('aria-expanded')).toBe('true')
    expect(today.text()).toContain('Vocab 9')
    expect(today.text()).toContain('Reading 9')
    expect(today.text()).toContain('Practice 9')
    expect(today.text()).toContain('10 phút')
    expect(today.text()).toContain('Học ngay')

    const others = w.findAll('li[id^="day-"]').filter(li => li.attributes('id') !== 'day-9')
    for (const row of others) {
      expect(row.find('button').attributes('aria-expanded')).toBe('false')
    }
  })

  it('clicking a row toggles its expansion (locked days expand too)', async () => {
    routeGet(() => Promise.resolve(buildOutline()))
    const w = mountPage()
    await flushPromises()

    const row10 = w.find('#day-10')
    expect(row10.find('button').attributes('aria-expanded')).toBe('false')

    await row10.find('button').trigger('click')
    expect(row10.find('button').attributes('aria-expanded')).toBe('true')
    expect(row10.text()).toContain('Vocab 10')

    await row10.find('button').trigger('click')
    expect(row10.find('button').attributes('aria-expanded')).toBe('false')
  })

  it('scrolls today\'s row into view once, centred', async () => {
    routeGet(() => Promise.resolve(buildOutline()))
    mountPage()
    await flushPromises()

    expect(Element.prototype.scrollIntoView).toHaveBeenCalledTimes(1)
    expect(Element.prototype.scrollIntoView).toHaveBeenCalledWith({ block: 'center' })
  })

  it('shows the empty state and creates a roadmap on 404 no_active_roadmap', async () => {
    routeGet(() => Promise.reject(new ApiError(404, 'no_active_roadmap')))
    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('Bạn chưa có lộ trình học.')
    const button = w.findAll('button').find(b => b.text().includes('Tạo lộ trình 28 ngày'))
    if (!button) throw new Error('no create-roadmap button')
    await button.trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/onboarding')
  })

  it('shows the error state on a network failure and retries', async () => {
    let fails = true
    routeGet(() => (fails ? Promise.reject(new TypeError('offline')) : Promise.resolve(buildOutline())))
    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('Không tải được lộ trình.')
    fails = false
    const retry = w.findAll('button').find(b => b.text().includes('Thử lại'))
    if (!retry) throw new Error('no retry button')
    await retry.trigger('click')
    await flushPromises()

    expect(api.get).toHaveBeenCalledWith('/api/v1/roadmap')
    expect(w.text()).toContain('Business English for meetings')
  })
})
