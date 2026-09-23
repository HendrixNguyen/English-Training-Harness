<script setup lang="ts">
import { computed } from 'vue'
import type { RoadmapNode } from '~/utils/roadmap'

const props = defineProps<{ node: RoadmapNode }>()

const glyph = computed(() => ({ completed: '⭐', today: '🌱', locked: '🔒' })[props.node.state])
const text = computed(() => ({ completed: 'Đã hoàn thành', today: 'HÔM NAY - Đang học', locked: 'Chưa mở khóa' })[props.node.state])
const pill = computed(() => ({
  completed: 'border-streak/40 bg-streak/10 text-ink dark:text-paper',
  today: 'border-growth bg-growth text-white scale-105',
  locked: 'border-mute/30 text-mute',
})[props.node.state])
</script>

<template>
  <component
    :is="node.state === 'today' ? 'NuxtLink' : 'div'"
    :id="`day-${node.day}`"
    :to="node.state === 'today' ? '/' : undefined"
    class="inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-semibold"
    :class="pill"
    :aria-current="node.state === 'today' ? 'step' : undefined"
  >
    <span aria-hidden="true">{{ glyph }}</span>
    <span>Ngày {{ node.day }}: {{ text }}</span>
  </component>
</template>
