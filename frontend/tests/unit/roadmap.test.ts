import { describe, expect, it } from 'vitest'
import { ROADMAP_DAYS, roadmapNodes } from '~/utils/roadmap'

describe('roadmapNodes (design §2.5)', () => {
  it('derives 28 nodes from day_number: before = completed, equal = today, after = locked', () => {
    const nodes = roadmapNodes(3)
    expect(nodes).toHaveLength(ROADMAP_DAYS)
    expect(nodes[0]).toEqual({ day: 1, week: 1, state: 'completed' })
    expect(nodes[2]).toEqual({ day: 3, week: 1, state: 'today' })
    expect(nodes[3]).toEqual({ day: 4, week: 1, state: 'locked' })
    expect(nodes[27]).toEqual({ day: 28, week: 4, state: 'locked' })
  })

  it('groups seven days to a week (airouter RoadmapSchema: 4 modules × 7 days)', () => {
    const nodes = roadmapNodes(1)
    expect(nodes[6].week).toBe(1)
    expect(nodes[7].week).toBe(2)
    expect(nodes[21].week).toBe(4)
  })

  it('clamps day_number into 1..28', () => {
    expect(roadmapNodes(0)[0].state).toBe('today')
    expect(roadmapNodes(99)[27].state).toBe('today')
  })
})
