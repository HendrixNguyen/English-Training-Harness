import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ContentViewer from '~/components/learn/ContentViewer.vue'
import CountdownTimer from '~/components/learn/CountdownTimer.vue'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import StateBlock from '~/components/ui/StateBlock.vue'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))

const navigateTo = vi.fn()
vi.stubGlobal('useRoute', () => ({ params: { id: 'ex-2' } }))
vi.stubGlobal('navigateTo', navigateTo)

const { default: LearnPage } = await import('~/pages/learn/[id].vue')

const T0 = Date.parse('2026-09-23T13:00:00Z')

const daily = {
  date: '2026-09-23',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 0,
  is_target_met: false,
  tasks: [
    { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu', duration_minutes: 10, is_completed: false, content_json: {} },
  ],
}

function mountPage() {
  return mount(LearnPage, {
    global: {
      components: { AppButton, AppCard, ContentViewer, CountdownTimer, StateBlock },
      mocks: { navigateTo },
    },
  })
}

describe('/learn/:id (wireframe 7.3)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    api.get.mockReset()
    api.post.mockReset()
    navigateTo.mockReset()
  })

  afterEach(() => {
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
})
