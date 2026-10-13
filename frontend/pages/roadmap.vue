<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoadmapStore } from '~/stores/roadmap'
import { useAuthStore } from '~/stores/auth'
import type { PetStatus } from '~/stores/pet'
import { useApi } from '~/composables/useApi'
import { calendarLabel, dayState, dayStatus, TASK_LABELS, type RoadmapDay } from '~/utils/roadmap'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import MapNode from '~/components/retro/MapNode.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'

const roadmap = useRoadmapStore()
const auth = useAuthStore()
const companion = ref<PetStatus | null>(null)
let companionRequest = 0
const expanded = ref<string | null>(null)
const offline = ref(false)
const cleared = computed(() => roadmap.outline?.modules.flatMap(m => m.days).filter(d => d.is_target_met === true).length ?? 0)
const today = computed(() => roadmap.outline?.modules.flatMap(m => m.days).find(d => d.date === roadmap.outline?.today))
let active = false
let interval: ReturnType<typeof setInterval> | undefined
function connection() { offline.value = !navigator.onLine }
function refresh() {
  connection()
  if (active) void roadmap.load()
}
function visible() { if (document.visibilityState === 'visible') refresh() }
function select(day: RoadmapDay) {
  if (dayState(day, roadmap.outline!.today) === 'today') { void navigateTo('/'); return }
  expanded.value = expanded.value === day.date ? null : day.date
}
async function loadCompanion() {
  const owner = auth.user?.id; const epoch = auth.sessionEpoch; const request = ++companionRequest
  try {
    const status = await useApi().get<PetStatus>('/api/v1/pet/status')
    if (active && owner === auth.user?.id && epoch === auth.sessionEpoch && request === companionRequest) companion.value = status
  } catch { /* Map remains readable without companion statistics. */ }
}
watch(() => [auth.user?.id, auth.sessionEpoch], () => {
  expanded.value = null; companion.value = null; companionRequest++
  if (active) { refresh(); void loadCompanion() }
}, { flush: 'sync' })
watch(() => [roadmap.outline?.roadmap_id, roadmap.outline?.today], async ([roadmapId, serverToday], [previousRoadmapId, previousToday]) => {
  const available = roadmap.outline?.modules.some(module => module.days.some(day => day.date === expanded.value))
  if (roadmapId !== previousRoadmapId || serverToday !== previousToday || !available) expanded.value = today.value?.date ?? null
  await nextTick()
  if (active && today.value) document.getElementById(`region-day-${today.value.day_number}`)?.scrollIntoView?.({ block: 'center' })
})
onMounted(() => {
  active = true; refresh()
  void loadCompanion()
  window.addEventListener('focus', refresh)
  window.addEventListener('online', refresh)
  window.addEventListener('offline', connection)
  document.addEventListener('visibilitychange', visible)
  interval = setInterval(() => {
    // The learner's server timezone can differ from the browser's timezone.
    if (document.visibilityState === 'visible') refresh()
  }, 60000)
})
onBeforeUnmount(() => {
  active = false; roadmap.cancelPending(); clearInterval(interval)
  window.removeEventListener('focus', refresh); window.removeEventListener('online', refresh); window.removeEventListener('offline', connection)
  document.removeEventListener('visibilitychange', visible)
})
</script>

