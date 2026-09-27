import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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

  it('down puts the rotation on the drawing, not the wrapper (folded bug)', () => {
    const w = mount(CompanionSprite, { props: { stage: 'wilted', health: 0, react: 'down' } })
    expect(w.find('g').attributes('transform')).toBe('rotate(90 16 16) translate(0 7)')
    expect(w.attributes('style') ?? '').not.toContain('rotate(')
  })

  it('runs no retro-* animation class when reduced', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'hit', reduced: true } })
    const html = w.html()
    expect(html).not.toMatch(/class="[^"]*\bretro-(breath|hop|shake|flash)\b/)
  })

  it('idle under reduced motion is the static base frame (data-frame 0)', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'idle', reduced: true } })
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

  it('the non-reduced animationend path still emits reacted', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'hit' } })
    w.trigger('animationend')
    expect(w.emitted('reacted')).toEqual([['hit']])
  })
})

describe('CompanionSprite --px unit (folded bug companionsprite-motion-is-off-the-pixel-grid)', () => {
  it.each([
    [128, 'full', '4'],
    [64, 'full', '2'],
    [32, 'full', '1'],
  ] as const)('size=%i crop=%s sets --px to %s', (size, crop, px) => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, size, crop } })
    expect(w.attributes('style')).toContain(`--px: ${px}`)
  })

  it('size=48 crop=face sets --px to 3 (a whole ×3 of the 16-grid)', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, size: 48, crop: 'face' } })
    expect(w.attributes('style')).toContain('--px: 3')
  })
})

describe('CompanionSprite levelup palette swap, non-reduced (folded bug)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('steps torch -> ink-0 -> base over 300ms, emits reacted once, never uses retro-flash', async () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'levelup' } })
    expect(w.find('svg').attributes('style')).toContain(`--px-g: ${tokens.torch}`)

    vi.advanceTimersByTime(100)
    await w.vm.$nextTick()
    expect(w.find('svg').attributes('style')).toContain(`--px-g: ${tokens['ink-0']}`)

    vi.advanceTimersByTime(100)
    await w.vm.$nextTick()
    expect(w.find('svg').attributes('style')).toContain(`--px-g: ${tokens.growth}`)
    expect(w.emitted('reacted')).toBeUndefined()

    vi.advanceTimersByTime(100)
    await w.vm.$nextTick()
    expect(w.emitted('reacted')).toEqual([['levelup']])

    expect(w.html()).not.toMatch(/\bretro-flash\b/)
  })

  it('unmounting mid-sequence leaves no pending timer', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'levelup' } })
    w.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})

describe('CompanionSprite static reactions under reduced motion (design amend A4)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('hit: data-frame 1 and translateY during the hold; no emit at 299ms; emits and returns to 0 at 300ms', async () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'hit', reduced: true } })
    expect(w.attributes('data-frame')).toBe('1')
    expect(w.attributes('style')).toContain('translateY(calc(var(--px) * -2px))')
    vi.advanceTimersByTime(299)
    expect(w.emitted('reacted')).toBeUndefined()
    vi.advanceTimersByTime(1)
    await w.vm.$nextTick()
    expect(w.emitted('reacted')).toEqual([['hit']])
    expect(w.attributes('data-frame')).toBe('0')
  })

  it('levelup: --px-g is torch during the hold, and emits at 300ms', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'levelup', reduced: true } })
    expect(w.find('svg').attributes('style')).toContain(`--px-g: ${tokens.torch}`)
    vi.advanceTimersByTime(300)
    expect(w.emitted('reacted')).toEqual([['levelup']])
  })

  it('miss: data-frame stays 0, and emits at 300ms', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'miss', reduced: true } })
    expect(w.attributes('data-frame')).toBe('0')
    vi.advanceTimersByTime(300)
    expect(w.emitted('reacted')).toEqual([['miss']])
  })

  it('changing react mid-hold restarts the timer and emits only for the new value', async () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'hit', reduced: true } })
    vi.advanceTimersByTime(150)
    await w.setProps({ react: 'miss' })
    vi.advanceTimersByTime(150)
    expect(w.emitted('reacted')).toBeUndefined()
    vi.advanceTimersByTime(150)
    expect(w.emitted('reacted')).toEqual([['miss']])
  })

  it('unmounting mid-hold emits nothing and leaves no pending timer', () => {
    const w = mount(CompanionSprite, { props: { stage: 'sprout', health: 80, react: 'hit', reduced: true } })
    w.unmount()
    expect(vi.getTimerCount()).toBe(0)
    vi.advanceTimersByTime(1000)
    expect(w.emitted('reacted')).toBeUndefined()
  })
})

describe('CompanionSprite reduced default (design amend A4)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('with no reduced prop, falls back to the OS setting: holds the static frame, then emits reacted', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: CompanionSpriteMocked } = await import('~/components/retro/CompanionSprite.vue')
    const w = mount(CompanionSpriteMocked, { props: { stage: 'sprout', health: 80, react: 'hit' } })
    expect(w.attributes('data-frame')).toBe('1')
    vi.advanceTimersByTime(300)
    expect(w.emitted('reacted')).toEqual([['hit']])
  })

  it('an explicit reduced=false overrides the OS setting: the animation class is applied', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: CompanionSpriteMocked } = await import('~/components/retro/CompanionSprite.vue')
    const w = mount(CompanionSpriteMocked, { props: { stage: 'sprout', health: 80, react: 'hit', reduced: false } })
    expect(w.classes()).toContain('retro-hop')
  })
})

describe('CompanionSprite live OS flip (design A11)', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('mounted without reduced, the OS flipping to reduced mid-session holds the frame and then emits reacted', async () => {
    const osReduced = ref(false)
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => osReduced }))
    const { default: CompanionSpriteMocked } = await import('~/components/retro/CompanionSprite.vue')
    const w = mount(CompanionSpriteMocked, { props: { stage: 'sprout', health: 80, react: 'hit' } })
    expect(w.classes()).toContain('retro-hop')
    osReduced.value = true
    await w.vm.$nextTick()
    expect(w.attributes('data-frame')).toBe('1')
    vi.advanceTimersByTime(300)
    await w.vm.$nextTick()
    expect(w.emitted('reacted')).toEqual([['hit']])
  })
})
