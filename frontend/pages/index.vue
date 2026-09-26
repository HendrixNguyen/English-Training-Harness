<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useGrowthMoment } from '~/composables/useGrowthMoment'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { speechLine } from '~/utils/plant'

const quest = useQuestStore()
const pet = usePetStore()
const { start, chips, grow, pulse, displayHealth, displayStage } = useGrowthMoment()

// Pinia state survives the SPA navigation from /learn/:id, so pet.status
// already holds the *after* values by this component's very first render —
// consumeDelta() must run before that render (not in onMounted, which fires
// after it) or the first paint shows the after-value, not the before-value
// the growth moment is built on.
start(pet.consumeDelta())

onMounted(() => {
  void Promise.all([pet.load(), quest.load()])
})

const bubble = computed(() => pet.status
  ? speechLine({
      stage: pet.status.stage,
      health: pet.status.health_points,
      targetMet: quest.targetMet,
      accumulatedSeconds: quest.accumulatedSeconds,
      streak: pet.status.current_streak,
      lastPracticedAt: pet.status.last_practiced_at,
    })
  : '')

function rowState(taskId: string, completed: boolean): 'done' | 'next' | 'locked' | 'open' {
  if (completed) return 'done'
  if (quest.targetMet) return 'open' // extra study is allowed once the day is met
  return taskId === quest.nextTaskId ? 'next' : 'locked'
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader :streak="pet.status?.current_streak ?? null" :pulse="pulse" />

    <NuxtLink
      v-if="pet.isWilted"
      to="/revive"
      class="mb-4 flex items-center justify-between rounded-card bg-alert px-4 py-3 font-semibold text-white"
    >
      <span>⚠️ Cây xanh đang bị héo rũ!</span>
      <span class="text-sm underline">Cứu cây ngay</span>
    </NuxtLink>

    <AppCard class="relative mb-4">
      <StateBlock v-if="pet.loading && !pet.status" state="loading" />
      <StateBlock v-else-if="pet.error && !pet.status" state="error" message="Không tải được cây của bạn." action="Thử lại" @action="pet.load()" />
      <template v-else-if="pet.status">
        <TransitionGroup name="chip" tag="div" class="absolute right-4 top-4 flex flex-col items-end gap-1.5" aria-live="polite">
          <GrowthChip v-for="c in chips" :key="c.tone" :text="c.text" :tone="c.tone" />
        </TransitionGroup>
        <PlantSvg :stage="displayStage(pet.status.stage)" :health="displayHealth(pet.status.health_points)" :grow="grow" />
        <HealthBar class="mt-3" :health="displayHealth(pet.status.health_points)" />
        <SpeechBubble v-if="bubble !== '…'" :line="bubble" />
      </template>
    </AppCard>

    <template v-if="quest.noRoadmap">
      <AppCard>
        <StateBlock state="empty" message="Bạn chưa có lộ trình học." action="Tạo lộ trình 28 ngày" @action="navigateTo('/onboarding')" />
      </AppCard>
    </template>
    <template v-else>
      <AppCard class="mb-4">
        <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
        <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được tiến độ." action="Thử lại" @action="quest.load()" />
        <SegmentedProgress v-else :value-seconds="quest.accumulatedSeconds" label="Tiến độ hôm nay" :met="quest.targetMet" />
      </AppCard>

      <AppCard title="Nhiệm vụ hôm nay (Quests)">
        <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
        <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được nhiệm vụ." action="Thử lại" @action="quest.load()" />
        <ul v-else class="divide-y divide-ink/10 dark:divide-paper/10">
          <QuestRow
            v-for="(task, i) in quest.sortedTasks"
            :key="task.id"
            :task="task"
            :index="i + 1"
            :state="rowState(task.id, task.is_completed)"
          />
        </ul>
      </AppCard>
    </template>
  </main>
</template>

<style scoped>
@media (prefers-reduced-motion: no-preference) {
  .chip-enter-active {
    transition: opacity 200ms ease-out, transform 200ms ease-out;
  }
  .chip-leave-active {
    transition: opacity 200ms ease-out;
  }
  .chip-enter-from {
    opacity: 0;
    transform: translateY(8px);
  }
  .chip-leave-to {
    opacity: 0;
  }
}
</style>
