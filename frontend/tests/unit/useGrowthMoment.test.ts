import { effectScope } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MOMENT_MS, chipsFor, useGrowthMoment } from '~/composables/useGrowthMoment'
import type { GrowthDelta } from '~/stores/pet'

function delta(overrides: Partial<GrowthDelta> = {}): GrowthDelta {
  return {
    healthFrom: 80,
    healthTo: 100,
    streakFrom: 5,
    streakTo: 6,
    stageFrom: 'sprout',
    stageTo: 'sprout',
    targetMetNow: true,
    ...overrides,
  }
}

describe('chipsFor (design growth-moment §3)', () => {
  it.each([
    [{ healthFrom: 80, healthTo: 100, streakFrom: 5, streakTo: 6, targetMetNow: true }, [{ tone: 'growth', text: '+20 máu' }, { tone: 'streak', text: '🔥 6 ngày' }]],
    [{ healthFrom: 100, healthTo: 100, streakFrom: 6, streakTo: 7, targetMetNow: true }, [{ tone: 'growth', text: 'Máu đầy' }, { tone: 'streak', text: '🔥 7 ngày' }]],
    [{ healthFrom: 80, healthTo: 80, streakFrom: 5, streakTo: 5, targetMetNow: false }, []],
    [{ healthFrom: 0, healthTo: 20, streakFrom: 0, streakTo: 1, targetMetNow: true }, [{ tone: 'growth', text: '+20 máu' }, { tone: 'streak', text: '🔥 1 ngày' }]],
  ])('%j -> %j', (partial, expected) => {
    expect(chipsFor(delta(partial as Partial<GrowthDelta>))).toEqual(expected)
  })
})

describe('useGrowthMoment', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('start(null) is inert', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, active, chips, displayHealth } = useGrowthMoment()
      start(null)
      expect(active.value).toBe(false)
      expect(chips.value).toEqual([])
      expect(displayHealth(80)).toBe(80)
    })
    scope.stop()
  })

  it('before-values for the first paint, then live', () => {
    let pending: FrameRequestCallback | null = null
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { pending = cb; return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, displayHealth, displayStage } = useGrowthMoment()
      start(delta({ stageFrom: 'sprout', stageTo: 'sapling' }))
      expect(displayHealth(100)).toBe(80)
      expect(displayStage('sapling')).toBe('sprout')
      pending!(0)
      expect(displayHealth(100)).toBe(100)
      expect(displayStage('sapling')).toBe('sapling')
    })
    scope.stop()
  })

  it('timeline', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, chips, active, grow, pulse } = useGrowthMoment()
      start(delta({ targetMetNow: true, stageFrom: 'sprout', stageTo: 'sprout' }))
      vi.advanceTimersByTime(149)
      expect(chips.value).toEqual([])
      vi.advanceTimersByTime(1)
      expect(chips.value.length).toBe(2)
      expect(pulse.value).toBe(true)
      vi.advanceTimersByTime(250)
      expect(grow.value).toBe(true)
      vi.advanceTimersByTime(1400)
      expect(chips.value).toEqual([])
      vi.advanceTimersByTime(200)
      expect(active.value).toBe(false)
      expect(grow.value).toBe(false)
      expect(pulse.value).toBe(false)
    })
    scope.stop()
  })

  it('grow only when the target was newly met and the stage did not change', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, grow } = useGrowthMoment()
      start(delta({ targetMetNow: false, stageFrom: 'sprout', stageTo: 'sprout' }))
      vi.advanceTimersByTime(400)
      expect(grow.value).toBe(false)
    })
    scope.stop()

    const scope2 = effectScope()
    scope2.run(() => {
      const { start, grow } = useGrowthMoment()
      start(delta({ targetMetNow: true, stageFrom: 'sprout', stageTo: 'sapling' }))
      vi.advanceTimersByTime(400)
      expect(grow.value).toBe(false)
    })
    scope2.stop()
  })

  it('dispose clears the timers', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, chips } = useGrowthMoment()
      start(delta({ targetMetNow: true }))
      vi.advanceTimersByTime(100)
      scope.stop()
      expect(() => vi.advanceTimersByTime(1900)).not.toThrow()
      expect(chips.value).toEqual([])
    })
  })
})

describe('MOMENT_MS', () => {
  it('matches design growth-moment §2', () => {
    expect(MOMENT_MS).toEqual({ chipsIn: 150, grow: 400, chipsOut: 1800, end: 2000 })
  })
})
