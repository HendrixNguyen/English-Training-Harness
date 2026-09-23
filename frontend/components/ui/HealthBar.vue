<script setup lang="ts">
import { computed } from 'vue'
import { healthTone } from '~/utils/plant'

const props = defineProps<{ health: number }>()
const clamped = computed(() => Math.min(100, Math.max(0, props.health)))
const barClass = computed(() => ({ growth: 'bg-growth', streak: 'bg-streak', alert: 'bg-alert' })[healthTone(clamped.value)])
</script>

<template>
  <div class="flex items-center gap-3">
    <span class="text-sm text-mute">Máu cây:</span>
    <div class="h-2 flex-1 overflow-hidden rounded-full bg-mute/20" role="meter" :aria-valuenow="clamped" aria-valuemin="0" aria-valuemax="100" aria-label="Máu cây">
      <div class="h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none" :class="barClass" :style="{ width: `${clamped}%` }" />
    </div>
    <span class="w-10 text-right text-sm font-semibold tabular-nums">{{ clamped }}%</span>
  </div>
</template>
