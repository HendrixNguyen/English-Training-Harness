import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import GrowthChip from '~/components/plant/GrowthChip.vue'

describe('GrowthChip (design growth-moment §5/§6)', () => {
  it('renders growth-tone text', () => {
    const w = mount(GrowthChip, { props: { text: '+20 máu', tone: 'growth' } })
    expect(w.text()).toBe('+20 máu')
    expect(w.attributes('data-growth-chip')).toBe('growth')
    expect(w.classes()).toContain('text-growth')
  })

  it('renders streak-tone text', () => {
    const w = mount(GrowthChip, { props: { text: '🔥 7 ngày', tone: 'streak' } })
    expect(w.text()).toBe('🔥 7 ngày')
    expect(w.attributes('data-growth-chip')).toBe('streak')
    expect(w.classes()).toContain('text-streak')
  })
})
