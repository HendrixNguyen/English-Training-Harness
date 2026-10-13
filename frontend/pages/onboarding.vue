<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useOnboardingApi, type AssessmentResponse, type QuizQuestion } from '~/composables/useOnboardingApi'
import { useQuestStore } from '~/stores/quest'
import { ApiError } from '~/utils/apiClient'

const GOALS = [
  { label: 'IELTS 7.0', emoji: '🎓', value: 'IELTS 7.0 Preparation' },
  { label: 'Business English', emoji: '💼', value: 'Business English' },
]

/** Honest copy per backend error code (handler.go); answers are kept in every case. */
function assessErrorMessage(e: unknown): string {
  if (e instanceof ApiError && e.code === 'rate_limited') return 'Bạn vừa gửi quá nhiều lần. Đợi một phút rồi thử lại.'
  if (e instanceof ApiError && e.code.startsWith('ai_')) return 'Máy chủ AI đang bận, chưa chấm được bài. Thử lại sau ít phút.'
  if (e instanceof ApiError && e.code === 'invalid_request') return 'Máy chủ không nhận thông tin đã gửi. Kiểm tra lại mục tiêu, tên cây và giờ nhắc học rồi thử lại.'
  return 'Không tạo được lộ trình. Thử lại.'
}

const api = useOnboardingApi()
const quest = useQuestStore()

const step = ref<'goal' | 'quiz' | 'waiting' | 'result'>('goal')
const retryAction = ref<'assessment' | 'quests' | 'calibration' | 'navigation'>('assessment')
const calibrationUsed = ref(false)
const calibrationLevel = ref<string | null>(null)
const hatch = ref(false)
const LEVELS = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2']
const levelIndex = computed(() => LEVELS.indexOf(result.value?.assessed_level ?? ''))
const goal = ref<string | null>(null)
const time = ref('20:00')
/** <input type="time"> yields HH:MM, or '' once cleared; §6.1 wants HH:MM:SS, built in next(). */
const TIME_RE = /^(?:[01]\d|2[0-3]):[0-5]\d$/
const timeValid = computed(() => TIME_RE.test(time.value))
const plantName = ref('')
/** Any script's letters and digits, plus spaces — the design's inline rule (plant-name §1). */
const PLANT_NAME_RE = /^[\p{L}\p{N} ]{1,30}$/u
const plantNameTrimmed = computed(() => plantName.value.normalize('NFC').trim())
const plantNameValid = computed(() => plantNameTrimmed.value === '' || PLANT_NAME_RE.test(plantNameTrimmed.value))
const canStart = computed(() => goal.value !== null && timeValid.value && plantNameValid.value)
const questions = ref<QuizQuestion[]>([])
const index = ref(0)
const answers = ref<Record<string, string>>({})
const loading = ref(false)
const error = ref<string | null>(null)
const result = ref<AssessmentResponse | null>(null)

onMounted(async () => {
  if (!quest.daily && !quest.noRoadmap) await quest.load()
  if (quest.daily) await navigateTo('/', { replace: true }) // home is the dashboard once a roadmap exists
})

const current = computed(() => questions.value[index.value])
const isLast = computed(() => index.value >= questions.value.length - 1)

async function startQuiz() {
  if (!canStart.value || loading.value) return
  loading.value = true
  error.value = null
  try {
    if (!questions.value.length) questions.value = (await api.quiz()).questions
    step.value = 'quiz'
  } catch {
    error.value = typeof navigator !== 'undefined' && !navigator.onLine
      ? 'Cần mạng để bắt đầu trận đầu tiên. Kết nối lại rồi thử lại.'
      : 'Không tải được bài kiểm tra. Thử lại.'
  } finally {
    loading.value = false
  }
}

async function next() {
  if (loading.value || !current.value || !answers.value[current.value.id]) return
  if (!isLast.value) {
    index.value += 1
    return
  }
  loading.value = true
  error.value = null
  step.value = 'waiting'
  retryAction.value = 'assessment'
  try {
    result.value = await api.assess({
      target_goal: goal.value!,
      notification_time: `${time.value}:00`,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      ...(plantNameTrimmed.value ? { plant_name: plantNameTrimmed.value } : {}),
      answers: Object.entries(answers.value).map(([question_id, selected_option]) => ({ question_id, selected_option })),
    })
    hatch.value = true
    await loadReady()
  } catch (e) {
    error.value = assessErrorMessage(e) // answers are kept
    step.value = 'quiz'
  } finally {
    loading.value = false
  }
}