<template>
  <main class="retro-screen space-y-6 text-ink-0">
    <AppHeader :streak="companion?.current_streak ?? null" />
    <div class="flex items-center justify-between">
      <NuxtLink to="/" class="flex min-h-12 items-center font-display text-[22px]">‹ Trại</NuxtLink>
      <span v-if="roadmap.outline" class="border-2 border-line-lit px-3 py-2 font-display text-[22px]" aria-label="Trình độ">{{ roadmap.outline.cefr_level }}</span>
    </div>
    <header>
      <p class="font-display text-base uppercase">Hành trình 28 ngày</p>
      <h1 class="font-display text-[28px] leading-8">{{ roadmap.outline?.title ?? 'Hành trình của cậu' }}</h1>
      <p v-if="roadmap.outline" class="text-base text-ink-1">Đã xong {{ cleared }}/28 ngày</p>
    </header>
    <StateBlock v-if="roadmap.loading && !roadmap.outline" state="loading" />
    <StateBlock v-else-if="roadmap.noRoadmap" state="empty" message="Cậu chưa có hành trình." action="Bắt đầu hành trình 28 ngày" @action="navigateTo('/onboarding')" />
    <StateBlock v-if="roadmap.error" state="error" message="Không tải được hành trình. Thử lại." action="Thử lại" @action="refresh" />
    <p v-if="offline && roadmap.outline" role="status" class="text-sm text-ink-1">Đang ngoại tuyến — bản đồ là bản đã lưu.</p>
    <template v-if="roadmap.outline">
      <RetroPanel
        v-for="module in roadmap.outline.modules"
        :id="today && module.days.some(d => d.date === today?.date) ? `region-day-${today.day_number}` : undefined"
        :key="module.week"
        :speaker="`Vùng ${module.week} · ${module.title}`" :announce="false"
        :tone="module.days.some(d => d.date === roadmap.outline?.today) ? 'growth' : 'plain'"
      >
        <p class="mb-4 mt-2 text-sm text-ink-1">{{ module.focus }}</p>
        <div class="relative">
          <div v-if="module.days.every(d => dayState(d, roadmap.outline!.today) === 'locked')" aria-hidden="true" class="pointer-events-none absolute inset-0 bg-ground-2/60" />
          <ol class="map-path relative grid grid-cols-3 gap-x-4 gap-y-6" :aria-label="`Các ngày vùng ${module.week}`">
            <li v-for="(day, index) in module.days" :key="day.date" class="relative flex flex-col items-center gap-1" :style="{ gridColumn: index >= 3 && index < 6 ? 6 - index : index % 3 + 1, gridRow: Math.floor(index / 3) + 1 }">
              <MapNode :day="day.day_number" :state="dayState(day, roadmap.outline.today)" :title="`${day.title}, ${dayStatus(day, roadmap.outline.today)}`" :expanded="dayState(day, roadmap.outline.today) === 'today' ? undefined : expanded === day.date" @select="select(day)">
                <template v-if="day.date === roadmap.outline.today" #sprite>
                  <span class="absolute -top-6 left-1"><CompanionSprite :stage="companion?.stage ?? 'unknown'" :health="companion?.health_points ?? 0" :size="32" :reduced="true" /></span>
                </template>
              </MapNode>
              <span class="text-center text-sm" :class="day.is_target_met ? 'text-torch' : 'text-ink-1'">{{ dayStatus(day, roadmap.outline.today) }}</span>
              <span v-if="day.date === roadmap.outline.today" class="text-sm text-growth">Hôm nay</span>
            </li>
          </ol>
        </div>
        <template v-for="day in module.days" :key="`detail-${day.date}`">
          <section v-if="expanded === day.date" class="mt-4 border-2 border-line-lit bg-ground-2 p-3" :aria-label="`Nhiệm vụ ngày ${day.day_number}`">
            <h3 class="font-display text-[22px]">Ngày {{ day.day_number }} · {{ calendarLabel(day.date) }} <span v-if="day.date === roadmap.outline.today">· Hôm nay</span></h3>
            <p class="mb-2 text-base">{{ day.title }}</p>
            <ul class="space-y-3 text-sm">
              <li v-for="task in [...day.tasks].sort((a, b) => Object.keys(TASK_LABELS).indexOf(a.task_type) - Object.keys(TASK_LABELS).indexOf(b.task_type))" :key="task.id">
                <p>{{ TASK_LABELS[task.task_type] ?? task.task_type }} · {{ task.title }}</p>
                <p class="text-ink-1">{{ task.duration_minutes }} phút · {{ task.is_completed === true ? 'Đã xong' : task.is_completed === null ? 'Chưa có dữ liệu hoàn thành' : 'Chưa hoàn thành' }}</p>
              </li>
            </ul>
            <p class="my-3 text-sm">{{ day.minutes_spent }}/30 phút · {{ day.is_target_met ? 'Đã đạt mục tiêu' : day.date === roadmap.outline.today ? 'Đang học' : day.minutes_spent > 0 ? 'Học dở' : 'Bỏ lỡ' }}</p>
            <RetroButton v-if="day.date === roadmap.outline.today" @click="navigateTo('/')">Vào</RetroButton>
          </section>
        </template>
        <p class="mt-4 text-right font-display text-xl">{{ module.days.filter(d => d.is_target_met === true).length }}/{{ module.days.length }} ngày</p>
      </RetroPanel>
    </template>
  </main>
</template>

<style scoped>
.map-path li:not(:nth-child(3n)):not(:last-child)::after { content: ''; position: absolute; top: 22px; left: calc(50% + 24px); width: calc(100% - 32px); height: 4px; background: theme('colors.line-dim'); pointer-events: none; }
.map-path li:nth-child(n+4):nth-child(-n+5)::after { left: auto; right: calc(50% + 24px); }
.map-path li:nth-child(3)::after, .map-path li:nth-child(6)::after { content: ''; position: absolute; top: 44px; left: calc(50% - 2px); width: 4px; height: 56px; background: theme('colors.line-dim'); pointer-events: none; }
</style>
