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
})
