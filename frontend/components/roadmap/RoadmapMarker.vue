<script setup lang="ts">
import { computed } from 'vue'
import type { NodeState } from '~/utils/roadmap'

const props = defineProps<{ state: NodeState }>()

const glyph = computed(() => ({ today: '🌱', completed: '★', partial: '◐', missed: '○', locked: '🔒' })[props.state])

const ring = computed(() => ({
  today: 'border-growth text-growth',
  completed: 'border-streak/40 bg-streak/10 text-streak',
  partial: 'border-streak/40 text-streak',
  missed: 'border-mute/40 text-mute',
  locked: 'border-mute/30 text-mute',
})[props.state])
</script>

<template>
  <span
    aria-hidden="true"
    :data-state="state"
    class="flex size-6 shrink-0 items-center justify-center rounded-full border text-xs"
    :class="ring"
  >{{ glyph }}</span>
</template>
