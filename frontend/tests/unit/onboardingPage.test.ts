import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GoalCard from '~/components/onboarding/GoalCard.vue'
import PlantSvg from '~/components/plant/PlantSvg.vue'
import AppButton from '~/components/ui/AppButton.vue'
import AppCard from '~/components/ui/AppCard.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { ApiError } from '~/utils/apiClient'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const navigateTo = vi.fn()
vi.stubGlobal('navigateTo', navigateTo)

const { default: OnboardingPage } = await import('~/pages/onboarding.vue')

/** Ten items shaped like backend bank.go's PublicBank(); prompts exist only in the real bank. */
const QUIZ = {
  questions: Array.from({ length: 10 }, (_, i) => ({
    id: `q${i + 1}`,
    prompt: i === 0 ? 'She ___ a teacher.' : `Real bank item ${i + 1}`,
    options: { A: 'am', B: 'is', C: 'are', D: 'be' },
  })),
}
/** §6.1 201 body with values the stub never produced. */
const ASSESSED = {
  status: 'success',
  assessed_level: 'C1',
  roadmap_id: 'b11c22d3-44e5-66f7-88a9-00bbccddeeff',
  pet_state: { plant_name: 'Cây Thử', health_points: 100, stage: 'sprout' },
}

function mountPage() {
  api.get.mockImplementation((path: string) =>
    path.endsWith('/onboarding/quiz') ? Promise.resolve(QUIZ) : Promise.reject(new ApiError(404, 'no_active_roadmap')))
  return mount(OnboardingPage, {
    global: {
      components: { AppButton, AppCard, GoalCard, PlantSvg, StateBlock },
      stubs: { AppHeader: true },
      mocks: { navigateTo },
    },
  })
}

async function click(w: VueWrapper, text: string) {
  const b = w.findAll('button').find(x => x.text().includes(text))
  if (!b) throw new Error(`no button "${text}"`)
  await b.trigger('click')
  await flushPromises()
}

/** Goal → start → answer option A on every item → submit. */
async function completeQuiz(w: VueWrapper) {
  await click(w, 'IELTS 7.0')
  await click(w, 'Bắt đầu bài kiểm tra')
  for (let i = 0; i < QUIZ.questions.length; i++) {
    await w.find('[role="radio"]').trigger('click')
    await click(w, i === QUIZ.questions.length - 1 ? 'Hoàn thành' : 'Tiếp tục')
  }
}

