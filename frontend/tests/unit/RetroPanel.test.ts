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
})
