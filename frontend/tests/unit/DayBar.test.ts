import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import DayBar from '~/components/retro/DayBar.vue'

describe('DayBar (design §5, ports SegmentedProgress.test.ts)', () => {
  it('renders three segments and the 20/30 counter', () => {
    const w = mount(DayBar, { props: { valueSeconds: 1200 } })
    expect(w.findAll('[data-segment]')).toHaveLength(3)
    expect(w.text()).toContain('20/30')
  })

  it('marks the target met with the kit caption', () => {
    const w = mount(DayBar, { props: { valueSeconds: 1800, met: true } })
    expect(w.text()).toContain('Phòng hôm nay đã xong')
  })

  it('renders a single 15-minute segment for revival', () => {
    const w = mount(DayBar, { props: { valueSeconds: 450, segments: 1, segmentSeconds: 900, cells: 40 } })
    expect(w.findAll('[data-segment]')).toHaveLength(1)
    expect(w.text()).toContain('7/15')
  })

  it('has one progressbar role on the group with seconds values', () => {
    const w = mount(DayBar, { props: { valueSeconds: 1200 } })
    const group = w.find('[role="progressbar"]')
    expect(group.attributes('aria-valuenow')).toBe('1200')
    expect(group.attributes('aria-valuemax')).toBe('1800')
  })

  it('folded bug: has an accessible name, default "Phòng hôm nay"', () => {
    const w = mount(DayBar, { props: { valueSeconds: 1200 } })
    expect(w.find('[role="progressbar"]').attributes('aria-label')).toBe('Phòng hôm nay')
  })

  it('a custom label prop overrides the accessible name (revive single segment)', () => {
    const w = mount(DayBar, { props: { valueSeconds: 450, segments: 1, segmentSeconds: 900, cells: 40, label: 'Nhiệm vụ hồi sinh' } })
    expect(w.find('[role="progressbar"]').attributes('aria-label')).toBe('Nhiệm vụ hồi sinh')
  })

  it('folded bug: each segment transition steps by the change in lit cells, not the raw cell count', async () => {
    const w = mount(DayBar, { props: { valueSeconds: 0 } })
    await w.setProps({ valueSeconds: 300 })
    const segs = w.findAll('[data-segment]')
    // segment 0 goes from 0/600s to 300/600s: 0 -> 10 lit cells (cells=20) = steps(10)
    expect(segs[0].attributes('style') ?? '').toContain('steps(10)')
  })
})

describe('DayBar reduced default (design amend A4)', () => {
  afterEach(() => {
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('with no reduced prop, falls back to the OS setting: no transition, retro-anim class', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: DayBarMocked } = await import('~/components/retro/DayBar.vue')
    const w = mount(DayBarMocked, { props: { valueSeconds: 1200 } })
    for (const seg of w.findAll('[data-segment]')) {
      expect(seg.attributes('style') ?? '').not.toContain('transition')
      expect(seg.classes()).toContain('retro-anim')
    }
  })

  it('an explicit reduced=false overrides the OS setting', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: DayBarMocked } = await import('~/components/retro/DayBar.vue')
    const w = mount(DayBarMocked, { props: { valueSeconds: 1200, reduced: false } })
    expect(w.find('[data-segment]').attributes('style') ?? '').toContain('transition')
  })
})

describe('DayBar live OS flip (design A11)', () => {
  afterEach(() => {
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('mounted without reduced, the OS flipping to reduced mid-session drops the transition', async () => {
    const osReduced = ref(false)
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => osReduced }))
    const { default: DayBarMocked } = await import('~/components/retro/DayBar.vue')
    const w = mount(DayBarMocked, { props: { valueSeconds: 1200 } })
    expect(w.find('[data-segment]').attributes('style') ?? '').toContain('transition')
    osReduced.value = true
    await w.vm.$nextTick()
    for (const seg of w.findAll('[data-segment]')) {
      expect(seg.attributes('style') ?? '').not.toContain('transition')
    }
  })
})