describe('/onboarding against the real endpoints (backend spec §6.1)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    api.get.mockReset()
    api.post.mockReset()
    navigateTo.mockReset()
  })

  it('loads the placement items from GET /api/v1/onboarding/quiz and renders what the server sent', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')
    await click(w, 'Bắt đầu bài kiểm tra')

    expect(api.get).toHaveBeenCalledWith('/api/v1/onboarding/quiz')
    expect(w.text()).toContain('Câu 1 / 10')
    expect(w.text()).toContain('She ___ a teacher.')
    expect(w.text()).not.toContain('Bản thử')
  })

  it('posts the §6.1 assessment body and renders the assessed level and pet from the response', async () => {
    api.post.mockResolvedValue(ASSESSED)
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    expect(api.post).toHaveBeenCalledTimes(1)
    const [path, body] = api.post.mock.calls[0] as [string, Record<string, unknown>]
    expect(path).toBe('/api/v1/onboarding/assessment')
    expect(body).toEqual({
      target_goal: 'IELTS 7.0 Preparation',
      notification_time: '20:00:00',
      timezone: expect.stringMatching(/.+/),
      answers: QUIZ.questions.map(q => ({ question_id: q.id, selected_option: 'A' })),
    })
    expect(w.text()).toContain('Trình độ của bạn: C1')
    expect(w.text()).toContain('Cây Thử đã nảy mầm')
  })

  it('rejects a plant name that is too long or has symbols inline, and accepts a trimmed plain one', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')

    const start = () => w.findAll('button').find(b => b.text().includes('Bắt đầu bài kiểm tra'))
    const nameInput = w.find('input[name="plant_name"]')

    await nameInput.setValue('a'.repeat(31))
    expect(start()?.attributes('disabled')).toBeDefined()
    expect(w.find('[role="note"]').text()).toContain('Tên cây')

    await nameInput.setValue('Mầm Non!')
    expect(start()?.attributes('disabled')).toBeDefined()
    expect(w.find('[role="note"]').text()).toContain('Tên cây')

    await nameInput.setValue('  Mầm Non ')
    expect(start()?.attributes('disabled')).toBeUndefined()
    expect(w.find('[role="note"]').exists()).toBe(false)

    await nameInput.setValue('   ')
    expect(start()?.attributes('disabled')).toBeUndefined()
    expect(w.find('[role="note"]').exists()).toBe(false)
  })

  it('sends plant_name trimmed when the learner typed one', async () => {
    api.post.mockResolvedValue(ASSESSED)
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')
    await w.find('input[name="plant_name"]').setValue('  Lá Xanh ')
    await completeQuiz(w)

    const [, body] = api.post.mock.calls[0] as [string, Record<string, unknown>]
    expect(body).toMatchObject({ plant_name: 'Lá Xanh' })
  })

  it('names a 429 rate_limited honestly and keeps the learner on the quiz with their answers', async () => {
    api.post.mockRejectedValue(new ApiError(429, 'rate_limited'))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    expect(w.find('[role="alert"]').text()).toContain('quá nhiều lần')
    expect(w.text()).toContain('Câu 10 / 10')
    expect(w.text()).not.toContain('Trình độ của bạn')
  })

  it('does not start the quiz while the reminder time is cleared, and says why', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')
    await w.find('input[type="time"]').setValue('')

    const start = w.findAll('button').find(b => b.text().includes('Bắt đầu bài kiểm tra'))
    if (!start) throw new Error('no start button')
    expect(start.attributes('disabled')).toBeDefined()
    expect(w.find('[role="note"]').text()).toContain('giờ nhắc học')
    await start.trigger('click')
    await flushPromises()
    expect(api.get).not.toHaveBeenCalledWith('/api/v1/onboarding/quiz')

    await w.find('input[type="time"]').setValue('07:30')
    expect(start.attributes('disabled')).toBeUndefined()
    expect(w.find('[role="note"]').exists()).toBe(false)
  })

  it('names a 400 invalid_request so the learner knows what to fix, keeping their answers', async () => {
    api.post.mockRejectedValue(new ApiError(400, 'invalid_request'))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    const alert = w.find('[role="alert"]').text()
    expect(alert).toContain('giờ nhắc học')
    expect(alert).not.toContain('Không tạo được lộ trình')
    expect(w.text()).toContain('Câu 10 / 10')
  })

  it.each([
    [503, 'ai_unavailable'],
    [502, 'ai_bad_output'],
    [502, 'ai_upstream_failed'],
  ])('renders the AI-specific copy for %i %s and keeps the learner on the quiz', async (status, code) => {
    api.post.mockRejectedValue(new ApiError(status, code))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)

    expect(w.find('[role="alert"]').text()).toContain('Máy chủ AI đang bận')
    expect(w.text()).toContain('Câu 10 / 10')
    expect(w.text()).not.toContain('Trình độ của bạn')
  })

  it('says the grading and roadmap can take a minute or two while the POST is pending', async () => {
    let finish!: (v: unknown) => void
    api.post.mockImplementation(() => new Promise((resolve) => { finish = resolve }))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)
    expect(w.text()).toContain('thường mất 1–2 phút')
    finish(ASSESSED)
    await flushPromises()
    expect(w.text()).not.toContain('thường mất 1–2 phút')
    expect(w.text()).toContain('C1')
  })
})
