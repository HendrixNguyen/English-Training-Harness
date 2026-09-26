<script setup lang="ts">
import { computed } from 'vue'
import { formatCountdown } from '~/utils/progress'

const props = defineProps<{ remainingSeconds: number }>()
const text = computed(() => formatCountdown(props.remainingSeconds))
</script>

<template>
  <span aria-live="off">
    <span class="font-body text-sm text-ink-1">Thời gian:</span>
    <span class="font-display text-xl tabular-nums" :class="remainingSeconds === 0 ? 'text-ember' : 'text-ink-0'">{{ text }}</span>
  </span>
</template>
