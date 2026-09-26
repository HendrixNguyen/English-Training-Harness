import { describe, expect, it, vi } from 'vitest'
import type { RoadmapDay, RoadmapOutline } from '~/stores/roadmap'
import {
  completedDays,
  dayState,
  formatDayDate,
  ROADMAP_DAYS,
  roadmapNodes,
  statusText,
  TARGET_MINUTES,
  TASK_LABELS,
} from '~/utils/roadmap'

// utils/roadmap.ts imports TASK_ORDER from ~/stores/quest, which imports the
// real useApi composable (Nuxt's #app alias, unavailable outside a Nuxt
// runtime) — mock it the way every store test does.
vi.mock('~/composables/useApi', () => ({ useApi: () => ({ get: vi.fn(), post: vi.fn() }) }))

/** Builds a day whose date is 2026-09-{n} (n <= 28) — tasks deliberately out
 * of the canonical order to prove roadmapNodes sorts them. */
function day(n: number, overrides: Partial<RoadmapDay> = {}): RoadmapDay {
  return {
    day_number: n,
    date: `2026-09-${String(n).padStart(2, '0')}`,
    title: `Day ${n}`,
    tasks: [
      { task_type: 'practice', title: 'practice task', duration_minutes: 10 },
      { task_type: 'vocabulary', title: 'vocabulary task', duration_minutes: 10 },
      { task_type: 'reading', title: 'reading task', duration_minutes: 10 },
    ],
    minutes_spent: 0,
    is_target_met: false,
    ...overrides,
  }
}

function outline(dayNumber: number, dayOverrides: Record<number, Partial<RoadmapDay>> = {}): RoadmapOutline {
  const modules = []
  for (let w = 1; w <= 4; w++) {
    const days: RoadmapDay[] = []
    for (let i = 0; i < 7; i++) {
      const n = (w - 1) * 7 + i + 1
      days.push(day(n, dayOverrides[n]))
    }
    modules.push({ week: w, title: `Week ${w}`, focus: 'focus', days })
  }
  return {
    roadmap_id: 'rm-1',
    title: 'Business English',
    cefr_level: 'B1',
    created_at: '2026-09-01T00:00:00Z',
    day_number: dayNumber,
    modules,
  }
}

describe('dayState (design §4.1)', () => {
  it('a past day that met the target is completed', () => {
    expect(dayState(day(1, { is_target_met: true, minutes_spent: 32 }), 5)).toBe('completed')
  })
  it('a past day with some minutes but no target is partial', () => {
    expect(dayState(day(1, { minutes_spent: 12 }), 5)).toBe('partial')
  })
  it('a past day with zero minutes is missed', () => {
    expect(dayState(day(1, { minutes_spent: 0 }), 5)).toBe('missed')
  })
  it('a past day with 30 minutes but not flagged met is still partial', () => {
    expect(dayState(day(1, { minutes_spent: 30, is_target_met: false }), 5)).toBe('partial')
  })
  it('day_number === today is always today, even met or at 0 minutes', () => {
    expect(dayState(day(5, { is_target_met: true, minutes_spent: 30 }), 5)).toBe('today')
    expect(dayState(day(5, { minutes_spent: 0 }), 5)).toBe('today')
  })
  it('a future day is locked even if a row (impossibly) says met', () => {
    expect(dayState(day(6, { is_target_met: true }), 5)).toBe('locked')
  })
})

describe('roadmapNodes (design §2)', () => {
  it('flattens 4x7 into 28 nodes in order', () => {
    const nodes = roadmapNodes(outline(1))
    expect(nodes).toHaveLength(ROADMAP_DAYS)
    expect(nodes[7].week).toBe(2)
  })
  it('sorts each node\'s tasks vocabulary → reading → practice', () => {
    const nodes = roadmapNodes(outline(1))
    expect(nodes[0].tasks.map(t => t.task_type)).toEqual(['vocabulary', 'reading', 'practice'])
  })
  it('carries the day\'s own title and date', () => {
    const nodes = roadmapNodes(outline(1))
    expect(nodes[0].title).toBe('Day 1')
    expect(nodes[0].date).toBe('2026-09-01')
  })
})

describe('completedDays', () => {
  it('counts days with is_target_met', () => {
    const o = outline(3, { 1: { is_target_met: true }, 2: { is_target_met: true } })
    expect(completedDays(o)).toBe(2)
  })
})

describe('statusText', () => {
  it('reports a fact per state', () => {
    expect(statusText({ day: 1, week: 1, date: '2026-09-01', title: 'Day 1', state: 'completed', minutesSpent: 32, isTargetMet: true, tasks: [] })).toBe('Đã hoàn thành')
    expect(statusText({ day: 1, week: 1, date: '2026-09-01', title: 'Day 1', state: 'partial', minutesSpent: 12, isTargetMet: false, tasks: [] })).toBe('12/30 phút')
    expect(statusText({ day: 1, week: 1, date: '2026-09-01', title: 'Day 1', state: 'missed', minutesSpent: 0, isTargetMet: false, tasks: [] })).toBe('Bỏ lỡ')
    expect(statusText({ day: 1, week: 1, date: '2026-09-01', title: 'Day 1', state: 'locked', minutesSpent: 0, isTargetMet: false, tasks: [] })).toBe('Chưa mở khóa')
    expect(statusText({ day: 1, week: 1, date: '2026-09-01', title: 'Day 1', state: 'today', minutesSpent: 10, isTargetMet: false, tasks: [] })).toBe('Đang học · 10/30 phút')
    expect(statusText({ day: 1, week: 1, date: '2026-09-01', title: 'Day 1', state: 'today', minutesSpent: 30, isTargetMet: true, tasks: [] })).toBe('Đã đủ 30 phút')
  })
})

describe('formatDayDate', () => {
  it('formats with the Vietnamese weekday abbreviation, parsed as UTC', () => {
    expect(formatDayDate('2026-09-22')).toBe('T3 22/9')
    expect(formatDayDate('2026-09-27')).toBe('CN 27/9')
    expect(formatDayDate('2026-10-01')).toBe('T5 1/10')
  })
})

describe('TASK_LABELS', () => {
  it('has exactly the three categories', () => {
    expect(TASK_LABELS).toEqual({ vocabulary: 'Từ vựng', reading: 'Đọc hiểu', practice: 'Thực hành' })
  })
})

describe('TARGET_MINUTES', () => {
  it('is 30', () => {
    expect(TARGET_MINUTES).toBe(30)
  })
})
