<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuthStore } from '~/stores/auth'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { ApiError } from '~/utils/apiClient'
import { classifyContent, hasTrustedContent, questionsOf } from '~/utils/content'
import { clearDraft, clearOwnerDrafts, clearTaskDrafts, initialSession, readDraft, validateSession, writeDraft, type DraftScope, type LearningSession } from '~/utils/learningSession'
import { clampDuration } from '~/utils/progress'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'

const route = useRoute()
const quest = useQuestStore()
const pet = usePetStore()
const auth = useAuthStore()
const owner = computed(() => auth.user?.id ?? '')
const account = computed(() => owner.value || `anonymous:${auth.accessToken ?? ''}`)
const id = computed(() => String(route.params.id))
const task = computed(() => quest.taskById(id.value))
const content = computed(() => classifyContent(task.value?.content_json, task.value?.task_type))
const questions = computed(() => questionsOf(content.value))
const timer = computed(() => quest.timers[id.value])
const remaining = computed(() => quest.remainingSeconds(id.value))
const session = ref<LearningSession>(initialSession(content.value))
const scope = ref<DraftScope | null>(null)
const posting = ref(false)
const checking = ref(false)
const error = ref<string | null>(null)
const notice = ref<string | null>(null)
const hydrating = ref(true)
const hydrationError = ref<string | null>(null)
const invalid = ref(false)
const memoryOnly = ref(false)
const summaryHeading = ref<HTMLElement | null>(null)
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)
const saved = computed(() => !!task.value?.is_completed && !session.value.uncertain)
const ready = computed(() => !hydrating.value && !hydrationError.value && (saved.value || (hasTrustedContent(task.value) && scope.value?.owner === owner.value && scope.value?.task === id.value && scope.value?.revision === task.value?.content_revision)))
const summary = computed(() => saved.value || session.value.phase === 'summary')
const result = computed(() => saved.value ? task.value?.quiz_result : null)
const answered = computed(() => saved.value ? result.value?.answered_count ?? null : Object.keys(session.value.answers).length)
const questionCount = computed(() => saved.value ? result.value?.question_count ?? null : questions.value.length)
const duration = computed(() => saved.value ? result.value?.duration_seconds ?? null : session.value.duration ?? quest.elapsedSeconds(id.value))
const timeText = computed(() => duration.value === null ? 'Thời gian học: chưa có dữ liệu' : `Thời gian học: ${Math.floor(duration.value / 60)} phút ${duration.value % 60} giây`)
let interval: ReturnType<typeof setInterval> | null = null
let generation = 0
let disposed = false
function sync() { quest.tick() }
function setOnline() { online.value = navigator.onLine }
function persist() { if (scope.value && !saved.value) memoryOnly.value = !writeDraft(scope.value, session.value) }
function changeSession(value: LearningSession) {
  const valid = validateSession(value, content.value)
  if (!valid || !ready.value || saved.value) return
  session.value = valid; persist()
  if (valid.phase === 'summary') void nextTick(() => summaryHeading.value?.focus())
}
async function openTask(repair = false) {
  const request = ++generation; const requestedId = id.value; const requestedAccount = account.value
  const current = () => !disposed && request === generation && id.value === requestedId && account.value === requestedAccount
  hydrating.value = true; hydrationError.value = null; invalid.value = false
  quest.resetForAccount()
  try {
    if (!quest.daily && !quest.noRoadmap) await quest.load()
    if (!current()) return
    const requested = quest.taskById(requestedId)
    if (requested && !requested.is_completed) {
      if (!hasTrustedContent(requested) && !online.value) throw new Error('offline')
      await quest.hydrateTask(requestedId, current, repair)
      if (!current()) return
    }
    const hydrated = quest.taskById(requestedId)
    if (!hydrated) return
    if (hydrated.is_completed) {
      session.value = { ...initialSession(content.value), phase: 'summary' }
      if (hydrated.content_revision) { const key = { owner: owner.value, task: hydrated.id, revision: hydrated.content_revision }; clearTaskDrafts(key); clearDraft(key) }
      scope.value = null
    } else {
      if (!hasTrustedContent(hydrated) || !hydrated.content_revision) throw new ApiError(502, 'ai_bad_output')
      const key = { owner: owner.value, task: hydrated.id, revision: hydrated.content_revision }
      clearTaskDrafts(key)
      scope.value = key
      session.value = readDraft(key, content.value) ?? initialSession(content.value)
      quest.startTimer(hydrated.id, hydrated.duration_minutes || 10); quest.tick()
      persist()
    }
  } catch (e) {
    if (!current()) return
    invalid.value = e instanceof ApiError && e.code === 'exercise_content_invalid'
    hydrationError.value = !online.value ? 'Cần kết nối để tải nội dung. Kết nối rồi thử lại.'
      : invalid.value ? 'Nội dung này cần được tạo lại.'
        : repair ? e instanceof ApiError && e.code === 'ai_bad_output' ? 'Chưa tạo được nội dung phù hợp. Quay lại sau.' : 'Chưa tạo lại được nội dung. Thử lại.'
          : e instanceof ApiError && e.code === 'exercise_not_found' ? 'Nhiệm vụ này không còn khả dụng.' : 'Chưa tải được nội dung học. Thử lại.'
    // Repair is always an explicit action, including subsequent failures.
    if (repair) invalid.value = true
  } finally { if (current()) hydrating.value = false }
  if (current() && session.value.uncertain && !hydrationError.value) await reconcile()
}
onMounted(() => {
  watch([id, account], (_value, previous) => {
    if (previous && previous[1] !== account.value && scope.value) clearOwnerDrafts(scope.value.owner)
    scope.value = null; session.value = initialSession(content.value)
    posting.value = false; checking.value = false; error.value = null; notice.value = null; memoryOnly.value = false
    void openTask()
  }, { immediate: true })
  watch(() => task.value?.content_revision, (revision, previous) => {
    if (!hydrating.value && scope.value && revision !== scope.value.revision && revision !== previous) {
      clearDraft(scope.value); notice.value = 'Nội dung đã thay đổi. Học lại từ đầu.'
      scope.value = null; posting.value = false; checking.value = false; error.value = null
      void openTask()
    }
  })
  interval = setInterval(() => quest.tick(), 1000)
  document.addEventListener('visibilitychange', sync); window.addEventListener('focus', sync)
  window.addEventListener('online', setOnline); window.addEventListener('offline', setOnline)
})
onBeforeUnmount(() => {
  disposed = true; generation += 1
  if (interval) clearInterval(interval)
  document.removeEventListener('visibilitychange', sync); window.removeEventListener('focus', sync)
  window.removeEventListener('online', setOnline); window.removeEventListener('offline', setOnline)
})
function requestGuard(matchRevision = true) {
  const request = generation; const key = scope.value && { ...scope.value }; const requestedAccount = account.value
  return () => !disposed && generation === request && account.value === requestedAccount && id.value === key?.task && (!matchRevision || task.value?.content_revision === key?.revision)
}
async function reconcile() {
  if (checking.value || posting.value || !scope.value || !session.value.uncertain || !online.value) return
  const current = requestGuard(); const key = { ...scope.value }; const answers = { ...session.value.answers }
  checking.value = true; error.value = 'Đang kiểm tra kết quả đã lưu…'
  try {
    const fresh = await quest.load(true)
    if (!current()) return
    const latest = quest.taskById(key.task)
    if (!fresh || !latest) throw new Error('unknown')
    if (latest.is_completed) {
      const stored = latest.quiz_result
      if (!stored || stored.question_count !== questions.value.length || stored.answered_count !== Object.keys(answers).length || stored.answers.length !== Object.keys(answers).length || stored.answers.some(a => answers[a.question_id] !== a.selected_option)) throw new Error('unknown')
      session.value.uncertain = false; clearDraft(key); error.value = null
    } else { session.value.uncertain = false; error.value = 'Chưa ghi được tiến độ. Thử lại.'; persist() }
  } catch {
    if (current()) { error.value = 'Chưa xác nhận được kết quả. Kết nối rồi kiểm tra lại.'; persist() }
  } finally { if (current()) checking.value = false }
}
async function complete() {
  if (!task.value || !scope.value || !ready.value || !online.value || posting.value || checking.value || saved.value || task.value.is_completed || session.value.uncertain || session.value.phase !== 'summary' || !validateSession(session.value, content.value)) return
  const current = requestGuard(); const key = { ...scope.value }
  session.value.duration ??= clampDuration(quest.elapsedSeconds(key.task, Date.now()))
  session.value.uncertain = true; persist(); posting.value = true; error.value = null
  let ambiguous = false
  try {
    const res = await quest.complete(key.task, session.value.duration, { ...session.value.answers }, key.revision, current)
    if (!current()) return
    if (!res.quiz_result || !task.value?.is_completed) { ambiguous = true; return }
    session.value.uncertain = false; clearDraft(key); pet.applyProgress(res)
    await nextTick(); summaryHeading.value?.focus()
  } catch (e) {
    if (!current()) return
    if (e instanceof ApiError && ['exercise_content_changed', 'exercise_content_invalid'].includes(e.code)) {
      // Recovery expects a new revision but must stay within this page lifetime.
      const recoveryCurrent = requestGuard(false)
      clearDraft(key); scope.value = null; posting.value = false; notice.value = 'Nội dung đã thay đổi. Học lại từ đầu.'
      const refreshed = await quest.load(true)
      if (!recoveryCurrent()) return
      if (refreshed) await openTask()
      else { hydrationError.value = 'Chưa tải được nội dung học. Thử lại.'; session.value = initialSession(content.value) }
    } else if (e instanceof ApiError && e.status >= 400 && e.status < 500) {
      session.value.uncertain = false; error.value = e.code === 'exercise_not_found' ? 'Nhiệm vụ này không còn khả dụng.' : 'Chưa ghi được tiến độ. Thử lại.'; persist()
    } else ambiguous = true
  } finally {
    if (current()) { posting.value = false; if (ambiguous) await reconcile() }
  }
}
function answerPrompt(questionId: string) { return questions.value.find(q => q.id === questionId)?.prompt ?? questionId }
function optionText(questionId: string, option: string) { return questions.value.find(q => q.id === questionId)?.options[option] ?? '' }
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-md flex-col px-4 pb-8">
    <div class="flex items-center justify-between py-3 text-sm">
      <NuxtLink to="/" class="text-mute hover:underline">‹ Quay lại</NuxtLink>
      <CountdownTimer v-if="timer" :remaining-seconds="remaining" />
    </div>
    <AppCard class="flex-1">
      <StateBlock v-if="hydrating" state="loading" message="Đang tải nội dung…" />
      <StateBlock v-else-if="hydrationError || (quest.error && !quest.daily)" state="error" :message="hydrationError || 'Chưa tải được nhiệm vụ. Thử lại.'" :action="invalid ? 'Tạo lại nội dung' : 'Thử lại'" @action="openTask(invalid)" />
      <StateBlock v-else-if="!task" state="empty" message="Nhiệm vụ này không còn khả dụng." action="Về trang chính" @action="navigateTo('/')" />
      <template v-else-if="ready">
        <ContentViewer v-if="!summary" :key="`${id}:${task.content_revision}`" :content="content" :title="task.title" :session="session" @session-change="changeSession" />
        <RetroPanel v-else>
          <div class="space-y-4 font-body text-[17px] leading-[26px]">
            <h2 ref="summaryHeading" tabindex="-1" class="font-bold">{{ task.title }}</h2>
            <p>{{ saved ? 'Đã lưu kết quả' : 'Chưa lưu kết quả' }}</p>
            <p v-if="answered !== null">Đã trả lời {{ answered }} / {{ questionCount }}</p>
            <p>{{ timeText }}</p>
            <p v-if="result?.status === 'graded'">Đúng {{ result.correct_count }} / {{ result.question_count }} · {{ result.score_percent }}%</p>
            <p v-else-if="result || (!saved && questions.length === 0)">Không chấm điểm</p>
            <p v-else-if="saved">Không có kết quả câu hỏi đã lưu</p>
            <ol v-if="result" class="space-y-4">
              <li v-for="answer in result.answers" :key="answer.question_id" class="border-t border-line-lit pt-3">
                <p>{{ answerPrompt(answer.question_id) }}</p>
                <p>Bạn chọn: {{ answer.selected_option }} {{ optionText(answer.question_id, answer.selected_option) }}</p>
                <p v-if="answer.correct_option !== null">Đáp án đúng: {{ answer.correct_option }} {{ optionText(answer.question_id, answer.correct_option) }}</p>
                <p v-if="answer.is_correct !== null">{{ answer.is_correct ? '✔ Trúng!' : 'Trượt…' }}</p>
                <p v-if="answer.explanation !== null">{{ answer.explanation }}</p>
              </li>
            </ol>
          </div>
        </RetroPanel>
      </template>
    </AppCard>
    <p v-if="notice" class="mt-3 font-body text-sm" role="status">{{ notice }}</p>
    <p v-if="error" class="mt-3 border border-ember bg-ground-1 px-4 py-3 font-body text-sm text-ink-0" role="status">{{ error }}</p>
    <p v-if="memoryOnly && !saved" class="mt-3 font-body text-sm">Tiến độ chưa lưu chỉ giữ trong lần mở này.</p>
    <p v-if="!online" class="mt-3 text-center text-sm text-mute">Cần kết nối để ghi tiến độ</p>
    <RetroButton v-if="ready && summary && saved" class="mt-4" block @click="navigateTo('/')">Về trang chính</RetroButton>
    <RetroButton v-else-if="ready && summary && session.uncertain && !posting" class="mt-4" block :loading="checking" :disabled="!online" @click="reconcile">Kiểm tra lại</RetroButton>
    <RetroButton v-else-if="ready && summary" class="mt-4" block :loading="posting" :disabled="!online || checking || task?.is_completed" @click="complete">Hoàn thành</RetroButton>
  </main>
</template>
