<script setup lang="ts">
import { computed } from 'vue'
import type { QuestTask } from '~/stores/quest'

const props = defineProps<{ task: QuestTask, index: number, state: 'done' | 'next' | 'locked' | 'open' }>()

const glyph = computed(() => ({ done: '[✓]', next: '[▶]', locked: '[ ]', open: '[ ]' })[props.state])
const action = computed(() => ({ done: 'Xong', next: 'Học', locked: 'Khóa', open: 'Học' })[props.state])
</script>

<template>
  <li class="flex items-center gap-3 py-2">
    <span class="w-7 font-mono text-sm" :class="state === 'done' ? 'text-growth' : 'text-mute'" aria-hidden="true">{{ glyph }}</span>
    <span class="flex-1" :class="{ 'text-mute': state === 'locked' }">
      {{ index }}. {{ task.title }}
      <span class="text-sm text-mute">({{ task.duration_minutes }}m)</span>
    </span>
    <NuxtLink v-if="state === 'next' || state === 'open'" :to="`/learn/${task.id}`" class="inline-flex h-9 items-center rounded-btn bg-growth px-4 text-sm font-semibold text-white" :aria-label="`Học: ${task.title}`">
      {{ action }}
    </NuxtLink>
    <span v-else class="inline-flex h-9 items-center px-4 text-sm font-semibold" :class="state === 'done' ? 'text-growth' : 'text-mute'">
      {{ action }}
    </span>
  </li>
</template>
