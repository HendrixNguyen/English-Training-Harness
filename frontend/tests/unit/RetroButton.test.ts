import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import RetroButton from '~/components/retro/RetroButton.vue'

const sfcSource = readFileSync(resolve(process.cwd(), 'components/retro/RetroButton.vue'), 'utf-8')

describe('RetroButton (design §5)', () => {
  it('is 48px tall', () => {
    const w = mount(RetroButton, { slots: { default: 'Vào' } })
    expect(w.classes()).toContain('h-12')
  })

  it.each(['primary', 'secondary', 'danger'] as const)('has a distinct class set for variant=%s', (variant) => {
    const w = mount(RetroButton, { props: { variant }, slots: { default: 'x' } })
    expect(w.classes().join(' ')).toMatch(new RegExp(variant === 'primary' ? 'bg-growth' : variant === 'secondary' ? 'bg-ground-2' : 'bg-ember'))
  })

  it('loading keeps the label in the DOM at opacity-0 and sets aria-busy + disabled', () => {
    const w = mount(RetroButton, { props: { loading: true }, slots: { default: 'Vào nhiệm vụ' } })
    expect(w.attributes('aria-busy')).toBe('true')
    expect(w.attributes('disabled')).toBeDefined()
    const label = w.find('[data-label]')
    expect(label.text()).toBe('Vào nhiệm vụ')
    expect(label.classes()).toContain('opacity-0')
  })

  it('disabled sets aria-disabled and stays in the DOM', () => {
    const w = mount(RetroButton, { props: { disabled: true }, slots: { default: 'x' } })
    expect(w.attributes('aria-disabled')).toBe('true')
    expect(w.exists()).toBe(true)
  })

  it('block is full width', () => {
    const w = mount(RetroButton, { props: { block: true }, slots: { default: 'x' } })
    expect(w.classes()).toContain('w-full')
  })

  it('folded bug: disabled and loading both drop the 4px depth shadow', () => {
    expect(sfcSource).toContain('.rb:disabled')
    expect(sfcSource).toContain(".rb[aria-busy='true']")
    expect(sfcSource).toMatch(/\.rb:disabled,\s*\.rb\[aria-busy='true'\]\s*\{\s*box-shadow:\s*none/)

    const disabled = mount(RetroButton, { props: { disabled: true }, slots: { default: 'x' } })
    expect(disabled.attributes('disabled')).toBeDefined()

    const loading = mount(RetroButton, { props: { loading: true }, slots: { default: 'x' } })
    expect(loading.attributes('aria-busy')).toBe('true')
    expect(loading.attributes('disabled')).toBeDefined()
  })
})
