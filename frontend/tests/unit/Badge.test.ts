import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Badge from '~/components/retro/Badge.vue'
import { tokens } from '~/tailwind.config'

describe('Badge (design §5)', () => {
  it('shows the streak count and Vietnamese aria-label', () => {
    const w = mount(Badge, { props: { kind: 'streak', count: 7, earned: true } })
    expect(w.text()).toContain('x7')
    expect(w.attributes('aria-label')).toBe('chuỗi 7 ngày')
  })

  it('labels shield and star kinds', () => {
    expect(mount(Badge, { props: { kind: 'shield' } }).attributes('aria-label')).toBe('khiên')
    expect(mount(Badge, { props: { kind: 'star' } }).attributes('aria-label')).toBe('sao')
  })

  it('an earned badge is drawn in torch; an unearned one is dimmed to line-dim', () => {
    const earned = mount(Badge, { props: { kind: 'streak', count: 3, earned: true } })
    const notEarned = mount(Badge, { props: { kind: 'streak', count: 3, earned: false } })
    expect(earned.find('svg').attributes('style')).toContain(`--px-t: ${tokens.torch}`)
    expect(notEarned.find('svg').attributes('style')).toContain(`--px-t: ${tokens['line-dim']}`)
  })

  it('(H3) pulse adds a stepped outline-pulse class, no scale, using the shared 300ms steps(2) keyframe', () => {
    const w = mount(Badge, { props: { kind: 'streak', count: 7, pulse: true } })
    const classes = w.classes().join(' ')
    expect(classes).toContain('retro-badge-pulse')
    expect(classes).not.toMatch(/scale/)
  })

  it('(H3) no pulse class when pulse is false', () => {
    const w = mount(Badge, { props: { kind: 'streak', count: 7, pulse: false } })
    expect(w.classes().join(' ')).not.toContain('retro-badge-pulse')
  })

  it('(H3) reduced motion drops the pulse class', () => {
    const w = mount(Badge, { props: { kind: 'streak', count: 7, pulse: true, reduced: true } })
    expect(w.classes().join(' ')).not.toContain('retro-badge-pulse')
  })
})
