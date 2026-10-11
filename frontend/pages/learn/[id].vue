<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuthStore } from '~/stores/auth'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { ApiError } from '~/utils/apiClient'
import { classifyContent, hasUsableContent } from '~/utils/content'

const route = useRoute()
const quest = useQuestStore()
const pet = usePetStore()
const auth = useAuthStore()
const account = computed(() => `${auth.user?.id ?? ''}:${auth.accessToken ?? ''}`)

const id = computed(() => String(route.params.id))
const task = computed(() => quest.taskById(id.value))
const content = computed(() => classifyContent(task.value?.content_json))
const timer = computed(() => quest.timers[id.value])
const remaining = computed(() => quest.remainingSeconds(id.value))
const answers = ref<Record<string, string>>({})
const finished = ref(false)
const posting = ref(false)
const error = ref<string | null>(null)
const hydrating = ref(true)
const hydrationError = ref<string | null>(null)
const ready = computed(() => !hydrating.value && !hydrationError.value && hasUsableContent(task.value?.content_json))
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)

let interval: ReturnType<typeof setInterval> | null = null

function sync() {
  quest.tick()
}

let generation = 0
let disposed = false

async function openTask() {
  const request = ++generation
  const requestedId = id.value
  const requestedAccount = account.value
  const current = () => !disposed && request === generation && id.value === requestedId && account.value === requestedAccount
  hydrating.value = true
  hydrationError.value = null
  quest.resetForAccount()
  try {
    if (!quest.daily && !quest.noRoadmap) await quest.load()
    if (!current()) return
    const requested = quest.taskById(requestedId)
    if (requested && !requested.is_completed) {
      if (!hasUsableContent(requested.content_json) && !online.value) throw new Error('offline')
      await quest.hydrateTask(requestedId, current)
      if (!current()) return
      const hydrated = quest.taskById(requestedId)
      if (hydrated && !hydrated.is_completed) {
        if (!hasUsableContent(hydrated.content_json)) throw new ApiError(502, 'ai_bad_output')
        quest.startTimer(hydrated.id, hydrated.duration_minutes || 10)
        quest.tick()
      }
    }
  } catch (e) {
    if (!current()) return
    hydrationError.value = !online.value ? 'Cần kết nối để tải nội dung. Kết nối rồi thử lại.'
      : e instanceof ApiError && e.code === 'exercise_not_found' ? 'Nhiệm vụ này không còn khả dụng. Về trang chính để tải lại.'
        : 'Chưa tải được nội dung học. Thử lại.'
  } finally {
    if (current()) hydrating.value = false
  }
}

onMounted(() => {
  watch([id, account], () => {
    answers.value = {}
    finished.value = false
    posting.value = false
    error.value = null
    void openTask()
  }, { immediate: true })
  interval = setInterval(() => quest.tick(), 1000)
  // A resumed tab snaps to the persisted wall-clock anchor immediately.
  document.addEventListener('visibilitychange', sync)
  window.addEventListener('focus', sync)
  window.addEventListener('online', setOnline)
  window.addEventListener('offline', setOnline)
})

onBeforeUnmount(() => {
  disposed = true
  generation += 1
  if (interval) clearInterval(interval) // the store keeps the wall-clock anchor (and aelp.timers keeps it across reloads), so re-entry resumes
  document.removeEventListener('visibilitychange', sync)
  window.removeEventListener('focus', sync)
  window.removeEventListener('online', setOnline)
  window.removeEventListener('offline', setOnline)
})

function setOnline() {
  online.value = navigator.onLine
}

const buttonLabel = computed(() => {
  if (task.value?.is_completed) return 'Đã hoàn thành'
  if (timer.value && remaining.value === 0) return 'Hết giờ — Hoàn thành'
  return 'Hoàn thành'
})

async function complete() {
  if (!task.value || posting.value || !ready.value || !online.value || task.value.is_completed || (!finished.value && (!timer.value || remaining.value !== 0))) return
  const request = generation
  const exerciseId = task.value.id
  const requestedAccount = account.value
  const current = () => !disposed && generation === request && id.value === exerciseId && account.value === requestedAccount
  posting.value = true
  error.value = null
  try {
    const res = await quest.complete(exerciseId, quest.elapsedSeconds(exerciseId, Date.now()), { ...answers.value })
    if (!current()) return
    pet.applyProgress(res)
    await navigateTo('/', { replace: true })
  } catch (e) {
    if (!current()) return
    error.value = e instanceof ApiError && e.code === 'exercise_not_found'
      ? 'Nhiệm vụ này không còn trong hôm nay. Về trang chính để tải lại.'
      : 'Chưa ghi được tiến độ. Thử lại.'
  } finally {
    if (current()) posting.value = false
  }
}
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-md flex-col px-4 pb-8">
    <div class="flex items-center justify-between py-3 text-sm">
      <NuxtLink to="/" class="text-mute hover:underline">‹ Quay lại</NuxtLink>
      <CountdownTimer v-if="timer" :remaining-seconds="remaining" />
    </div>

    <AppCard class="flex-1">
      <StateBlock v-if="hydrating" state="loading" />
      <StateBlock v-else-if="hydrationError || (quest.error && !quest.daily)" state="error" :message="hydrationError || 'Chưa tải được nhiệm vụ. Thử lại.'" action="Thử lại" @action="openTask" />
      <StateBlock v-else-if="!task" state="empty" message="Nhiệm vụ này không có trong hôm nay." action="Về trang chính" @action="navigateTo('/')" />
      <template v-else>
        <ContentViewer
          v-if="ready && !finished && !task.is_completed"
          :key="id"
          :content="content"
          :title="task.title"
          @answer="(qid, opt) => { answers[qid] = opt }"
          @finished="finished = true"
        />
        <div v-else class="space-y-4">
          <h2 class="font-display text-2xl">
            {{ task.title }}
          </h2>
          <p v-if="task.is_completed" class="text-ink dark:text-growth">
            ✓ Đã hoàn thành
          </p>
          <p v-else class="text-mute">
            Bạn đã xem hết nội dung. Ghi lại thời gian học để tưới cây.
          </p>
        </div>
      </template>
    </AppCard>

    <p v-if="error" class="mt-3 rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
      {{ error }}
    </p>
    <p v-if="!online" class="mt-3 text-center text-sm text-mute">
      Cần kết nối để ghi tiến độ
    </p>

    <AppButton
      v-if="task"
      class="mt-4"
      block
      :loading="posting"
      :disabled="!ready || task.is_completed || !online || (!finished && (!timer || remaining !== 0))"
      @click="complete"
    >
      {{ buttonLabel }}
    </AppButton>
  </main>
</template>
