import { TASK_ORDER } from '~/stores/quest'
import type { RoadmapDay, RoadmapOutline, RoadmapTask } from '~/stores/roadmap'

/**
 * design harness/designs/roadmap-tree.md §4.1. States come from the server's
 * per-day daily_progress join and its day_number; the client clock is never
 * consulted.
 */
export const ROADMAP_DAYS = 28
export const TARGET_MINUTES = 30

export type NodeState = 'completed' | 'partial' | 'missed' | 'today' | 'locked'

export interface RoadmapNode {
  day: number
  week: number
  date: string
  title: string
  state: NodeState
  minutesSpent: number
  isTargetMet: boolean
  tasks: RoadmapTask[]
}

/** The three task categories in Vietnamese (design §2). */
export const TASK_LABELS: Record<string, string> = {
  vocabulary: 'Từ vựng',
  reading: 'Đọc hiểu',
  practice: 'Thực hành',
}

function rank(type: string): number {
  const i = (TASK_ORDER as readonly string[]).indexOf(type)
  return i === -1 ? TASK_ORDER.length : i
}

/**
 * dayState precedence (design §4.1): today → completed (is_target_met) →
 * partial (minutes, no target) → missed → locked — by day_number relative to
 * the server's day_number, never the client clock.
 */
export function dayState(day: RoadmapDay, todayNumber: number): NodeState {
  if (day.day_number === todayNumber) return 'today'
  if (day.day_number > todayNumber) return 'locked'
  if (day.is_target_met) return 'completed'
  return day.minutes_spent > 0 ? 'partial' : 'missed'
}

/** Flattens the outline's 4 modules × 7 days into 28 nodes in order. */
export function roadmapNodes(outline: RoadmapOutline): RoadmapNode[] {
  const nodes: RoadmapNode[] = []
  for (const m of outline.modules) {
    for (const d of m.days) {
      nodes.push({
        day: d.day_number,
        week: m.week,
        date: d.date,
        title: d.title,
        state: dayState(d, outline.day_number),
        minutesSpent: d.minutes_spent,
        isTargetMet: d.is_target_met,
        tasks: [...d.tasks].sort((a, b) => rank(a.task_type) - rank(b.task_type)),
      })
    }
  }
  return nodes
}

/** Counts days across all modules whose is_target_met is true. */
export function completedDays(outline: RoadmapOutline): number {
  return outline.modules.reduce((n, m) => n + m.days.filter(d => d.is_target_met).length, 0)
}

/** A status line states a fact, never a judgement (design register). */
export function statusText(node: RoadmapNode): string {
  switch (node.state) {
    case 'completed':
      return 'Đã hoàn thành'
    case 'partial':
      return `${node.minutesSpent}/${TARGET_MINUTES} phút`
    case 'missed':
      return 'Bỏ lỡ'
    case 'locked':
      return 'Chưa mở khóa'
    case 'today':
      return node.isTargetMet ? `Đã đủ ${TARGET_MINUTES} phút` : `Đang học · ${node.minutesSpent}/${TARGET_MINUTES} phút`
  }
}

const WEEKDAYS = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7']

/** Parses a YYYY-MM-DD date as UTC (never `new Date(iso)` in local time, so
 * this is stable in any TZ) and formats it "{weekday} {d}/{m}". */
export function formatDayDate(date: string): string {
  const [y, m, d] = date.split('-').map(Number)
  const dt = new Date(Date.UTC(y, m - 1, d))
  return `${WEEKDAYS[dt.getUTCDay()]} ${d}/${m}`
}
