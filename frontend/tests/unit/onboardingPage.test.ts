import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RetroButton from '~/components/retro/RetroButton.vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import SpeechBox from '~/components/retro/SpeechBox.vue'
import QuestNode from '~/components/retro/QuestNode.vue'
import GoalCard from '~/components/onboarding/GoalCard.vue'
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
const DAILY = { date: '2026-10-13', day_number: 1, total_minutes_required: 30, accumulated_seconds: 0, is_target_met: false, tasks: [{ id: 'task-real', task_type: 'reading', title: 'A real C1 reading quest', duration_minutes: 10, is_completed: false, content_json: {} }] }
const ASSESSED = {
  status: 'success',
  assessed_level: 'C1',
  roadmap_id: 'b11c22d3-44e5-66f7-88a9-00bbccddeeff',
  pet_state: { plant_name: 'Cây Thử', health_points: 100, stage: 'sprout' },
}

function mountPage() {
  api.get.mockImplementation((path: string) =>
    path.endsWith('/onboarding/quiz') ? Promise.resolve(QUIZ) : api.post.mock.calls.length ? Promise.resolve(DAILY) : Promise.reject(new ApiError(404, 'no_active_roadmap')))
  return mount(OnboardingPage, {
    global: {
      components: { GoalCard, StateBlock, RetroButton, RetroPanel, CompanionSprite, SpeechBox, QuestNode },
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
  await click(w, 'Bắt đầu trận đầu tiên')
  for (let i = 0; i < QUIZ.questions.length; i++) {
    await w.find('[role="radio"]').trigger('click')
    await click(w, i === QUIZ.questions.length - 1 ? 'Kết thúc' : 'Chọn')
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
    await click(w, 'Bắt đầu trận đầu tiên')

    expect(api.get).toHaveBeenCalledWith('/api/v1/onboarding/quiz')
    expect(w.text()).toContain('Lượt 1 / 10')
    expect(w.text()).toContain('She ___ a teacher.')
    expect(w.text()).not.toContain('Bản thử')
  })

  it('returns to profile and resumes the same placement turn with its answer', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')
    await click(w, 'Bắt đầu trận đầu tiên')
    await w.find('[role="radio"]').trigger('click')
    await click(w, 'Chọn')
    await w.find('[role="radio"]').trigger('click')
    await click(w, '‹ Sửa thông tin')
    expect(w.text()).toContain('Chọn hành trình')
    await click(w, 'Bắt đầu trận đầu tiên')
    expect(w.text()).toContain('Lượt 2 / 10')
    expect(w.find('[role="radio"]').attributes('aria-checked')).toBe('true')
    expect(api.get.mock.calls.filter(([path]) => path === '/api/v1/onboarding/quiz')).toHaveLength(1)
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
    expect(w.text()).toContain('Cấp C1')
    expect(w.text()).toContain('Cây Thử')
  })

  it('rejects a plant name that is too long or has symbols inline, and accepts a trimmed plain one', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')

    const start = () => w.findAll('button').find(b => b.text().includes('Bắt đầu trận đầu tiên'))
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
    expect(w.text()).toContain('Lượt 10 / 10')
    expect(w.text()).not.toContain('Cấp ')
  })

  it('does not start the quiz while the reminder time is cleared, and says why', async () => {
    const w = mountPage()
    await flushPromises()
    await click(w, 'IELTS 7.0')
    await w.find('input[type="time"]').setValue('')

    const start = w.findAll('button').find(b => b.text().includes('Bắt đầu trận đầu tiên'))
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
    expect(w.text()).toContain('Lượt 10 / 10')
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
    expect(w.text()).toContain('Lượt 10 / 10')
    expect(w.text()).not.toContain('Cấp ')
  })

  it('shows honest synchronous waiting while the assessment POST is pending', async () => {
    let finish!: (v: unknown) => void
    api.post.mockImplementation(() => new Promise((resolve) => { finish = resolve }))
    const w = mountPage()
    await flushPromises()
    await completeQuiz(w)
    expect(w.text()).toContain('Thời gian tạo hành trình có thể kéo dài')
    finish(ASSESSED)
    await flushPromises()
    expect(w.text()).not.toContain('Thời gian tạo hành trình có thể kéo dài')
    expect(w.text()).toContain('C1')
  })
})

