<script setup lang="ts">
import { computed } from 'vue'
import { SEGMENT_COUNT, SEGMENT_SECONDS, minutesOf, percentOf, segmentFills } from '~/utils/progress'

const props = withDefaults(defineProps<{
  valueSeconds: number
  label: string
  segments?: number
  segmentSeconds?: number
  met?: boolean
}>(), { segments: SEGMENT_COUNT, segmentSeconds: SEGMENT_SECONDS, met: false })

const target = computed(() => props.segments * props.segmentSeconds)
const fills = computed(() => segmentFills(props.valueSeconds, props.segmentSeconds, props.segments))
const minutes = computed(() => minutesOf(Math.min(props.valueSeconds, target.value)))
const pct = computed(() => percentOf(props.valueSeconds, target.value))
const totalMinutes = computed(() => minutesOf(target.value))
</script>

<template>
  <div>
    <div class="flex items-baseline justify-between">
      <span class="text-xs font-semibold uppercase tracking-wider text-mute">{{ label }}:</span>
      <span class="font-display text-2xl" :class="met ? 'text-growth' : ''">
        {{ minutes }} / {{ totalMinutes }} phút
      </span>
    </div>
    <div
      class="mt-2 flex gap-1.5"
      role="progressbar"
      :aria-valuenow="Math.min(valueSeconds, target)"
      aria-valuemin="0"
      :aria-valuemax="target"
      :aria-label="label"
    >
      <div v-for="(fill, i) in fills" :key="i" class="h-3 flex-1 overflow-hidden rounded-full bg-mute/20">
        <div
          data-segment
          class="h-full rounded-full bg-growth transition-[width] duration-300 motion-reduce:transition-none"
          :style="{ width: `${Math.round(fill * 100)}%` }"
        />
      </div>
    </div>
    <p class="mt-1 text-right text-xs text-mute">
      <template v-if="met">Mục tiêu hôm nay đã đạt ✓</template>
      <template v-else>{{ pct }}%</template>
    </p>
  </div>
</template>
