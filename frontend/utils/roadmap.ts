export const ROADMAP_DAYS = 28

export type NodeState = 'completed' | 'today' | 'locked'
export interface RoadmapNode {
  day: number
  week: number
  state: NodeState
}

/**
 * design §2.5. There is no per-day completion endpoint, so "completed" means
 * "before today" — derived from GET /quests/daily day_number (open question).
 */
export function roadmapNodes(dayNumber: number, total = ROADMAP_DAYS): RoadmapNode[] {
  const today = Math.min(total, Math.max(1, Math.floor(dayNumber)))
  return Array.from({ length: total }, (_, i) => {
    const day = i + 1
    return {
      day,
      week: Math.floor(i / 7) + 1,
      state: day < today ? 'completed' : day === today ? 'today' : 'locked',
    }
  })
}

/** GET /api/v1/roadmap: summaries only, no exercise content or invented history. */
export interface RoadmapTask { id: string, task_type: string, title: string, duration_minutes: number, is_completed: boolean | null }
export interface RoadmapDay { day_number: number, date: string, title: string, tasks: RoadmapTask[], minutes_spent: number, is_target_met: boolean }
export interface RoadmapModule { week: number, title: string, focus: string, days: RoadmapDay[] }
export interface RoadmapOutline { roadmap_id: string, title: string, cefr_level: string, today: string, day_number: number, modules: RoadmapModule[] }
export type MapState = 'cleared' | 'today' | 'partial' | 'missed' | 'locked'
export const TASK_LABELS: Record<string, string> = { vocabulary: 'Từ vựng', reading: 'Đọc hiểu', practice: 'Thực hành' }
export function dayState(day: RoadmapDay, today: string): MapState {
  if (day.date === today) return 'today'
  if (day.is_target_met === true) return 'cleared'
  if (day.date > today) return 'locked'
  return day.minutes_spent > 0 ? 'partial' : 'missed'
}
export function dayStatus(day: RoadmapDay, today: string): string {
  const state = dayState(day, today)
  return state === 'cleared' ? 'Đã xong' : state === 'missed' ? 'Bỏ lỡ' : state === 'locked' ? 'Chưa mở' : day.is_target_met ? 'Đã đủ 30 phút' : `${day.minutes_spent}/30 phút`
}
/** Parse calendar components in UTC; browser timezone must never move a server date. */
export function calendarLabel(date: string): string {
  const [year, month, day] = date.split('-').map(Number)
  const weekday = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'][new Date(Date.UTC(year!, month! - 1, day!)).getUTCDay()]
  return `${weekday} ${day}/${month}`
}