async function loadReady() {
  retryAction.value = 'quests'
  if (await quest.load(true)) step.value = 'result'
  else {
    step.value = 'waiting'
    error.value = 'Hành trình đã được lưu, nhưng chưa tải được nhiệm vụ. Thử lại.'
  }
}

async function calibrate(delta: number) {
  if (loading.value || calibrationUsed.value || levelIndex.value < 0) return
  const nextIndex = levelIndex.value + delta
  if (nextIndex < 0 || nextIndex >= LEVELS.length) return
  hatch.value = false // Leaving the reveal must never replay the hatch on return.
  calibrationLevel.value = LEVELS[nextIndex]!
  await regenerate()
}

async function regenerate() {
  if (loading.value || !calibrationLevel.value || calibrationUsed.value) return
  loading.value = true
  error.value = null
  step.value = 'waiting'
  retryAction.value = 'calibration'
  try {
    const response = await api.regenerate(calibrationLevel.value)
    if (result.value) result.value = { ...result.value, assessed_level: response.assessed_level, roadmap_id: response.roadmap_id }
    calibrationUsed.value = true
    await loadReady()
  } catch {
    error.value = 'Chưa vẽ lại được. Giữ hành trình hiện tại?'
  } finally { loading.value = false }
}

async function retry() {
  if (loading.value) return
  if (retryAction.value === 'assessment') return next()
  if (retryAction.value === 'calibration') return regenerate()
  if (retryAction.value === 'navigation') return finish()
  loading.value = true
  error.value = null
  try { await loadReady() } finally { loading.value = false }
}

async function finish() {
  if (loading.value) return
  retryAction.value = 'navigation' // Keeping a journey changes retries to reads, never regeneration.
  step.value = 'waiting'
  loading.value = true
  error.value = null
  try {
    if (await quest.load(true)) await navigateTo('/', { replace: true })
    else error.value = 'Chưa tải được nhiệm vụ. Thử lại.'
  } finally { loading.value = false }
}
</script>

