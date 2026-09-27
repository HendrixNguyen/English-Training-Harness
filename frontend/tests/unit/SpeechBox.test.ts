import { mount } from '@vue/test-utils'
import { ref } from 'vue'
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

describe('SpeechBox reduced default (design amend A4)', () => {
  afterEach(() => {
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('with no reduced prop, falls back to the OS setting: shows the full line at mount', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: SpeechBoxMocked } = await import('~/components/retro/SpeechBox.vue')
    const w = mount(SpeechBoxMocked, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80 } })
    expect(w.find('[data-typed]').text()).toBe('Tớ khát rồi')
  })

  it('an explicit reduced=false overrides the OS setting', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: SpeechBoxMocked } = await import('~/components/retro/SpeechBox.vue')
    const w = mount(SpeechBoxMocked, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80, reduced: false } })
    expect(w.find('[data-typed]').text()).toBe('')
  })
})

describe('SpeechBox live OS flip (design A11)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('flipping the OS setting mid-line reveals the whole line and emits settled exactly once', async () => {
    const osReduced = ref(false)
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => osReduced }))
    const { default: SpeechBoxMocked } = await import('~/components/retro/SpeechBox.vue')
    const w = mount(SpeechBoxMocked, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80 } })
    vi.advanceTimersByTime(30 * 2)
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Tớ')
    osReduced.value = true
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Tớ khát rồi')
    expect(w.emitted('settled')).toHaveLength(1)
  })

  it('flipping the OS setting after the line has already settled does not emit settled again', async () => {
    const osReduced = ref(false)
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => osReduced }))
    const { default: SpeechBoxMocked } = await import('~/components/retro/SpeechBox.vue')
    const w = mount(SpeechBoxMocked, { props: { line: 'Tớ khát rồi', name: 'Mầm', stage: 'sprout', health: 80 } })
    vi.advanceTimersByTime(30 * 'Tớ khát rồi'.length)
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Tớ khát rồi')
    expect(w.emitted('settled')).toHaveLength(1)
    osReduced.value = true
    await w.vm.$nextTick()
    expect(w.emitted('settled')).toHaveLength(1)
  })

  it('flipping the OS setting true then false after the reveal does not retype; a new line types normally', async () => {
    const osReduced = ref(false)
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => osReduced }))
    const { default: SpeechBoxMocked } = await import('~/components/retro/SpeechBox.vue')
    const w = mount(SpeechBoxMocked, { props: { line: 'Aaa', name: 'Mầm', stage: 'sprout', health: 80 } })
    vi.advanceTimersByTime(30)
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('A')
    osReduced.value = true
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Aaa')
    osReduced.value = false
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Aaa')
    await w.setProps({ line: 'Bbb' })
    expect(w.find('[data-typed]').text()).toBe('')
    vi.advanceTimersByTime(30 * 3)
    await w.vm.$nextTick()
    expect(w.find('[data-typed]').text()).toBe('Bbb')
  })
})
