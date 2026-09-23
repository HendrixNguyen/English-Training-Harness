import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SegmentedProgress from '~/components/ui/SegmentedProgress.vue'

describe('SegmentedProgress (design §1)', () => {
  it('renders three segments and the wireframe 7.2 label', () => {
    const w = mount(SegmentedProgress, { props: { valueSeconds: 1200, label: 'Tiến độ hôm nay' } })
    const segments = w.findAll('[data-segment]')
    expect(segments).toHaveLength(3)
    expect(segments[0].attributes('style')).toContain('width: 100%')
    expect(segments[1].attributes('style')).toContain('width: 100%')
    expect(segments[2].attributes('style')).toContain('width: 0%')
    expect(w.text()).toContain('20 / 30 phút')
    expect(w.text()).toContain('66%')
    expect(w.find('[role="progressbar"]').attributes('aria-valuenow')).toBe('1200')
    expect(w.find('[role="progressbar"]').attributes('aria-valuemax')).toBe('1800')
  })

  it('marks the target met', () => {
    const w = mount(SegmentedProgress, { props: { valueSeconds: 1800, label: 'x', met: true } })
    expect(w.text()).toContain('Mục tiêu hôm nay đã đạt')
  })

  it('renders a single 15-minute segment for revival', () => {
    const w = mount(SegmentedProgress, { props: { valueSeconds: 450, label: 'Học 15 phút để hồi sinh', segments: 1, segmentSeconds: 900 } })
    expect(w.findAll('[data-segment]')).toHaveLength(1)
    expect(w.text()).toContain('7 / 15 phút')
  })
})
