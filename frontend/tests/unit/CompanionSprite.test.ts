import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import { PLANT_STAGES } from '~/utils/plant'
import { tokens } from '~/tailwind.config'

describe('CompanionSprite (design §4)', () => {
  it.each(PLANT_STAGES)('sets data-stage for %s', (stage) => {
    const w = mount(CompanionSprite, { props: { stage, health: 80 } })
    expect(w.attributes('data-stage')).toBe(stage)
  })

  it('falls back to sprout in ink-2 for an unknown stage', () => {
    const w = mount(CompanionSprite, { props: { stage: 'cactus', health: 80 } })
    expect(w.attributes('data-stage')).toBe('sprout')
    expect(w.find('svg').attributes('style')).toContain(`--px-g: ${tokens['ink-2']}`)
  })

  it('carries name, stage and HP in the aria-label', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, name: 'Mầm' } })
    expect(w.attributes('aria-label')).toBe('Mầm, giai đoạn sprout, 80 HP')
  })

  it('rotates for react=down', () => {
    const w = mount(CompanionSprite, { props: { stage: 'wilted', health: 0, react: 'down' } })
    expect(w.attributes('style')).toContain('rotate(90')
  })

  it('runs no retro-* animation class when reduced, and shows a static frame', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'hit', reduced: true } })
    const html = w.html()
    expect(html).not.toMatch(/class="[^"]*\bretro-(breath|hop|shake|flash)\b/)
    expect(w.attributes('data-frame')).toBe('0')
  })

  it('snaps a requested size down to a whole multiple of the 32-grid', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, size: 50 } })
    expect(w.find('svg').attributes('width')).toBe('32')
  })

  it('crops to the face region at a whole ×3 for the 48px portrait', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, size: 48, crop: 'face' } })
    expect(w.find('svg').attributes('viewBox')).toBe('8 16 16 16')
  })

  it('tints ember below 30 HP', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sapling', health: 25 } })
    expect(w.find('svg').attributes('style')).toContain(`--px-g: ${tokens.ember}`)
  })
})
