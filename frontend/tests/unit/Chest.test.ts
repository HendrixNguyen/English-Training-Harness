import { mount } from '@vue/test-utils'
import { ref } from 'vue'
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

describe('Chest reduced default (design amend A4)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('with no reduced prop, falls back to the OS setting: opens at once and emits synchronously', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: ChestMocked } = await import('~/components/retro/Chest.vue')
    const w = mount(ChestMocked, { props: { items, open: false } })
    await w.setProps({ open: true })
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(true)
    expect(w.emitted('opened')).toHaveLength(1)
  })

  it('an explicit reduced=false overrides the OS setting', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: ChestMocked } = await import('~/components/retro/Chest.vue')
    const w = mount(ChestMocked, { props: { items, open: false, reduced: false } })
    await w.setProps({ open: true })
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(false)
    vi.advanceTimersByTime(200)
    await w.vm.$nextTick()
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(true)
  })
})

describe('Chest live OS flip (design A11)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('mounted without reduced, the OS flipping to reduced mid-session opens at once on the next open', async () => {
    const osReduced = ref(false)
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => osReduced }))
    const { default: ChestMocked } = await import('~/components/retro/Chest.vue')
    const w = mount(ChestMocked, { props: { items, open: false } })
    osReduced.value = true
    await w.vm.$nextTick()
    await w.setProps({ open: true })
    expect(w.find('[data-glyph="chestOpen"]').exists()).toBe(true)
    expect(w.emitted('opened')).toHaveLength(1)
  })
})
