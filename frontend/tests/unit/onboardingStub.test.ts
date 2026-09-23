import { describe, expect, it } from 'vitest'
import { STUB_QUIZ, stubAssessment } from '~/stubs/onboarding'

describe('onboarding stub (mirrors the onboarding plan and Backend spec §6.1)', () => {
  it('serves questions shaped like GET /api/v1/onboarding/quiz', () => {
    expect(STUB_QUIZ.questions.length).toBeGreaterThanOrEqual(3)
    for (const q of STUB_QUIZ.questions) {
      expect(q).toMatchObject({ id: expect.any(String), prompt: expect.any(String) })
      expect(Object.keys(q.options)).toEqual(['A', 'B', 'C', 'D'])
    }
  })

  it('answers the §6.1 assessment shape', () => {
    const res = stubAssessment({
      target_goal: 'IELTS 7.0',
      notification_time: '20:00:00',
      timezone: 'Asia/Ho_Chi_Minh',
      answers: [{ question_id: 'q1', selected_option: 'B' }],
    })
    expect(res).toEqual({
      status: 'success',
      assessed_level: expect.stringMatching(/^[ABC][12]$/),
      roadmap_id: expect.any(String),
      pet_state: { plant_name: 'My Green Buddy', health_points: 100, stage: 'sprout' },
    })
  })
})
