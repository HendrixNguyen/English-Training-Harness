import { describe, expect, it } from 'vitest'
import {
  DAILY_TARGET_SECONDS,
  SEGMENT_COUNT,
  SEGMENT_SECONDS,
  clampDuration,
  formatCountdown,
  minutesOf,
  percentOf,
  segmentFills,
} from '~/utils/progress'

describe('the 30-minute target (design §1)', () => {
  it('is three segments of ten minutes', () => {
    expect(SEGMENT_COUNT).toBe(3)
    expect(SEGMENT_SECONDS).toBe(600)
    expect(DAILY_TARGET_SECONDS).toBe(1800)
  })

  it('fills segments left to right', () => {
    expect(segmentFills(0)).toEqual([0, 0, 0])
    expect(segmentFills(300)).toEqual([0.5, 0, 0])
    expect(segmentFills(1200)).toEqual([1, 1, 0])
    expect(segmentFills(1500)).toEqual([1, 1, 0.5])
    expect(segmentFills(4000)).toEqual([1, 1, 1])
  })

  it('supports the 15-minute single-segment revive bar', () => {
    expect(segmentFills(450, 900, 1)).toEqual([0.5])
  })

  it('labels minutes and percent the way wireframe 7.2 does (20 / 30, 66%)', () => {
    expect(minutesOf(1200)).toBe(20)
    expect(minutesOf(1259)).toBe(20)
    expect(percentOf(1200)).toBe(66)
    expect(percentOf(1800)).toBe(100)
    expect(percentOf(5000)).toBe(100)
  })

  it('clamps a posted duration to the backend range 1..3600', () => {
    expect(clampDuration(0)).toBe(1)
    expect(clampDuration(-5)).toBe(1)
    expect(clampDuration(599.7)).toBe(599)
    expect(clampDuration(99_999)).toBe(3600)
  })

  it('formats a countdown as mm:ss', () => {
    expect(formatCountdown(582)).toBe('09:42')
    expect(formatCountdown(0)).toBe('00:00')
    expect(formatCountdown(3600)).toBe('60:00')
  })
})
