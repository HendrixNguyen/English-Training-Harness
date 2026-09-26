import { describe, expect, it } from 'vitest'
import { PLANT_STAGES, STREAK_MILESTONES, daysSince, healthTone, normalizeStage, speechLine, stageForStreak } from '~/utils/plant'

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

  it('stageForStreak follows the backend streak table and wilts at 0 health', () => {
    expect(stageForStreak(0, 80)).toBe('sprout')
    expect(stageForStreak(2, 80)).toBe('sprout')
    expect(stageForStreak(3, 80)).toBe('sapling')
    expect(stageForStreak(6, 80)).toBe('sapling')
    expect(stageForStreak(7, 80)).toBe('flowering')
    expect(stageForStreak(13, 80)).toBe('flowering')
    expect(stageForStreak(14, 80)).toBe('fruitful')
    expect(stageForStreak(30, 100)).toBe('fruitful')
    expect(stageForStreak(5, 0)).toBe('wilted')
  })

  it('speechLine: partway lines count the minutes left, rounded up', () => {
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 600 })).toBe('Còn 20 phút nữa thôi!')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 1200 })).toBe('Còn 10 phút nữa thôi!')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 1770 })).toBe('Còn 1 phút nữa thôi!')
    expect(speechLine({ stage: 'sprout', health: 10, targetMet: false, accumulatedSeconds: 600 })).toBe('Còn 20 phút nữa thôi!')
  })

  it('speechLine: milestone streaks prefix the met line', () => {
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7 })).toBe('7 ngày liên tiếp! Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 8 })).toBe('Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    expect(STREAK_MILESTONES).toEqual([3, 7, 14, 21, 28])
  })

  it.each([3, 7, 14, 21, 28])('speechLine: milestone streak %i prefixes the met line', (streak) => {
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak })).toBe(`${streak} ngày liên tiếp! Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿`)
  })

  it('speechLine: a missed day is noticed only when nothing was studied today', () => {
    const now = new Date('2026-09-25T10:00:00Z')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 0, lastPracticedAt: '2026-09-23T20:00:00Z', now })).toBe('Hôm qua tớ nhớ bạn… Tưới 10 phút nhé?')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 0, lastPracticedAt: '2026-09-24T20:00:00Z', now })).toBe('Tưới cho tớ 10 phút học đi!')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 600, lastPracticedAt: '2026-09-23T20:00:00Z', now })).toBe('Còn 20 phút nữa thôi!')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 0, lastPracticedAt: null, now })).toBe('Tưới cho tớ 10 phút học đi!')
  })

  it('speechLine: wilted stays silent whatever else is true', () => {
    expect(speechLine({ stage: 'wilted', health: 0, targetMet: true, streak: 7, accumulatedSeconds: 1800 })).toBe('…')
  })
})
