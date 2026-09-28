import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import { tokens } from '~/tailwind.config'

describe('RetroPanel (design §5)', () => {
  it('renders the speaker tab with aria-labelledby and aria-live', () => {
    const w = mount(RetroPanel, { props: { speaker: 'Mầm' }, slots: { default: 'Xin chào' } })
    const h2 = w.find('h2')
    expect(h2.exists()).toBe(true)
    expect(h2.text()).toBe('Mầm')
    const section = w.find('section')
    expect(section.attributes('aria-labelledby')).toBe(h2.attributes('id'))
    expect(section.attributes('aria-live')).toBe('polite')
  })

  it('has no aria-live when there is no speaker', () => {
    const w = mount(RetroPanel, { slots: { default: 'x' } })
    expect(w.find('section').attributes('aria-live')).toBeUndefined()
  })

  it('recolours the outer ring for tone=ember', () => {
    const w = mount(RetroPanel, { props: { tone: 'ember' }, slots: { default: 'x' } })
    const style = w.find('section').attributes('style') ?? ''
    expect(style).toContain(tokens.ember)
  })

  it('renders the portrait slot to the left of the default slot', () => {
    const w = mount(RetroPanel, {
      slots: { portrait: '<div class="the-portrait" />', default: '<p class="the-body">hi</p>' },
    })
    const html = w.html()
    expect(html.indexOf('the-portrait')).toBeLessThan(html.indexOf('the-body'))
  })

  it.each(['plain', 'ember'] as const)('tone=%s always sets its own ground-1, text-ink-0, border-2 and a ring (design amend A1/A2)', (tone) => {
    const w = mount(RetroPanel, { props: { tone }, slots: { default: 'x' } })
    const section = w.find('section')
    expect(section.classes()).toContain('bg-ground-1')
    expect(section.classes()).toContain('text-ink-0')
    expect(section.classes()).toContain('border-2')
    expect(section.attributes('style') ?? '').toContain('box-shadow')
  })

  it('band swaps p-4 for p-2 w-full but keeps the fill, line and ring (design amend A2 clarification)', () => {
    const w = mount(RetroPanel, { props: { band: true }, slots: { default: 'x' } })
    const section = w.find('section')
    expect(section.classes()).toContain('bg-ground-1')
    expect(section.classes()).toContain('border-2')
    expect(section.classes()).toContain('p-2')
    expect(section.classes()).toContain('w-full')
    expect(section.classes()).not.toContain('p-4')
    expect(section.attributes('style') ?? '').toContain('box-shadow')
  })
})
