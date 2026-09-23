import { useRuntimeConfig } from '#app'
import { useApi } from '~/composables/useApi'
import { STUB_QUIZ, stubAssessment, type AssessmentRequest, type AssessmentResponse, type QuizResponse } from '~/stubs/onboarding'

export function useOnboardingApi() {
  const stub = useRuntimeConfig().public.stubOnboarding === 'true'
  return {
    isStub: stub,
    quiz: (): Promise<QuizResponse> => (stub ? Promise.resolve(STUB_QUIZ) : useApi().get<QuizResponse>('/api/v1/onboarding/quiz')),
    assess: (req: AssessmentRequest): Promise<AssessmentResponse> =>
      stub ? Promise.resolve(stubAssessment(req)) : useApi().post<AssessmentResponse>('/api/v1/onboarding/assessment', req),
  }
}
