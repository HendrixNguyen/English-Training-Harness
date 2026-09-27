import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CountdownTimer from '~/components/learn/CountdownTimer.vue'

/**
 * Design amend A2: `CountdownTimer` paints its own `ground-1` plate and
 * `line-dim` border, so its `ink-0` digits and `ink-1` caption read on any
 * v1 surface (blocker: near-invisible on light-scheme `bg-paper`).
 */
describe('CountdownTimer (design amend A2)', () => {
  it('root is bg-ground-1 border-2 border-line-dim; caption ink-1; digits ink-0', () => {
    const w = mount(CountdownTimer, { props: { remainingSeconds: 125 } })
    const root = w.find('span')
    expect(root.classes()).toContain('bg-ground-1')
    expect(root.classes()).toContain('border-2')
    expect(root.classes()).toContain('border-line-dim')
    const caption = w.findAll('span').find(s => s.text() === 'Thời gian:')
    expect(caption?.classes()).toContain('text-ink-1')
    const digits = w.findAll('span').find(s => s.text() === '02:05')
    expect(digits?.classes()).toContain('text-ink-0')
  })

  it('digits turn text-ember at 0', () => {
    const w = mount(CountdownTimer, { props: { remainingSeconds: 0 } })
    const digits = w.findAll('span').find(s => s.text() === '00:00')
    expect(digits?.classes()).toContain('text-ember')
    expect(digits?.classes()).not.toContain('text-ink-0')
  })

  it('keeps remainingSeconds prop and aria-live="off"', () => {
    const w = mount(CountdownTimer, { props: { remainingSeconds: 42 } })
    expect(w.find('[aria-live="off"]').exists()).toBe(true)
  })
})