<template>
  <main class="retro-screen mx-auto max-w-md space-y-6 px-4">
    <AppHeader />
    <RetroPanel v-if="step === 'goal'" :announce="false">
      <h1 class="font-display text-[28px] leading-8">Chọn hành trình</h1>
      <div class="mt-4 flex gap-4" role="radiogroup" aria-label="Mục tiêu">
        <GoalCard v-for="g in GOALS" :key="g.value" :label="g.label" :emoji="g.emoji" :selected="goal === g.value" @select="goal = g.value" />
      </div>
      <label class="mt-6 block">
        <span class="font-body text-sm text-ink-1">Đặt tên cho bạn đồng hành (không bắt buộc)</span>
        <input v-model="plantName" name="plant_name" type="text" maxlength="30" autocomplete="off" enterkeyhint="done" placeholder="Mầm Non" :aria-invalid="!plantNameValid || undefined" :aria-describedby="plantNameValid ? undefined : 'plant-name-note'" class="mt-2 block min-h-12 w-full rounded-sm border-2 border-line-lit bg-ground-2 px-3 font-body text-[17px] text-ink-0">
        <span v-if="!plantNameValid" id="plant-name-note" class="mt-2 block text-sm text-ember" role="note">Tên cây dài 1–30 ký tự, chỉ gồm chữ, số và dấu cách.</span>
      </label>
      <label class="mt-6 block">
        <span class="text-sm text-ink-1">Giờ nhắc học</span>
        <input v-model="time" type="time" required :aria-invalid="!timeValid || undefined" :aria-describedby="timeValid ? undefined : 'time-note'" class="mt-2 block min-h-12 w-full rounded-sm border-2 border-line-lit bg-ground-2 px-3 font-display text-xl text-ink-0 [color-scheme:dark]">
        <span v-if="!timeValid" id="time-note" class="mt-2 block text-sm text-ember" role="note">Chọn một giờ nhắc học để tiếp tục.</span>
      </label>
    </RetroPanel>

    <template v-else-if="step === 'quiz'">
      <SpeechBox v-if="index === 0" :announce="false" :name="plantNameTrimmed || 'Mầm Non'" stage="seed" line="Tớ đang xem cậu đánh… cứ chọn theo cảm giác." />
      <RetroPanel :announce="false" :speaker="plantNameTrimmed || 'Mầm Non'">
        <StateBlock v-if="questions.length === 0" state="empty" message="Chưa có trận đầu tiên. Quay lại sau." action="Về trang chính" @action="navigateTo('/')" />
        <template v-else-if="current">
          <CompanionSprite stage="seed" :health="100" :size="64" />
          <p class="font-display text-xl text-ink-1">Lượt {{ index + 1 }} / {{ questions.length }}</p>
          <p class="mt-4 font-body text-[17px] leading-[26px]">{{ current.prompt }}</p>
          <div class="mt-4 space-y-2" role="radiogroup" :aria-label="current.prompt">
            <button v-for="(text, key) in current.options" :key="key" type="button" role="radio" :aria-checked="answers[current.id] === key" class="block min-h-14 w-full rounded-sm border-2 px-4 py-3 text-left font-body text-[17px]" :class="answers[current.id] === key ? 'border-growth bg-ground-2' : 'border-line-lit bg-ground-1'" @click="answers[current.id] = String(key)">
              <span aria-hidden="true">{{ answers[current.id] === key ? '▶' : '○' }}</span> ({{ key }}) {{ text }}
            </button>
          </div>
        </template>
      </RetroPanel>
    </template>

    <RetroPanel v-else-if="step === 'waiting'" :announce="false" :speaker="plantNameTrimmed || 'Mầm Non'">
      <CompanionSprite stage="seed" :health="100" />
      <p v-if="loading" role="status" class="mt-4 font-body text-[17px]">{{ retryAction === 'calibration' ? 'Tớ đang vẽ lại bản đồ cho cậu…' : result ? 'Tớ đang chuẩn bị nhiệm vụ cho cậu…' : 'Tớ đang chấm bài và vẽ bản đồ cho cậu…' }} Thời gian tạo hành trình có thể kéo dài. Giữ trang mở cho đến khi có kết quả.</p>
      <p v-if="loading" aria-hidden="true" class="retro-dots mt-4 font-display text-[28px]">…</p>
    </RetroPanel>

    <template v-else-if="result">
      <RetroPanel :announce="false" :speaker="result.pet_state.plant_name" tone="torch" class="text-center">
        <CompanionSprite :stage="result.pet_state.stage" :health="result.pet_state.health_points" :name="result.pet_state.plant_name" :react="hatch ? 'levelup' : 'idle'" @reacted="hatch = false" />
        <h1 class="mt-4 font-display text-[34px] leading-9 text-torch">Cấp {{ result.assessed_level }}</h1>
        <p class="mt-4 font-body text-[17px]">Tớ nở rồi! Cậu ở cấp {{ result.assessed_level }} — hành trình {{ goal }}.</p>
      </RetroPanel>
      <h2 class="font-display text-[28px]">Ngày {{ quest.daily?.day_number }}</h2>
      <ul class="space-y-2">
        <QuestNode v-for="(task, taskIndex) in quest.daily?.tasks" :key="task.id" :task="task" :index="taskIndex" :state="task.is_completed ? 'done' : 'open'" connector="none" @enter="finish" />
      </ul>
      <RetroPanel v-if="!calibrationUsed" :announce="false">
        <h2 class="mb-4 font-body text-[17px]">Ngày 1 trông thế nào?</h2>
        <div class="flex flex-wrap gap-2">
          <RetroButton variant="secondary" :disabled="loading || levelIndex < 0 || levelIndex >= LEVELS.length - 1" @click="calibrate(1)">Quá dễ</RetroButton>
          <RetroButton variant="secondary" :loading="loading" @click="finish">Vừa sức</RetroButton>
          <RetroButton variant="secondary" :disabled="loading || levelIndex <= 0" @click="calibrate(-1)">Quá khó</RetroButton>
        </div>
      </RetroPanel>
    </template>

    <RetroPanel v-if="error" :announce="false" tone="ember">
      <p role="alert" class="font-body text-[17px]">{{ error }}</p>
      <div class="mt-4 flex flex-wrap gap-4">
        <RetroButton v-if="step === 'quiz' || step === 'waiting'" variant="secondary" :loading="loading" @click="retry">Thử lại</RetroButton>
        <RetroButton v-if="step === 'quiz'" variant="secondary" @click="step = 'goal'">Sửa thông tin</RetroButton>
        <RetroButton v-if="retryAction === 'calibration' && result" :loading="loading" @click="finish">Giữ</RetroButton>
      </div>
    </RetroPanel>
    <div v-if="step !== 'waiting'" class="retro-bottom-bar">
      <RetroButton v-if="step === 'goal'" block :disabled="!canStart" :loading="loading" @click="startQuiz">Bắt đầu trận đầu tiên</RetroButton>
      <RetroButton v-else-if="step === 'quiz' && current" block :disabled="!answers[current.id]" :loading="loading" @click="next">{{ isLast ? 'Kết thúc' : 'Chọn' }}</RetroButton>
      <RetroButton v-else-if="step === 'result'" block :loading="loading" @click="finish">Vào trại →</RetroButton>
    </div>
  </main>
</template>
