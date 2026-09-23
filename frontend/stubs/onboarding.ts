/**
 * STUB — replaces the onboarding backend until its slice merges. Shapes:
 * quiz = the onboarding plan's QuizResponse; assessment = Backend spec §6.1.
 * Selected by NUXT_PUBLIC_STUB_ONBOARDING=true (nuxt.config runtimeConfig).
 */
export interface QuizQuestion {
  id: string
  prompt: string
  options: Record<string, string>
}
export interface QuizResponse {
  questions: QuizQuestion[]
}
export interface AssessmentRequest {
  target_goal: string
  notification_time: string
  timezone: string
  answers: { question_id: string, selected_option: string }[]
}
export interface AssessmentResponse {
  status: 'success'
  assessed_level: string
  roadmap_id: string
  pet_state: { plant_name: string, health_points: number, stage: string }
}

export const STUB_QUIZ: QuizResponse = {
  questions: [
    { id: 'q1', prompt: 'She ___ to work every day.', options: { A: 'go', B: 'goes', C: 'going', D: 'gone' } },
    { id: 'q2', prompt: 'Choose the correct formal phrasing for requesting a price quotation:', options: { A: 'Give me the cost details right now.', B: 'Could you please provide a price quotation?', C: 'Send me how much this thing costs.', D: 'Price. Now.' } },
    { id: 'q3', prompt: 'If I ___ more time, I would travel.', options: { A: 'have', B: 'had', C: 'has', D: 'having' } },
    { id: 'q4', prompt: 'The report ___ by the team before the deadline.', options: { A: 'completed', B: 'was completed', C: 'has completing', D: 'complete' } },
    { id: 'q5', prompt: 'Which word is closest in meaning to "inquire"?', options: { A: 'ask', B: 'ignore', C: 'answer', D: 'inspire' } },
  ],
}

const STUB_KEY = { q1: 'B', q2: 'B', q3: 'B', q4: 'B', q5: 'A' } as const

export function stubAssessment(req: AssessmentRequest): AssessmentResponse {
  const correct = req.answers.filter(a => (STUB_KEY as Record<string, string>)[a.question_id] === a.selected_option).length
  const level = correct >= 5 ? 'B2' : correct >= 3 ? 'B1' : correct >= 1 ? 'A2' : 'A1'
  return {
    status: 'success',
    assessed_level: level,
    roadmap_id: 'stub-roadmap-0000-0000-000000000000',
    pet_state: { plant_name: 'My Green Buddy', health_points: 100, stage: 'sprout' },
  }
}
