import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MapNode from '~/components/retro/MapNode.vue'

describe('MapNode (design §5)', () => {
  it('cleared: torch star, aria-label word', () => {
    const w = mount(MapNode, { props: { day: 3, state: 'cleared', title: 'Ôn từ vựng' } })
    expect(w.classes().join(' ')).toContain('border-torch')
    expect(w.find('[data-glyph="star"]').exists()).toBe(true)
    expect(w.attributes('aria-label')).toBe('Ngày 3: Ôn từ vựng, đã xong')
  })

  it('today: border-growth, aria-current=step', () => {
    const w = mount(MapNode, { props: { day: 9, state: 'today', title: 'Luyện nghe' } })
    expect(w.classes().join(' ')).toContain('border-growth')
    expect(w.attributes('aria-current')).toBe('step')
    expect(w.attributes('aria-label')).toBe('Ngày 9: Luyện nghe, hôm nay')
  })

  it('partial: growth fill on the lower half', () => {
    const w = mount(MapNode, { props: { day: 5, state: 'partial', title: 'x' } })
    expect(w.find('[data-partial-fill]').exists()).toBe(true)
    expect(w.attributes('aria-label')).toContain('một phần')
  })

  it('missed: border-ember, ring glyph', () => {
    const w = mount(MapNode, { props: { day: 6, state: 'missed', title: 'x' } })
    expect(w.classes().join(' ')).toContain('border-ember')
    expect(w.find('[data-glyph="ring"]').exists()).toBe(true)
    expect(w.attributes('aria-label')).toContain('bỏ lỡ')
  })

  it('locked: ink-2, aria-disabled, no select on click', async () => {
    const w = mount(MapNode, { props: { day: 20, state: 'locked', title: 'x' } })
    expect(w.classes().join(' ')).toContain('text-ink-2')
    expect(w.attributes('aria-disabled')).toBe('true')
    expect(w.attributes('aria-label')).toContain('khoá')
    await w.trigger('click')
    expect(w.emitted('select')).toBeUndefined()
  })

  it('emits select(day) for a non-locked tile', async () => {
    const w = mount(MapNode, { props: { day: 9, state: 'today', title: 'x' } })
    await w.trigger('click')
    expect(w.emitted('select')).toEqual([[9]])
  })

  it('expanded sets aria-expanded', () => {
    const w = mount(MapNode, { props: { day: 9, state: 'today', title: 'x', expanded: true } })
    expect(w.attributes('aria-expanded')).toBe('true')
  })
})
