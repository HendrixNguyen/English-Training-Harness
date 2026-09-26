import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SpeechBox from '~/components/retro/SpeechBox.vue'

describe('SpeechBox (design §5)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('types the line in over time at 30ms/char', async () => {
    const w = mount(SpeechBox, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80 } })
    const typed = w.find('[data-typed]')
    expect(typed.text()).toBe('')
    vi.advanceTimersByTime(30 * 3)
    await w.vm.$nextTick()
    expect(typed.text()).toBe('Tớ')
    vi.advanceTimersByTime(30 * 20)
    await w.vm.$nextTick()
    expect(typed.text()).toBe('Tớ khát rồi')
  })

  it('reveals the full line and emits settled on click', async () => {
    const w = mount(SpeechBox, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80 } })
    await w.trigger('click')
    expect(w.find('[data-typed]').text()).toBe('Tớ khát rồi')
    expect(w.emitted('settled')).toHaveLength(1)
  })

  it('shows the full line at once under reduced motion', () => {
    const w = mount(SpeechBox, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80, reduced: true } })
    expect(w.find('[data-typed]').text()).toBe('Tớ khát rồi')
  })

  it('always carries the full line in a visually-hidden live region', () => {
    const w = mount(SpeechBox, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80 } })
    const live = w.find('[data-live-line]')
    expect(live.text()).toBe('Tớ khát rồi')
    expect(live.find('[data-typed]').exists()).toBe(false)
  })

  it('restarts typing when the line changes', async () => {
    const w = mount(SpeechBox, { props: { line: 'Aaa', name: 'Mầm', stage: 'sprout', health: 80 } })
    vi.advanceTimersByTime(30 * 3)
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Aaa')
    await w.setProps({ line: 'Bbb' })
    expect(w.find('[data-typed]').text()).toBe('')
    vi.advanceTimersByTime(30 * 3)
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Bbb')
  })
})
