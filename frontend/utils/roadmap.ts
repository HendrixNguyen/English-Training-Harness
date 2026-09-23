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
