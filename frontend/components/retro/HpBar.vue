<script setup lang="ts">
import { computed } from 'vue'
import { tokens } from '~/tailwind.config'
import { healthTone } from '~/utils/plant'

/**
 * HP-style bar (design §5). Fill snaps to 4-px cells so the width is
 * always deterministic and testable; tone follows `healthTone()` (growth
 * ≥60, torch 30-59, ember <30).
 */
const props = withDefaults(defineProps<{
  value: number
  max?: number
  label?: string
  cells?: number
  reduced?: boolean
}>(), { max: 100, label: 'HP', cells: 25, reduced: false })

const TONE_HEX: Record<'growth' | 'streak' | 'alert', string> = {
  growth: tokens.growth,
  streak: tokens.torch,
  alert: tokens.ember,
}

const ratio = computed(() => Math.min(1, Math.max(0, props.value / props.max)))
const lit = computed(() => Math.round(ratio.value * props.cells))
const trackPx = computed(() => props.cells * 4 + 4)
const tone = computed(() => TONE_HEX[healthTone(ratio.value * 100)])

const fillStyle = computed(() => ({
  width: `${lit.value * 4}px`,
  backgroundColor: tone.value,
  ...(props.reduced ? {} : { transition: `width 300ms steps(${Math.max(1, Math.abs(lit.value))})` }),
}))
</script>

<template>
  <div class="flex items-center gap-2">
    <span class="font-display text-xl leading-6 text-ink-0">{{ label }} {{ value }}/{{ max }}</span>
    <div
      class="h-3 border-2 border-line-dim bg-ground-2"
      :style="{ width: `${trackPx}px` }"
      role="meter"
      :aria-valuenow="value"
      aria-valuemin="0"
      :aria-valuemax="max"
      :aria-label="label"
    >
      <i data-fill class="block h-full" :style="fillStyle" />
    </div>
  </div>
</template>
