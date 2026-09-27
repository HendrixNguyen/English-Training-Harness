import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import HpBar from '~/components/retro/HpBar.vue'
import { tokens } from '~/tailwind.config'

function fillWidthPx(html: string): number {
  const m = html.match(/width:\s*(\d+)px/)
  if (!m) throw new Error(`no width found in: ${html}`)
  return Number(m[1])
}

describe('HpBar (design §5)', () => {
  it('snaps the fill width to 4px cells for every value 0-100', () => {
    for (let value = 0; value <= 100; value++) {
      const w = mount(HpBar, { props: { value } })
      const fill = w.find('[data-fill]')
      const px = fillWidthPx(fill.attributes('style') ?? '')
      expect(px % 4).toBe(0)
      expect(px).toBeLessThanOrEqual(25 * 4)
    }
  })

  it('tones ember below 30, torch 30-59, growth from 60', () => {
    const colorOf = (value: number) => {
      const w = mount(HpBar, { props: { value } })
      return w.find('[data-fill]').attributes('style') ?? ''
    }
    expect(colorOf(29)).toContain(tokens.ember)
    expect(colorOf(30)).toContain(tokens.torch)
    expect(colorOf(59)).toContain(tokens.torch)
    expect(colorOf(60)).toContain(tokens.growth)
  })

  it('is a meter with aria-valuenow', () => {
    const w = mount(HpBar, { props: { value: 42 } })
    const meter = w.find('[role="meter"]')
    expect(meter.attributes('aria-valuenow')).toBe('42')
    expect(meter.attributes('aria-valuemin')).toBe('0')
    expect(meter.attributes('aria-valuemax')).toBe('100')
  })

  it('has no transition under reduced motion', () => {
    const w = mount(HpBar, { props: { value: 50, reduced: true } })
    expect(w.find('[data-fill]').attributes('style') ?? '').not.toContain('transition')
  })

  it('folded bug: clamps the label and aria-valuenow to 0..max, never the raw value', () => {
    const low = mount(HpBar, { props: { value: -5 } })
    expect(low.text()).toContain('HP 0/100')
    expect(low.find('[role="meter"]').attributes('aria-valuenow')).toBe('0')

    const high = mount(HpBar, { props: { value: 130 } })
    expect(high.text()).toContain('HP 100/100')
    expect(high.find('[role="meter"]').attributes('aria-valuenow')).toBe('100')
  })

  it('folded bug: the fill transition steps by the change in lit cells, not the raw cell count', async () => {
    const w = mount(HpBar, { props: { value: 40 } })
    await w.setProps({ value: 44 })
    expect(w.find('[data-fill]').attributes('style') ?? '').toContain('steps(1)')

    const w2 = mount(HpBar, { props: { value: 0 } })
    await w2.setProps({ value: 100 })
    expect(w2.find('[data-fill]').attributes('style') ?? '').toContain('steps(25)')
  })
})

describe('HpBar reduced default (design amend A4)', () => {
  afterEach(() => {
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('with no reduced prop, falls back to the OS setting: no transition, retro-anim class', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: HpBarMocked } = await import('~/components/retro/HpBar.vue')
    const w = mount(HpBarMocked, { props: { value: 50 } })
    const fill = w.find('[data-fill]')
    expect(fill.attributes('style') ?? '').not.toContain('transition')
    expect(fill.classes()).toContain('retro-anim')
  })

  it('an explicit reduced=false overrides the OS setting', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: HpBarMocked } = await import('~/components/retro/HpBar.vue')
    const w = mount(HpBarMocked, { props: { value: 50, reduced: false } })
    expect(w.find('[data-fill]').attributes('style') ?? '').toContain('transition')
  })
})

describe('HpBar live OS flip (design A11)', () => {
  afterEach(() => {
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('mounted without reduced, the OS flipping to reduced mid-session drops the transition', async () => {
    const osReduced = ref(false)
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => osReduced }))
    const { default: HpBarMocked } = await import('~/components/retro/HpBar.vue')
    const w = mount(HpBarMocked, { props: { value: 50 } })
    expect(w.find('[data-fill]').attributes('style') ?? '').toContain('transition')
    osReduced.value = true
    await w.vm.$nextTick()
    expect(w.find('[data-fill]').attributes('style') ?? '').not.toContain('transition')
  })
})
