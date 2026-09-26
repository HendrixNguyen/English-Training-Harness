<script setup lang="ts">
import { computed } from 'vue'
import RoadmapMarker from '~/components/roadmap/RoadmapMarker.vue'
import AppCard from '~/components/ui/AppCard.vue'
import { formatDayDate, statusText, TASK_LABELS, type RoadmapNode } from '~/utils/roadmap'

const props = defineProps<{ node: RoadmapNode, expanded: boolean }>()
defineEmits<{ 'update:expanded': [boolean] }>()

const titleClass = computed(() => (props.node.state === 'missed' || props.node.state === 'locked' ? 'text-mute' : ''))
const statusClass = computed(() => {
  if (props.node.state === 'today') return 'text-growth'
  if (props.node.state === 'completed' || props.node.state === 'partial') return 'text-streak'
  return 'text-mute'
})
</script>

<template>
  <li :id="`day-${node.day}`" class="flex flex-col gap-2">
    <button
      type="button"
      class="flex min-h-14 w-full items-center gap-3 rounded-btn text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-growth focus-visible:ring-offset-2"
      :aria-expanded="expanded"
      :aria-current="node.state === 'today' ? 'step' : undefined"
      @click="$emit('update:expanded', !expanded)"
    >
      <RoadmapMarker :state="node.state" />
      <span class="min-w-0 flex-1">
        <span class="block text-[13px] leading-[18px] tabular-nums text-mute">Ngày {{ node.day }} · {{ formatDayDate(node.date) }}</span>
        <span class="block text-base font-semibold" :class="titleClass">{{ node.title }}</span>
        <span class="block text-[13px] leading-[18px]" :class="statusClass">{{ statusText(node) }}</span>
      </span>
      <span v-if="node.state === 'today'" class="shrink-0 rounded-full bg-growth px-2 text-xs text-white">HÔM NAY</span>
    </button>
    <AppCard v-show="expanded" class="ml-9">
      <ul :aria-label="`Nhiệm vụ ngày ${node.day}`" class="space-y-2">
        <li v-for="task in node.tasks" :key="task.task_type" class="flex items-baseline justify-between gap-2 text-[13px]">
          <span>{{ TASK_LABELS[task.task_type] ?? task.task_type }} · {{ task.title }}</span>
          <span class="shrink-0 tabular-nums text-mute">{{ task.duration_minutes }} phút</span>
        </li>
      </ul>
      <NuxtLink
        v-if="node.state === 'today'"
        to="/"
        class="mt-3 inline-flex h-12 w-full items-center justify-center gap-2 rounded-btn bg-growth px-5 font-semibold text-white hover:bg-growth/90"
      >
        Học ngay →
      </NuxtLink>
    </AppCard>
  </li>
</template>
