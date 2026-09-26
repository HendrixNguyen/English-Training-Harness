import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ShieldRow from '~/components/plant/ShieldRow.vue'

const slots = (w: ReturnType<typeof mount>) => w.findAll('[data-shield]').map(el => el.attributes('data-shield'))

describe('ShieldRow (design pet-streak-shield §4.1)', () => {
  it.each([[0, ['empty', 'empty']], [1, ['held', 'empty']], [2, ['held', 'held']]])('draws two slots for %d held', (shields, want) => {
    const w = mount(ShieldRow, { props: { shields, lastUsedOn: null, today: '2026-09-25' } })
    expect(slots(w)).toEqual(want)
    expect(w.find('[role="img"]').attributes('aria-label')).toBe(`Khiên: ${shields} trên 2`)
    expect(w.text()).not.toContain('đã đỡ')
  })

  it('marks the first empty slot as spent and shows the caption within seven days', () => {
    const w = mount(ShieldRow, { props: { shields: 1, lastUsedOn: '2026-09-24', today: '2026-09-25' } })
    expect(slots(w)).toEqual(['held', 'spent'])
    expect(w.text()).toContain('Khiên đã đỡ cho ngày 24/09.')
    expect(w.find('[role="img"]').attributes('aria-label')).toContain('24/09')
    expect(slots(mount(ShieldRow, { props: { shields: 0, lastUsedOn: '2026-09-24', today: '2026-09-25' } }))).toEqual(['spent', 'empty'])
  })

  it('shows no spent marker after seven days, and none on a full rack', () => {
    const old = mount(ShieldRow, { props: { shields: 1, lastUsedOn: '2026-09-17', today: '2026-09-25' } })
    expect(slots(old)).toEqual(['held', 'empty'])
    expect(old.text()).not.toContain('đã đỡ')
    const full = mount(ShieldRow, { props: { shields: 2, lastUsedOn: '2026-09-24', today: '2026-09-25' } })
    expect(slots(full)).toEqual(['held', 'held'])
    expect(full.text()).toContain('Khiên đã đỡ cho ngày 24/09.') // the caption still teaches that the hit was taken
  })

  it('clamps out-of-range counts', () => {
    expect(slots(mount(ShieldRow, { props: { shields: 5, lastUsedOn: null, today: '2026-09-25' } }))).toEqual(['held', 'held'])
    expect(slots(mount(ShieldRow, { props: { shields: -1, lastUsedOn: null, today: '2026-09-25' } }))).toEqual(['empty', 'empty'])
  })
})
