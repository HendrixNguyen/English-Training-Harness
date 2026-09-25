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
  if (e instanceof ApiError && e.code === 'invalid_request') return 'Máy chủ không nhận thông tin đã gửi. Kiểm tra lại mục tiêu và giờ nhắc học rồi thử lại.'
  return 'Không tạo được lộ trình. Thử lại.'
}

const api = useOnboardingApi()
const quest = useQuestStore()

const step = ref<'goal' | 'quiz' | 'result'>('goal')
const goal = ref<string | null>(null)
const time = ref('20:00')
/** <input type="time"> yields HH:MM, or '' once cleared; §6.1 wants HH:MM:SS, built in next(). */
const TIME_RE = /^\d{2}:\d{2}$/
const timeValid = computed(() => TIME_RE.test(time.value))
const canStart = computed(() => goal.value !== null && timeValid.value)
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
  if (!canStart.value) return
  loading.value = true
  error.value = null
  try {
    questions.value = (await api.quiz()).questions
    step.value = 'quiz'
  } catch {
    error.value = 'Không tải được bài kiểm tra. Thử lại.'
  } finally {
    loading.value = false
  }
}

async function next() {
  if (!isLast.value) {
    index.value += 1
    return
  }
  loading.value = true
  error.value = null
  try {
    result.value = await api.assess({
      target_goal: goal.value!,
      notification_time: `${time.value}:00`,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      answers: Object.entries(answers.value).map(([question_id, selected_option]) => ({ question_id, selected_option })),
    })
    step.value = 'result'
  } catch (e) {
    error.value = assessErrorMessage(e) // answers are kept
  } finally {
    loading.value = false
  }
}

async function finish() {
  await quest.load()
  await navigateTo('/', { replace: true })
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader />

    <AppCard v-if="step === 'goal'" class="mt-2">
      <h1 class="font-display text-2xl">
        Mục tiêu học của bạn là gì?
      </h1>
      <div class="mt-4 flex gap-3" role="radiogroup" aria-label="Mục tiêu">
        <GoalCard v-for="g in GOALS" :key="g.value" :label="g.label" :emoji="g.emoji" :selected="goal === g.value" @select="goal = g.value" />
      </div>
      <label class="mt-6 block">
        <span class="text-sm text-mute">Chọn giờ nhắc học hằng ngày</span>
        <input v-model="time" type="time" required :aria-invalid="!timeValid || undefined" class="mt-1 block w-full rounded-btn border border-ink/15 bg-transparent px-3 py-2 dark:border-paper/15">
        <span v-if="!timeValid" class="mt-1 block text-sm text-alert" role="note">Chọn một giờ nhắc học để tiếp tục.</span>
      </label>
      <AppButton class="mt-6" block :disabled="!canStart" :loading="loading" @click="startQuiz">
        Bắt đầu bài kiểm tra đầu vào
      </AppButton>
    </AppCard>

    <AppCard v-else-if="step === 'quiz'" class="mt-2">
      <StateBlock v-if="questions.length === 0" state="empty" message="Chưa có bài kiểm tra. Quay lại sau." action="Về trang chính" @action="navigateTo('/')" />
      <template v-else-if="current">
        <p class="text-sm text-mute">
          Câu {{ index + 1 }} / {{ questions.length }}
        </p>
        <p class="mt-2 text-lg">
          "{{ current.prompt }}"
        </p>
        <div class="mt-4 space-y-2" role="radiogroup">
          <button
            v-for="(text, key) in current.options"
            :key="key"
            type="button"
            role="radio"
            :aria-checked="answers[current.id] === key"
            class="block w-full rounded-btn border px-4 py-3 text-left"
            :class="answers[current.id] === key ? 'border-growth bg-growth/10' : 'border-ink/15 dark:border-paper/15'"
            @click="answers[current.id] = String(key)"
          >
            ({{ key }}) {{ text }}
          </button>
        </div>
        <AppButton class="mt-6" block :disabled="!answers[current.id]" :loading="loading" @click="next">
          {{ isLast ? 'Hoàn thành' : 'Tiếp tục' }}
        </AppButton>
        <p v-if="loading && isLast" class="mt-3 text-center text-sm text-mute" role="status">
          Đang chấm bài và soạn lộ trình 28 ngày — thường mất 1–2 phút. Đừng đóng trang.
        </p>
      </template>
    </AppCard>

    <AppCard v-else-if="result" class="mt-2 text-center">
      <PlantSvg :stage="result.pet_state.stage" :health="result.pet_state.health_points" />
      <p class="mt-3 font-display text-2xl">
        Trình độ của bạn: {{ result.assessed_level }}
      </p>
      <p class="text-mute">
        {{ result.pet_state.plant_name }} đã nảy mầm. Tưới cây bằng 30 phút học mỗi ngày.
      </p>
      <AppButton class="mt-4" block @click="finish">
        Xem nhiệm vụ hôm nay
      </AppButton>
    </AppCard>

    <p v-if="error" class="mt-3 rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
      {{ error }}
    </p>
  </main>
</template>
