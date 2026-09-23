<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { ApiError } from '~/utils/apiClient'
import { classifyContent } from '~/utils/content'

const route = useRoute()
const quest = useQuestStore()
const pet = usePetStore()

const id = computed(() => String(route.params.id))
const task = computed(() => quest.taskById(id.value))
const content = computed(() => classifyContent(task.value?.content_json))
const timer = computed(() => quest.timers[id.value])
const answers = ref<Record<string, string>>({})
const finished = ref(false)
const posting = ref(false)
const error = ref<string | null>(null)
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)

let interval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  if (!quest.daily && !quest.noRoadmap) await quest.load()
  if (task.value && !task.value.is_completed) {
    quest.startTimer(task.value.id, task.value.duration_minutes || 10)
    interval = setInterval(() => quest.tick(task.value!.id), 1000)
  }
  window.addEventListener('online', setOnline)
  window.addEventListener('offline', setOnline)
})

onBeforeUnmount(() => {
  if (interval) clearInterval(interval) // the store keeps remainingSeconds, so re-entry resumes
  window.removeEventListener('online', setOnline)
  window.removeEventListener('offline', setOnline)
})

function setOnline() {
  online.value = navigator.onLine
}

const buttonLabel = computed(() => {
  if (task.value?.is_completed) return 'Đã hoàn thành'
  if (timer.value?.remainingSeconds === 0) return 'Hết giờ — Hoàn thành'
  return 'Hoàn thành'
})

async function complete() {
  if (!task.value || posting.value) return
  posting.value = true
  error.value = null
  try {
    const res = await quest.complete(task.value.id, quest.elapsedSeconds(task.value.id), answers.value)
    pet.applyProgress(res)
    await navigateTo('/', { replace: true })
  } catch (e) {
    error.value = e instanceof ApiError && e.code === 'exercise_not_found'
      ? 'Nhiệm vụ này không còn trong hôm nay. Về trang chính để tải lại.'
      : 'Chưa ghi được tiến độ. Thử lại.'
  } finally {
    posting.value = false
  }
}
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-md flex-col px-4 pb-8">
    <div class="flex items-center justify-between py-3 text-sm">
      <NuxtLink to="/" class="text-mute hover:underline">‹ Quay lại</NuxtLink>
      <CountdownTimer v-if="timer" :remaining-seconds="timer.remainingSeconds" />
    </div>

    <AppCard class="flex-1">
      <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
      <StateBlock v-else-if="!task" state="empty" message="Nhiệm vụ này không có trong hôm nay." action="Về trang chính" @action="navigateTo('/')" />
      <template v-else>
        <ContentViewer
          v-if="!finished && !task.is_completed"
          :content="content"
          :title="task.title"
          @answer="(qid, opt) => { answers[qid] = opt }"
          @finished="finished = true"
        />
        <div v-else class="space-y-4">
          <h2 class="font-display text-2xl">
            {{ task.title }}
          </h2>
          <p v-if="task.is_completed" class="text-growth">
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
      :disabled="task.is_completed || !online || (!finished && timer?.remainingSeconds !== 0)"
      @click="complete"
    >
      {{ buttonLabel }}
    </AppButton>
  </main>
</template>
