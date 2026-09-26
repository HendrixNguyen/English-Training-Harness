import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Chest from '~/components/retro/Chest.vue'

const items = [{ icon: '🔥', label: 'Chuỗi 7 ngày' }]

describe('Chest (design §5)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('shows the closed frame until open', () => {
    const w = mount(Chest, { props: { items, open: false } })
    expect(w.find('[data-glyph="chestClosed"]').exists()).toBe(true)
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(false)
    expect(w.find('ul').exists()).toBe(false)
  })

  it('opens after 200ms and emits opened, list is aria-live polite', async () => {
    const w = mount(Chest, { props: { items, open: false } })
    await w.setProps({ open: true })
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(false)
    vi.advanceTimersByTime(200)
    await w.vm.$nextTick()
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(true)
    expect(w.emitted('opened')).toHaveLength(1)
    const list = w.find('ul')
    expect(list.attributes('aria-live')).toBe('polite')
    expect(list.text()).toContain('Chuỗi 7 ngày')
  })

  it('opens at once under reduced motion', async () => {
    const w = mount(Chest, { props: { items, open: false, reduced: true } })
    await w.setProps({ open: true })
    await w.vm.$nextTick()
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(true)
    expect(w.emitted('opened')).toHaveLength(1)
  })

  it('clears its timer on unmount', async () => {
    const w = mount(Chest, { props: { items, open: false } })
    await w.setProps({ open: true })
    w.unmount()
    expect(() => vi.advanceTimersByTime(500)).not.toThrow()
  })
})
