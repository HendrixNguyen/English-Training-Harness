import { describe, expect, it } from 'vitest'
import { PLANT_STAGES, STREAK_MILESTONES, daysBetweenDates, daysSince, healthTone, localDateYmd, normalizeStage, shieldSpentLine, speechLine, stageForStreak } from '~/utils/plant'

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

  it('speaks the plant\'s name where "tớ" would be, and stays byte-identical without one', () => {
    expect(speechLine({ stage: 'sapling', health: 10, targetMet: true, name: 'Mầm Non' })).toBe('Cảm ơn bạn, hôm nay Mầm Non đủ nước rồi 🌿')
    expect(speechLine({ stage: 'sapling', health: 10, targetMet: false, name: 'Mầm Non' })).toBe('Mầm Non sắp héo mất! Học một chút nhé?')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, name: 'Mầm Non' })).toBe('Tưới cho tớ 10 phút học đi!')
    expect(speechLine({ stage: 'sapling', health: 10, targetMet: true, name: '  ' })).toBe('Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
  })

  it('counts whole days since last practice, or null when unknown', () => {
    const now = new Date('2026-09-23T10:00:00Z')
    expect(daysSince('2026-09-21T20:15:00Z', now)).toBe(1)
    expect(daysSince('2026-09-20T09:00:00Z', now)).toBe(3)
    expect(daysSince(null, now)).toBeNull()
    expect(daysSince('not a date', now)).toBeNull()
  })

  it('announces the shield on every 7th day while one is held (design §4.2, corrected 2026-09-27: only on the earn day)', () => {
    const earn = 'Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️'
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7, shields: 1 })).toBe(earn)
    expect(speechLine({ stage: 'fruitful', health: 100, targetMet: true, streak: 21, shields: 2 })).toBe(earn) // full rack: still true
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 8, shields: 1 })).toBe('Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    // shields: 0 on a milestone day falls through to main's milestone-prefixed met line, not the plain one.
    expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7, shields: 0 })).toBe('7 ngày liên tiếp! Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿')
    expect(speechLine({ stage: 'wilted', health: 0, targetMet: false, streak: 7, shields: 1 })).toBe('…')
    expect(speechLine({ stage: 'sprout', health: 80, targetMet: false })).toBe('Tưới cho tớ 10 phút học đi!') // callers without the new fields
    // the morning after: target not met today, streak/shields carried over from yesterday's earn — the ordinary health nudge, not the earn line.
    expect(speechLine({ stage: 'flowering', health: 80, targetMet: false, streak: 7, shields: 1 })).toBe('Tưới cho tớ 10 phút học đi!')
    // after a shielded spend: still not the earn line.
    expect(speechLine({ stage: 'fruitful', health: 80, targetMet: false, streak: 14, shields: 1 })).toBe('Tưới cho tớ 10 phút học đi!')
  })

  it('shows the spent-shield caption for seven days, dd/mm', () => {
    expect(shieldSpentLine('2026-09-24', '2026-09-24')).toBe('Khiên đã đỡ cho ngày 24/09.')
    expect(shieldSpentLine('2026-09-24', '2026-10-01')).toBe('Khiên đã đỡ cho ngày 24/09.') // day 7
    expect(shieldSpentLine('2026-09-24', '2026-10-02')).toBeNull() // day 8
    expect(shieldSpentLine('2026-09-24', '2026-09-23')).toBeNull() // clock skew: a future spend is not shown
    expect(shieldSpentLine(null, '2026-09-24')).toBeNull()
    expect(shieldSpentLine('not a date', '2026-09-24')).toBeNull()
    expect(daysBetweenDates('2026-02-28', '2026-03-01')).toBe(1)
    expect(localDateYmd(new Date(2026, 8, 5, 23, 30))).toBe('2026-09-05') // local getters, never toISOString
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
