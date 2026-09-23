<script setup lang="ts">
import { computed } from 'vue'
import { formatCountdown } from '~/utils/progress'

const props = defineProps<{ remainingSeconds: number }>()
const text = computed(() => formatCountdown(props.remainingSeconds))
</script>

<template>
  <span class="tabular-nums" :class="remainingSeconds === 0 ? 'text-alert' : ''" aria-live="off">
    ⏱️ Thời gian: <span class="font-semibold">{{ text }}</span>
  </span>
</template>
