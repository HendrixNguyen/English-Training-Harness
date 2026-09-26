import { useApi } from '~/composables/useApi'

/** GET /api/v1/onboarding/quiz item (backend bank.go PublicQuestion — no answer, no level). */
export interface QuizQuestion {
  id: string
  prompt: string
  options: Record<string, string>
}
export interface QuizResponse {
  questions: QuizQuestion[]
}
/** Backend spec §6.1 POST /onboarding/assessment body. */
export interface AssessmentRequest {
  target_goal: string
  notification_time: string
  timezone: string
  /** Optional; trimmed 1–30 chars; omitted when blank → server default "Mầm Non". */
  plant_name?: string
  answers: { question_id: string, selected_option: string }[]
}
/** §6.1 response — 201 on a new roadmap, 200 when one was already active. */
export interface AssessmentResponse {
  status: 'success'
  assessed_level: string
  roadmap_id: string
  pet_state: { plant_name: string, health_points: number, stage: string }
}

export function useOnboardingApi() {
  return {
    quiz: (): Promise<QuizResponse> => useApi().get<QuizResponse>('/api/v1/onboarding/quiz'),
    assess: (req: AssessmentRequest): Promise<AssessmentResponse> =>
      useApi().post<AssessmentResponse>('/api/v1/onboarding/assessment', req),
  }
}
