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

describe('chipsFor (design retro-hub.md Addendum H3 — kit chest-item chips)', () => {
  it.each([
    [{ healthFrom: 80, healthTo: 100, streakFrom: 5, streakTo: 6, targetMetNow: true }, [{ tone: 'growth', text: '+20 HP' }, { tone: 'torch', text: 'x6' }]],
    [{ healthFrom: 100, healthTo: 100, streakFrom: 6, streakTo: 7, targetMetNow: true }, [{ tone: 'growth', text: 'HP đầy' }, { tone: 'torch', text: 'x7' }]],
    [{ healthFrom: 80, healthTo: 80, streakFrom: 5, streakTo: 5, targetMetNow: false }, []],
    [{ healthFrom: 0, healthTo: 20, streakFrom: 0, streakTo: 1, targetMetNow: true }, [{ tone: 'growth', text: '+20 HP' }, { tone: 'torch', text: 'x1' }]],
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
      const { start, active, chips, react, displayHealth } = useGrowthMoment()
      start(null)
      expect(active.value).toBe(false)
      expect(chips.value).toEqual([])
      expect(react.value).toBe('idle')
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

  it('timeline: react turns "hit" at grow time when the stage did not change, and resets to idle at the end', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, chips, active, react, pulse } = useGrowthMoment()
      start(delta({ targetMetNow: true, stageFrom: 'sprout', stageTo: 'sprout' }))
      vi.advanceTimersByTime(149)
      expect(chips.value).toEqual([])
      vi.advanceTimersByTime(1)
      expect(chips.value.length).toBe(2)
      expect(pulse.value).toBe(true)
      vi.advanceTimersByTime(250)
      expect(react.value).toBe('hit')
      vi.advanceTimersByTime(1400)
      expect(chips.value).toEqual([])
      vi.advanceTimersByTime(200)
      expect(active.value).toBe(false)
      expect(react.value).toBe('idle')
      expect(pulse.value).toBe(false)
    })
    scope.stop()
  })

  it('react is "levelup" at grow time when the stage changed, whether or not the target was met', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, react } = useGrowthMoment()
      start(delta({ targetMetNow: true, stageFrom: 'sprout', stageTo: 'sapling' }))
      vi.advanceTimersByTime(400)
      expect(react.value).toBe('levelup')
    })
    scope.stop()

    const scope2 = effectScope()
    scope2.run(() => {
      const { start, react } = useGrowthMoment()
      start(delta({ targetMetNow: false, stageFrom: 'sprout', stageTo: 'sapling' }))
      vi.advanceTimersByTime(400)
      expect(react.value).toBe('levelup')
    })
    scope2.stop()
  })

  it('react stays "idle" when the target was not newly met and the stage did not change', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
    const scope = effectScope()
    scope.run(() => {
      const { start, react } = useGrowthMoment()
      start(delta({ targetMetNow: false, stageFrom: 'sprout', stageTo: 'sprout' }))
      vi.advanceTimersByTime(400)
      expect(react.value).toBe('idle')
    })
    scope.stop()
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
