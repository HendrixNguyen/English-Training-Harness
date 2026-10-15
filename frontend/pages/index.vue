<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import AppHeader from '~/components/AppHeader.vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import HpBar from '~/components/retro/HpBar.vue'
import DayBar from '~/components/retro/DayBar.vue'
import QuestNode from '~/components/retro/QuestNode.vue'
import SpeechBox from '~/components/retro/SpeechBox.vue'
import RetroToast from '~/components/retro/RetroToast.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { useReducedMotion } from '~/composables/useReducedMotion'
import { useRetroToast } from '~/composables/useRetroToast'
import { chipsFor, useGrowthMoment } from '~/composables/useGrowthMoment'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'

const quest = useQuestStore()
const pet = usePetStore()
const reduced = useReducedMotion()
const { start, active, chips, reaction, pulse, displayHealth, displayStage } = useGrowthMoment()
// Consume before first paint: the store already holds the saved after-values.
const savedDelta = pet.consumeDelta()
start(savedDelta)
const earnedWords = savedDelta ? chipsFor(savedDelta).map(chip => chip.text) : []
const toast = useRetroToast()
const offlineNotified = ref(false)
function notifyOffline() {
  if (!navigator.onLine && !offlineNotified.value) {
    offlineNotified.value = true
    toast.show('Đang ngoại tuyến — tớ nhớ tiến độ giúp cậu.')
  }
}
onMounted(() => {
  void Promise.all([pet.load(), quest.load()])
  notifyOffline()
  window.addEventListener('offline', notifyOffline)
})
onBeforeUnmount(() => window.removeEventListener('offline', notifyOffline))
const name = computed(() => pet.status?.plant_name || 'Cây xanh')
const bubble = computed(() => {
  if (!pet.status || pet.isWilted) return ''
  if (quest.targetMet) return 'Hôm nay tớ đủ nước rồi, cảm ơn cậu.'
  if (quest.daily?.day_number === 1 && quest.accumulatedSeconds === 0) return `Chào cậu. Tớ là ${name.value} — cùng đi phòng đầu tiên nhé?`
  if (pet.status.health_points >= 60) return 'Tưới cho tớ 10 phút đi, cậu.'
  if (pet.status.health_points >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return 'Tớ sắp héo mất. Học một chút nhé?'
})
function rowState(taskId: string, completed: boolean): 'done' | 'current' | 'locked' | 'open' {
  if (completed) return 'done'
  if (quest.targetMet) return 'open'
  return taskId === quest.nextTaskId ? 'current' : 'locked'
}
const nextIndex = computed(() => quest.sortedTasks.findIndex(task => task.id === quest.nextTaskId) + 1)
const earnedConnector = computed(() => {
  if (!active.value || reduced.value) return -1
  const next = quest.sortedTasks.findIndex(task => task.id === quest.nextTaskId)
  const index = next < 0 ? quest.sortedTasks.length - 2 : next - 1
  return index >= 0 && quest.sortedTasks[index]?.is_completed ? index : -1
})
const announcement = computed(() => [bubble.value, ...earnedWords].filter(Boolean).join(' '))
const canNavigate = computed(() => !!quest.daily && !quest.loading && !quest.noRoadmap && quest.sortedTasks.length > 0)
function enter(id: string) { void navigateTo(`/learn/${id}`) }
function next() { if (quest.nextTaskId) enter(quest.nextTaskId); else void navigateTo('/roadmap') }
</script>

<template>
  <main class="retro-screen mx-auto max-w-md px-4">
    <AppHeader :streak="pet.status?.current_streak ?? null" :pulse="pulse" />
    <RetroPanel class="mb-6" :announce="false" :speaker="pet.status ? name : undefined" :tone="pet.isWilted ? 'ember' : 'plain'">
      <div v-if="pet.loading && !pet.status" class="flex items-center gap-4 text-ink-2">
        <CompanionSprite stage="seed" :health="0" :size="64" :reduced="true" class="opacity-50" />
        <StateBlock state="loading" />
      </div>
      <StateBlock v-else-if="pet.error && !pet.status" state="error" message="Không tải được bạn đồng hành. Thử lại." action="Thử lại" @action="pet.load()" />
      <template v-else-if="pet.status">
        <div class="mb-3 flex flex-wrap items-center justify-center gap-3">
          <CompanionSprite :name="name" :stage="displayStage(pet.status.stage)" :health="displayHealth(pet.status.health_points)" :react="pet.isWilted ? 'down' : reaction" />
          <HpBar :value="displayHealth(pet.status.health_points)" />
        </div>
        <div class="mb-3 flex flex-wrap gap-2">
          <span v-for="chip in chips" :key="chip.tone" :data-growth-chip="chip.tone" class="border-2 border-line-lit bg-ground-2 px-2 py-1 text-sm" :class="[chip.tone === 'growth' ? 'text-growth' : 'text-torch', !reduced ? 'return-chip' : '']">{{ chip.text }}</span>
        </div>
        <RetroButton v-if="pet.isWilted" variant="danger" block @click="navigateTo('/revive')">Hồi sinh {{ name }}</RetroButton>
        <SpeechBox v-else :announce="false" :line="bubble" :name="name" :stage="pet.status.stage" :health="pet.status.health_points" />
      </template>
    </RetroPanel>
    <span data-hub-announcement class="sr-only" aria-live="polite" aria-atomic="true">{{ announcement }}</span>
    <StateBlock v-if="quest.noRoadmap" state="empty" message="Cậu chưa có hành trình." action="Bắt đầu hành trình 28 ngày" @action="navigateTo('/onboarding')" />
    <template v-else>
      <div v-if="quest.loading && !quest.daily" aria-busy="true">
        <StateBlock state="loading" />
        <div data-loading-quests class="mt-4 space-y-3" aria-hidden="true">
          <div v-for="i in 3" :key="i" data-loading-tile class="flex min-h-14 items-center gap-3 border-2 border-line-dim bg-ground-2 px-3">
            <span class="h-8 w-8 shrink-0 bg-line-dim" />
            <span class="h-3 w-2/3 bg-line-dim" />
          </div>
        </div>
      </div>
      <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được nhiệm vụ. Thử lại." action="Thử lại" @action="quest.load()" />
      <template v-else-if="quest.daily">
        <h2 class="font-display text-xl text-ink-1">Phòng hôm nay</h2>
        <DayBar :value-seconds="quest.accumulatedSeconds" :segment-seconds="quest.daily.total_minutes_required * 60 / 3" :met="quest.targetMet" />
        <ul class="mt-4 [container-type:inline-size]" aria-label="Nhiệm vụ hôm nay">
          <QuestNode v-for="(task, i) in quest.sortedTasks" :key="task.id" :class="{ 'return-connector': i === earnedConnector }" :task="task" :index="i" :state="rowState(task.id, task.is_completed)" :connector="i === quest.sortedTasks.length - 1 ? 'none' : (task.is_completed || quest.targetMet ? 'lit' : 'dim')" @enter="enter" />
        </ul>
      </template>
    </template>
    <div v-if="canNavigate" data-next-task class="retro-bottom-bar">
      <RetroButton block @click="next">{{ quest.nextTaskId ? `Vào nhiệm vụ ${nextIndex} →` : 'Xem hành trình' }}</RetroButton>
    </div>
    <RetroToast />
  </main>
</template>

<style scoped>
@media (prefers-reduced-motion: no-preference) {
  .return-chip { animation: earned-rise 150ms steps(1, end) both; }
  .return-connector :deep([data-connector]) {
    transform-origin: top;
    animation: earned-path 300ms steps(4, end) both;
  }
}
@keyframes earned-rise {
  from { transform: translateY(8px); }
  to { transform: translateY(0); }
}
@keyframes earned-path {
  from { clip-path: inset(0 0 100% 0); }
  to { clip-path: inset(0); }
}
</style>
