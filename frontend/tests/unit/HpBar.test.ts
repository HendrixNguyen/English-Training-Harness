import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
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
})