describe('retro onboarding readiness and calibration', () => {
 beforeEach(() => { setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset(); navigateTo.mockReset() })
 it('renders real day-one tasks and keeps the journey with Vừa sức', async () => {
 api.post.mockResolvedValue(ASSESSED)
 const w = mountPage(); await flushPromises(); await completeQuiz(w)
 expect(w.text()).toContain('A real C1 reading quest')
 await click(w, 'Vừa sức')
 expect(api.post).toHaveBeenCalledTimes(1)
 expect(navigateTo).toHaveBeenCalledWith('/', { replace: true })
 })
 it('calibrates through JSON, retries failure and consumes the one use only on success', async () => {
 api.post.mockResolvedValueOnce(ASSESSED).mockRejectedValueOnce(new ApiError(503, 'ai_unavailable')).mockResolvedValueOnce({ status: 'success', assessed_level: 'C2', roadmap_id: 'new-id' })
 const w = mountPage(); await flushPromises(); await completeQuiz(w)
 await click(w, 'Quá dễ')
 expect(api.post).toHaveBeenLastCalledWith('/api/v1/roadmaps/regenerate', { cefr_level: 'C2' })
 expect(w.text()).toContain('Chưa vẽ lại được')
 await click(w, 'Thử lại')
 expect(w.text()).toContain('Cấp C2')
 expect(w.text()).not.toContain('Quá dễ')
 expect(w.findComponent(CompanionSprite).props('react')).toBe('idle')
 expect(api.post).toHaveBeenCalledTimes(3)
 })
 it.each(['A1', 'C2'])('disables calibration beyond %s', async level => {
 api.post.mockResolvedValue({ ...ASSESSED, assessed_level: level })
 const w = mountPage(); await flushPromises(); await completeQuiz(w)
 const button = w.findAll('button').find(b => b.text().includes(level === 'A1' ? 'Quá khó' : 'Quá dễ'))!
 expect(button.attributes('disabled')).toBeDefined()
 })
 it('retries assessment with all ten answers intact and shows no answer feedback', async () => {
 api.post.mockRejectedValueOnce(new ApiError(502, 'ai_bad_output')).mockResolvedValueOnce(ASSESSED)
 const w = mountPage(); await flushPromises(); await completeQuiz(w)
 expect(w.text()).not.toMatch(/Trúng|Trượt|Đáp án/)
 await click(w, 'Thử lại')
 expect(api.post.mock.calls[1]![1]).toEqual(api.post.mock.calls[0]![1])
 expect(w.text()).toContain('Cấp C1')
 })
})

describe('ready fetch failure after saved assessment', () => {
 it('retries the task read without submitting assessment again', async () => {
 setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset(); navigateTo.mockReset()
 api.post.mockResolvedValue(ASSESSED)
 const w = mountPage(); await flushPromises()
 api.get.mockImplementation((path: string) => path.endsWith('/onboarding/quiz') ? Promise.resolve(QUIZ) : Promise.reject(new Error('offline')))
 await completeQuiz(w)
 expect(w.text()).toContain('Hành trình đã được lưu')
 expect(w.text()).not.toContain('A real C1 reading quest')
 expect(w.text()).not.toContain('Vào trại')
 api.get.mockResolvedValue(DAILY)
 await click(w, 'Thử lại')
 expect(api.post).toHaveBeenCalledOnce()
 expect(w.text()).toContain('A real C1 reading quest')
 })
})

describe('profile correction preserves placement', () => {
 it('keeps answers, name and time when correcting an invalid assessment request', async () => {
 setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset(); navigateTo.mockReset()
 api.post.mockRejectedValueOnce(new ApiError(400, 'invalid_request')).mockResolvedValueOnce(ASSESSED)
 const w = mountPage(); await flushPromises()
 await w.get('input[name="plant_name"]').setValue('Lá Xanh')
 await w.get('input[type="time"]').setValue('07:30')
 await completeQuiz(w)
 await click(w, 'Sửa thông tin')
 expect((w.get('input[name="plant_name"]').element as HTMLInputElement).value).toBe('Lá Xanh')
 expect((w.get('input[type="time"]').element as HTMLInputElement).value).toBe('07:30')
 await w.get('input[type="time"]').setValue('08:00')
 await click(w, 'Bắt đầu trận đầu tiên')
 expect(w.text()).toContain('Lượt 10 / 10')
 await click(w, 'Kết thúc')
 expect(api.post.mock.calls[1]![1]).toMatchObject({ notification_time: '08:00:00', plant_name: 'Lá Xanh', answers: QUIZ.questions.map(q => ({ question_id: q.id, selected_option: 'A' })) })
 })
})

describe('keeping the journey after failed calibration', () => {
 it('retries only the failed navigation read after Giữ, then enters the hub', async () => {
 setActivePinia(createPinia()); api.get.mockReset(); api.post.mockReset(); navigateTo.mockReset()
 api.post.mockResolvedValueOnce(ASSESSED).mockRejectedValue(new ApiError(503, 'ai_unavailable'))
 const w = mountPage(); await flushPromises(); await completeQuiz(w)
 await click(w, 'Quá dễ')
 expect(api.post).toHaveBeenCalledTimes(2)
 api.get.mockRejectedValue(new Error('offline'))
 await click(w, 'Giữ')
 expect(navigateTo).not.toHaveBeenCalled()
 expect(w.text()).toContain('Chưa tải được nhiệm vụ')
 expect(w.find('[role="status"]').exists()).toBe(false)
 expect(w.text()).not.toContain('Giữ trang mở')
 api.get.mockResolvedValue(DAILY)
 await click(w, 'Thử lại')
 expect(api.post).toHaveBeenCalledTimes(2)
 expect(api.get).toHaveBeenLastCalledWith('/api/v1/quests/daily?fresh=true')
 expect(navigateTo).toHaveBeenCalledWith('/', { replace: true })
 })
})
