import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MapNode from '~/components/retro/MapNode.vue'

describe('MapNode (design §5)', () => {
  it('cleared: torch star, aria-label word', () => {
    const w = mount(MapNode, { props: { day: 3, state: 'cleared', title: 'Ôn từ vựng' } })
    expect(w.find('[data-tile]').classes().join(' ')).toContain('border-torch')
    expect(w.find('[data-glyph="star"]').exists()).toBe(true)
    expect(w.attributes('aria-label')).toBe('Ngày 3: Ôn từ vựng, đã xong')
  })

  it('today: border-growth, aria-current=step', () => {
    const w = mount(MapNode, { props: { day: 9, state: 'today', title: 'Luyện nghe' } })
    expect(w.find('[data-tile]').classes().join(' ')).toContain('border-growth')
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
    expect(w.find('[data-tile]').classes().join(' ')).toContain('border-ember')
    expect(w.find('[data-glyph="ring"]').exists()).toBe(true)
    expect(w.attributes('aria-label')).toContain('bỏ lỡ')
  })

  it('folded bug (retro-roadmap R6): missed is a hollow ember ring — no ember fill, day number in ink-1', () => {
    const w = mount(MapNode, { props: { day: 6, state: 'missed', title: 'x' } })
    const tile = w.find('[data-tile]')
    expect(tile.classes().join(' ')).not.toContain('bg-ember')
    expect(tile.classes()).toContain('text-ink-1')
    expect(tile.classes()).not.toContain('text-ink-0')
  })

  it('locked: ink-2, aria-disabled, no select on click', async () => {
    const w = mount(MapNode, { props: { day: 20, state: 'locked', title: 'x' } })
    expect(w.find('[data-tile]').classes().join(' ')).toContain('text-ink-2')
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

  it('the button is a 44px wrapper (group relative p-0.5) and carries the torch focus classes (design amend A5)', () => {
    const w = mount(MapNode, { props: { day: 9, state: 'today', title: 'x' } })
    const classes = w.classes()
    expect(classes).toContain('group')
    expect(classes).toContain('relative')
    expect(classes).toContain('p-0.5')
    for (const c of ['focus-visible:outline', 'focus-visible:outline-2', 'focus-visible:outline-offset-2', 'focus-visible:outline-torch', 'focus-visible:ring-0', 'focus-visible:ring-offset-0']) {
      expect(classes).toContain(c)
    }
  })

  it('[data-tile] is 40x40, bg-ground-1, text-ink-0 (text-ink-2 when locked)', () => {
    const today = mount(MapNode, { props: { day: 9, state: 'today', title: 'x' } })
    const tile = today.find('[data-tile]')
    expect(tile.classes()).toContain('h-10')
    expect(tile.classes()).toContain('w-10')
    expect(tile.classes()).toContain('bg-ground-1')
    expect(tile.classes()).toContain('text-ink-0')

    const locked = mount(MapNode, { props: { day: 20, state: 'locked', title: 'x' } })
    const lockedTile = locked.find('[data-tile]')
    expect(lockedTile.classes()).toContain('text-ink-2')
    expect(lockedTile.classes()).not.toContain('text-ink-0')
  })

  it('the pressed inset classes are on [data-tile], absent when locked', () => {
    const today = mount(MapNode, { props: { day: 9, state: 'today', title: 'x' } })
    const tile = today.find('[data-tile]')
    expect(tile.classes()).toContain('group-active:translate-y-[2px]')
    expect(tile.classes()).toContain('group-active:border-line-dim')

    const locked = mount(MapNode, { props: { day: 20, state: 'locked', title: 'x' } })
    const lockedTile = locked.find('[data-tile]')
    expect(lockedTile.classes()).not.toContain('group-active:translate-y-[2px]')
    expect(lockedTile.classes()).not.toContain('group-active:border-line-dim')
  })

  it('the day number is relative z-10, and the partial band is h-2, not h-1/2', () => {
    const w = mount(MapNode, { props: { day: 5, state: 'partial', title: 'x' } })
    const number = w.find('span.font-display')
    expect(number.classes()).toContain('relative')
    expect(number.classes()).toContain('z-10')
    const band = w.find('[data-partial-fill]')
    expect(band.classes()).toContain('h-2')
    expect(band.classes()).not.toContain('h-1/2')
  })
})
