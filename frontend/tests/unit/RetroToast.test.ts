import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import RetroToast from '~/components/retro/RetroToast.vue'
import { useRetroToast } from '~/composables/useRetroToast'
import { tokens } from '~/tailwind.config'

describe('RetroToast + useRetroToast (design §5)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    const { queue, dismiss } = useRetroToast()
    while (queue.value.length) dismiss()
  })
  afterEach(() => vi.useRealTimers())

  it('shows one line at a time, the next after 2000ms', async () => {
    const { show } = useRetroToast()
    const w = mount(RetroToast)
    show('a')
    show('b')
    await w.vm.$nextTick()
    expect(w.text()).toContain('a')
    expect(w.text()).not.toContain('b')
    vi.advanceTimersByTime(2000)
    await w.vm.$nextTick()
    expect(w.text()).toContain('b')
  })

  it('is a role=status region', async () => {
    const { show } = useRetroToast()
    const w = mount(RetroToast)
    show('a')
    await w.vm.$nextTick()
    expect(w.find('[role="status"]').exists()).toBe(true)
  })

  it('recolours the outer ring by tone', async () => {
    const { show } = useRetroToast()
    const w = mount(RetroToast)
    show('lỗi rồi', 'ember')
    await w.vm.$nextTick()
    expect(w.html()).toContain(tokens.ember)
  })

  it('renders nothing when the queue is empty', () => {
    const w = mount(RetroToast)
    expect(w.find('[role="status"]').exists()).toBe(false)
  })
})
