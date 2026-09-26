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

  it('grow adds the one-shot breath class to the plant, not the pot', () => {
    const grown = mount(PlantSvg, { props: { stage: 'sapling', health: 80, grow: true } })
    expect(grown.find('[data-plant]').classes()).toContain('plant-grow')
    const notGrown = mount(PlantSvg, { props: { stage: 'sapling', health: 80 } })
    expect(notGrown.find('[data-plant]').classes()).not.toContain('plant-grow')
    const pot = grown.find('path')
    expect(pot.element.closest('[data-plant]')).toBeNull()
  })

  it('a stage change renders the new stage group', async () => {
    const w = mount(PlantSvg, { props: { stage: 'sprout', health: 80 } })
    await w.setProps({ stage: 'sapling' })
    expect(w.find('[data-stage="sapling"]').exists()).toBe(true)
    expect(w.find('[data-stage="sprout"]').exists()).toBe(false)
  })
})
