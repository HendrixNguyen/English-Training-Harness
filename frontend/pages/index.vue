<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { speechLine } from '~/utils/plant'

const quest = useQuestStore()
const pet = usePetStore()

onMounted(() => {
  void Promise.all([pet.load(), quest.load()])
})

const bubble = computed(() => pet.status
  ? speechLine({ stage: pet.status.stage, health: pet.status.health_points, targetMet: quest.targetMet, name: pet.status.plant_name })
  : '')

function rowState(taskId: string, completed: boolean): 'done' | 'next' | 'locked' | 'open' {
  if (completed) return 'done'
  if (quest.targetMet) return 'open' // extra study is allowed once the day is met
  return taskId === quest.nextTaskId ? 'next' : 'locked'
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader :streak="pet.status?.current_streak ?? null" />

    <NuxtLink
      v-if="pet.isWilted"
      to="/revive"
      class="mb-4 flex items-center justify-between rounded-card bg-alert px-4 py-3 font-semibold text-white"
    >
      <span>⚠️ {{ pet.status?.plant_name || 'Cây xanh' }} đang bị héo rũ!</span>
      <span class="text-sm underline">Cứu cây ngay</span>
    </NuxtLink>

    <AppCard class="mb-4">
      <StateBlock v-if="pet.loading && !pet.status" state="loading" />
      <StateBlock v-else-if="pet.error && !pet.status" state="error" message="Không tải được cây của bạn." action="Thử lại" @action="pet.load()" />
      <template v-else-if="pet.status">
        <p v-if="pet.status.plant_name" class="text-center font-display text-lg">
          {{ pet.status.plant_name }}
        </p>
        <PlantSvg :stage="pet.status.stage" :health="pet.status.health_points" />
        <HealthBar class="mt-3" :health="pet.status.health_points" />
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
