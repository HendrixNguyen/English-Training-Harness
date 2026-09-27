import { describe, expect, it } from 'vitest'
import {
  MAX_SHIELDS,
  PLANT_STAGES,
  SHIELD_EVERY_DAYS,
  STREAK_MILESTONES,
  daysBetweenDates,
  daysSince,
  healthTone,
  localDateYmd,
  normalizeStage,
  shieldSpentLine,
  speechLine,
  stageForStreak,
} from '~/utils/plant'

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

  it('STREAK_MILESTONES / MAX_SHIELDS / SHIELD_EVERY_DAYS pin the constants the backend engine and migration 0004 agree on', () => {
    expect(STREAK_MILESTONES).toEqual([3, 7, 14, 21, 28])
    expect(MAX_SHIELDS).toBe(2)
    expect(SHIELD_EVERY_DAYS).toBe(7)
  })

  // --- streak-shield helpers, copied verbatim from
  // origin/harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days
  // frontend/tests/unit/plant.test.ts (design amend, plan "Order constraint"). ---
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

  describe('speechLine (design retro-hub.md Addendum H1 — kit register, first match wins)', () => {
    it('row 1: wilted or 0 health is silent, whatever else is true (the revive band speaks instead)', () => {
      expect(speechLine({ stage: 'wilted', health: 0, targetMet: false })).toBe('…')
      expect(speechLine({ stage: 'sprout', health: 0, targetMet: true, streak: 7, shields: 1 })).toBe('…')
    })

    it('row 2: a shield earned today (targetMet, streak a multiple of 7, shields > 0)', () => {
      expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7, shields: 1 }))
        .toBe('Tròn 7 ngày liên tiếp! Cậu có khiên giữ chuỗi ngày rồi.')
      expect(speechLine({ stage: 'fruitful', health: 100, targetMet: true, streak: 21, shields: 2 }))
        .toBe('Tròn 21 ngày liên tiếp! Cậu có khiên giữ chuỗi ngày rồi.') // full rack: still true
    })

    it('row 2 requires targetMet — without it, falls through to a later row (never repeats the next morning)', () => {
      expect(speechLine({ stage: 'flowering', health: 80, targetMet: false, streak: 7, shields: 1 }))
        .toBe('Tưới cho tớ 10 phút đi, cậu.') // falls to row 8
      expect(speechLine({ stage: 'flowering', health: 80, targetMet: false, streak: 7, shields: 1, accumulatedSeconds: 600 }))
        .toBe('Còn 20 phút nữa là xong phòng hôm nay!') // falls to row 5
    })

    it('row 2 is skipped entirely while shields is not a number (before the streak-shield branch merges)', () => {
      expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7 }))
        .toBe('7 ngày liên tiếp! Hôm nay tớ đủ nước rồi, cảm ơn cậu.') // row 3, not row 2
      expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7, shields: 0 }))
        .toBe('7 ngày liên tiếp! Hôm nay tớ đủ nước rồi, cảm ơn cậu.') // shields present but 0 → also not row 2
    })

    it('row 3: target met on a milestone streak', () => {
      for (const streak of STREAK_MILESTONES) {
        expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak }))
          .toBe(`${streak} ngày liên tiếp! Hôm nay tớ đủ nước rồi, cảm ơn cậu.`)
      }
    })

    it('row 4: target met, no milestone', () => {
      expect(speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 8 }))
        .toBe('Hôm nay tớ đủ nước rồi, cảm ơn cậu.')
      expect(speechLine({ stage: 'sprout', health: 100, targetMet: true })).toBe('Hôm nay tớ đủ nước rồi, cảm ơn cậu.')
    })

    it('row 5: mid-day, minutes left round up', () => {
      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 600 })).toBe('Còn 20 phút nữa là xong phòng hôm nay!')
      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 1200 })).toBe('Còn 10 phút nữa là xong phòng hôm nay!')
      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 1770 })).toBe('Còn 1 phút nữa là xong phòng hôm nay!')
    })

    it('row 6: day 1, nothing studied, no streak, never practiced — greets by name', () => {
      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, dayNumber: 1, accumulatedSeconds: 0, streak: 0, lastPracticedAt: null, name: 'Mầm Non' }))
        .toBe('Chào cậu. Tớ là Mầm Non — cùng vào phòng đầu tiên nhé?')
    })

    it('row 6 is day-1-only: the same values on day 2 are not the first day (falls to row 8)', () => {
      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, dayNumber: 2, accumulatedSeconds: 0, streak: 0, lastPracticedAt: null, name: 'Mầm Non' }))
        .toBe('Tưới cho tớ 10 phút đi, cậu.')
    })

    it('row 7: missed — 25h since practice (still "yesterday") is not missed; 2 calendar days is', () => {
      const now = new Date(2026, 8, 25, 12, 0, 0) // local noon, away from any midnight boundary
      const yesterday25hAgo = new Date(now.getTime() - 25 * 3600 * 1000).toISOString()
      const twoDaysAgo = new Date(now.getTime() - 48 * 3600 * 1000).toISOString()

      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, lastPracticedAt: yesterday25hAgo, now }))
        .toBe('Tưới cho tớ 10 phút đi, cậu.') // row 8, not missed
      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false, lastPracticedAt: twoDaysAgo, now }))
        .toBe('Hôm qua tớ nhớ cậu… Tưới 10 phút nhé?')
    })

    it('rows 8-10: health bands when nothing else matches', () => {
      expect(speechLine({ stage: 'sprout', health: 80, targetMet: false })).toBe('Tưới cho tớ 10 phút đi, cậu.')
      expect(speechLine({ stage: 'sapling', health: 45, targetMet: false })).toBe('Tớ hơi khát rồi… 10 phút thôi?')
      expect(speechLine({ stage: 'sapling', health: 10, targetMet: false })).toBe('Tớ sắp héo mất. Học một chút nhé?')
    })

    it('no returned line contains "bạn", an emoji, or more than one "!"; the name appears only in row 6', () => {
      const lines = [
        speechLine({ stage: 'wilted', health: 0, targetMet: false }),
        speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7, shields: 1 }),
        speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 7 }),
        speechLine({ stage: 'flowering', health: 100, targetMet: true, streak: 8 }),
        speechLine({ stage: 'sprout', health: 80, targetMet: false, accumulatedSeconds: 600 }),
        speechLine({ stage: 'sprout', health: 80, targetMet: false, dayNumber: 1, accumulatedSeconds: 0, streak: 0, lastPracticedAt: null, name: 'Mầm Non' }),
        speechLine({ stage: 'sprout', health: 80, targetMet: false, lastPracticedAt: new Date(Date.now() - 172_800_000).toISOString() }),
        speechLine({ stage: 'sprout', health: 80, targetMet: false }),
        speechLine({ stage: 'sapling', health: 45, targetMet: false }),
        speechLine({ stage: 'sapling', health: 10, targetMet: false }),
      ]
      for (const line of lines) {
        expect(line).not.toContain('bạn')
        expect(line).not.toMatch(/\p{Extended_Pictographic}/u)
        expect(line.split('!').length - 1).toBeLessThanOrEqual(1)
      }
      const withoutName = lines.filter((_, i) => i !== 5)
      for (const line of withoutName) expect(line).not.toContain('Mầm Non')
    })
  })
})
