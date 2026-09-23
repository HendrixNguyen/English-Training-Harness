import { describe, expect, it } from 'vitest'
import { PLANT_STAGES, daysSince, healthTone, normalizeStage, speechLine } from '~/utils/plant'

describe('plant helpers (design §3)', () => {
  it('knows the six DDL stages', () => {
    expect(PLANT_STAGES).toEqual(['seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted'])
  })

  it('normalises unknown stages to sprout and flags them', () => {
    expect(normalizeStage('fruitful')).toEqual({ stage: 'fruitful', known: true })
    expect(normalizeStage('cactus')).toEqual({ stage: 'sprout', known: false })
    expect(normalizeStage(undefined)).toEqual({ stage: 'sprout', known: false })
  })

  it('tones health: growth ≥ 60, streak ≥ 30, alert below', () => {
    expect(healthTone(100)).toBe('growth')
    expect(healthTone(60)).toBe('growth')
    expect(healthTone(59)).toBe('streak')
    expect(healthTone(30)).toBe('streak')
    expect(healthTone(29)).toBe('alert')
    expect(healthTone(0)).toBe('alert')
  })

  it('speaks the wireframe 7.2 line when healthy and the target is not met', () => {
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false })).toBe('Tưới cho tớ 10 phút học đi!')
    expect(speechLine({ stage: 'sapling', health: 45, targetMet: false })).toBe('Tớ hơi khát rồi… 10 phút thôi?')
    expect(speechLine({ stage: 'sapling', health: 10, targetMet: false })).toBe('Tớ sắp héo mất! Học một chút nhé?')
    expect(speechLine({ stage: 'sapling', health: 10, targetMet: true })).toBe('Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    expect(speechLine({ stage: 'wilted', health: 0, targetMet: false })).toBe('…')
  })

  it('counts whole days since last practice, or null when unknown', () => {
    const now = new Date('2026-09-23T10:00:00Z')
    expect(daysSince('2026-09-21T20:15:00Z', now)).toBe(1)
    expect(daysSince('2026-09-20T09:00:00Z', now)).toBe(3)
    expect(daysSince(null, now)).toBeNull()
    expect(daysSince('not a date', now)).toBeNull()
  })
})
