import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlantSvg from '~/components/plant/PlantSvg.vue'

describe('PlantSvg (design §3)', () => {
  it.each(['seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted'])('draws stage %s', (stage) => {
    const w = mount(PlantSvg, { props: { stage, health: 80 } })
    expect(w.find(`[data-stage="${stage}"]`).exists()).toBe(true)
    expect(w.find('svg').attributes('aria-label')).toContain(stage)
    expect(w.find('svg').attributes('aria-label')).toContain('80')
  })

  it('falls back to the sprout silhouette for an unknown stage', () => {
    const w = mount(PlantSvg, { props: { stage: 'cactus', health: 50 } })
    expect(w.find('[data-stage="sprout"]').exists()).toBe(true)
    expect(w.find('svg').classes()).toContain('text-mute')
  })

  it('tints leaves by health tone and droops when wilted', () => {
    expect(mount(PlantSvg, { props: { stage: 'sapling', health: 20 } }).find('svg').classes()).toContain('text-alert')
    expect(mount(PlantSvg, { props: { stage: 'sapling', health: 45 } }).find('svg').classes()).toContain('text-streak')
    const wilted = mount(PlantSvg, { props: { stage: 'wilted', health: 0 } })
    expect(wilted.find('[data-stage="wilted"]').classes()).toContain('plant-droop')
  })
})
