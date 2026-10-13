import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import RoadmapPage from '~/pages/roadmap.vue'
import AppHeader from '~/components/AppHeader.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { useAuthStore } from '~/stores/auth'
import { useRoadmapStore } from '~/stores/roadmap'
import { ApiError } from '~/utils/apiClient'
const api = { get: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const navigate = vi.fn()
const tasks = ['vocabulary', 'reading', 'practice'].map((task_type, i) => ({ id: `t${i}`, task_type, title: `Actual ${task_type}`, duration_minutes: 10, is_completed: i === 0 ? true : i === 1 ? false : null }))
const data = { roadmap_id: 'r1', title: 'Real meetings journey', cefr_level: 'B1', today: '2026-10-04', day_number: 4, modules: Array.from({ length: 4 }, (_, week) => ({ week: week + 1, title: `Real region ${week + 1}`, focus: `Focus ${week + 1}`, days: Array.from({ length: 7 }, (_, i) => { const n = week * 7 + i + 1; return { day_number: n, date: `2026-10-${String(n).padStart(2, '0')}`, title: `Actual day ${n}`, tasks, minutes_spent: n === 2 ? 12 : n === 3 ? 30 : 0, is_target_met: n === 3 } }) })) }
let w: ReturnType<typeof mount>
function render() { w = mount(RoadmapPage, { global: { components: { AppHeader, StateBlock }, stubs: { NuxtLink: { template: '<a><slot /></a>' } }, mocks: { navigateTo: navigate } } }); return w }
beforeEach(() => { setActivePinia(createPinia()); localStorage.clear(); api.get.mockReset(); navigate.mockReset(); vi.stubGlobal('navigateTo', navigate); vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] }); api.get.mockImplementation((path: string) => Promise.resolve(path.endsWith('/pet/status') ? { plant_name: 'Mầm', health_points: 80, stage: 'sprout', current_streak: 1, last_practiced_at: null } : data)) })
afterEach(() => { w?.unmount(); vi.useRealTimers(); vi.unstubAllGlobals(); vi.restoreAllMocks() })
describe('real retro world map', () => {
  it('renders actual regions, truthful states and today tasks, with inert future nodes', async () => {
    render(); await flushPromises()
    expect(w.text()).toContain('Real meetings journey'); expect(w.text()).toContain('Vùng 4 · Real region 4'); expect(w.text()).toContain('Focus 2')
    expect(w.findAll('[data-glyph="star"]')).toHaveLength(1)
    expect(w.findAll('[data-glyph="ring"]')).toHaveLength(1)
    expect(w.findAll('[data-partial-fill]')).toHaveLength(1)
    expect(w.text()).toContain('Actual vocabulary'); expect(w.text()).toContain('Actual reading'); expect(w.text()).toContain('Actual practice')
    expect(w.text()).toContain('Chưa có dữ liệu hoàn thành')
    expect(w.find('[aria-live="polite"]').exists()).toBe(false)
    await w.get('button[aria-disabled="true"]').trigger('click'); expect(navigate).not.toHaveBeenCalled()
    await w.get('[aria-current="step"]').trigger('click'); expect(navigate).toHaveBeenCalledWith('/')
    await w.get('button[aria-label^="Ngày 2:"]').trigger('click'); expect(w.get('button[aria-label^="Ngày 2:"]').attributes('aria-expanded')).toBe('true'); expect(w.text()).toContain('Actual day 2')
  })
  it('rereads entry, focus, visibility and server-date polling; replacement changes expanded day', async () => {
    render(); await flushPromises(); expect(api.get).toHaveBeenCalledWith('/api/v1/roadmap')
    await w.get('button[aria-label^="Ngày 2:"]').trigger('click')
    expect(w.get('button[aria-label^="Ngày 2:"]').attributes('aria-expanded')).toBe('true')
    window.dispatchEvent(new Event('focus')); await flushPromises()
    expect(w.get('button[aria-label^="Ngày 2:"]').attributes('aria-expanded')).toBe('true')
    document.dispatchEvent(new Event('visibilitychange')); await flushPromises()
    vi.advanceTimersByTime(60000); await flushPromises(); expect(api.get.mock.calls.filter(([path]) => path === '/api/v1/roadmap').length).toBeGreaterThanOrEqual(4)
    api.get.mockResolvedValue({ ...data, roadmap_id: 'r2', today: '2026-10-29', day_number: 29 }); window.dispatchEvent(new Event('focus')); await flushPromises()
    expect(w.find('[aria-current="step"]').exists()).toBe(false); expect(w.find('[aria-label^="Nhiệm vụ ngày"]').exists()).toBe(false)
    w.unmount(); render(); await flushPromises(); expect(useRoadmapStore().outline?.roadmap_id).toBe('r2')
  })
  it('retains same-owner memory under offline failure with an honest offline notice', async () => {
    render(); await flushPromises()
    vi.spyOn(navigator, 'onLine', 'get').mockReturnValue(false)
    api.get.mockRejectedValue(new Error('offline'))
    window.dispatchEvent(new Event('offline')); window.dispatchEvent(new Event('focus')); await flushPromises()
    expect(w.text()).toContain('Real meetings journey')
    expect(w.text()).toContain('Đang ngoại tuyến — bản đồ là bản đã lưu.')
    expect(w.text()).toContain('Không tải được hành trình')
  })
  it('shows retry, empty onboarding and clears former owner synchronously', async () => {
    api.get.mockRejectedValueOnce(new Error('offline')); render(); await flushPromises(); expect(w.text()).toContain('Không tải được hành trình')
    api.get.mockRejectedValueOnce(new ApiError(404, 'no_active_roadmap')); await w.get('button').trigger('click'); await flushPromises(); expect(w.text()).toContain('Cậu chưa có hành trình.')
    await w.get('button').trigger('click'); expect(navigate).toHaveBeenCalledWith('/onboarding')
    api.get.mockResolvedValue(data); window.dispatchEvent(new Event('focus')); await flushPromises(); expect(w.text()).toContain('Real meetings journey')
    api.get.mockImplementation(() => new Promise(() => {})); const auth = useAuthStore(); auth.user = { id: 'new', email: '', full_name: '', cefr_current: null }; await flushPromises(); expect(w.text()).not.toContain('Real meetings journey')
  })
})
