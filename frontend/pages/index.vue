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
import { useRetroToast } from '~/composables/useRetroToast'
import { useGrowthMoment } from '~/composables/useGrowthMoment'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'

const quest = useQuestStore()
const pet = usePetStore()
const { start, chips, reaction, pulse, displayHealth, displayStage } = useGrowthMoment()
// Consume before first paint: the store already holds the saved after-values.
start(pet.consumeDelta())
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
const canNavigate = computed(() => !!quest.daily && !quest.loading && !quest.noRoadmap && quest.sortedTasks.length > 0)
function enter(id: string) { void navigateTo(`/learn/${id}`) }
function next() { if (quest.nextTaskId) enter(quest.nextTaskId); else void navigateTo('/roadmap') }
</script>

<template>
  <main class="retro-screen mx-auto max-w-md px-4">
    <AppHeader :streak="pet.status?.current_streak ?? null" :pulse="pulse" />
    <RetroPanel class="mb-6" :speaker="pet.status ? name : undefined" :tone="pet.isWilted ? 'ember' : 'plain'">
      <StateBlock v-if="pet.loading && !pet.status" state="loading" />
      <StateBlock v-else-if="pet.error && !pet.status" state="error" message="Không tải được bạn đồng hành. Thử lại." action="Thử lại" @action="pet.load()" />
      <template v-else-if="pet.status">
        <div class="mb-3 flex flex-wrap items-center justify-center gap-3">
          <CompanionSprite :name="name" :stage="displayStage(pet.status.stage)" :health="displayHealth(pet.status.health_points)" :react="pet.isWilted ? 'down' : reaction" />
          <HpBar :value="displayHealth(pet.status.health_points)" />
        </div>
        <div class="mb-3 flex flex-wrap gap-2" aria-live="polite">
          <span v-for="chip in chips" :key="chip.tone" :data-growth-chip="chip.tone" class="border-2 border-line-lit bg-ground-2 px-2 py-1 text-sm" :class="chip.tone === 'growth' ? 'text-growth' : 'text-torch'">{{ chip.text }}</span>
        </div>
        <RetroButton v-if="pet.isWilted" variant="danger" block @click="navigateTo('/revive')">Hồi sinh {{ name }}</RetroButton>
        <SpeechBox v-else :line="bubble" :name="name" :stage="pet.status.stage" :health="pet.status.health_points" />
      </template>
    </RetroPanel>
    <StateBlock v-if="quest.noRoadmap" state="empty" message="Cậu chưa có hành trình." action="Bắt đầu hành trình 28 ngày" @action="navigateTo('/onboarding')" />
    <template v-else>
      <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
      <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được nhiệm vụ. Thử lại." action="Thử lại" @action="quest.load()" />
      <template v-else-if="quest.daily">
        <h2 class="font-display text-xl text-ink-1">Phòng hôm nay</h2>
        <DayBar :value-seconds="quest.accumulatedSeconds" :segment-seconds="quest.daily.total_minutes_required * 60 / 3" :met="quest.targetMet" />
        <ul class="mt-4" aria-label="Nhiệm vụ hôm nay">
          <QuestNode v-for="(task, i) in quest.sortedTasks" :key="task.id" :task="task" :index="i" :state="rowState(task.id, task.is_completed)" :connector="i === quest.sortedTasks.length - 1 ? 'none' : (task.is_completed || quest.targetMet ? 'lit' : 'dim')" @enter="enter" />
        </ul>
      </template>
    </template>
    <div v-if="canNavigate" data-next-task class="retro-bottom-bar">
      <RetroButton block @click="next">{{ quest.nextTaskId ? `Vào nhiệm vụ ${nextIndex} →` : 'Xem hành trình' }}</RetroButton>
    </div>
    <RetroToast />
  </main>
</template>
