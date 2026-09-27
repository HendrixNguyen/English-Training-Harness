<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { tokens } from '~/tailwind.config'
import { healthTone } from '~/utils/plant'
import { useReducedMotion } from '~/composables/useReducedMotion'

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
}>(), { max: 100, label: 'HP', cells: 25, reduced: undefined })

const os = useReducedMotion()
/** design amend A4: an explicit prop overrides the OS setting. */
const isReduced = computed(() => props.reduced ?? os.value)

const TONE_HEX: Record<'growth' | 'streak' | 'alert', string> = {
  growth: tokens.growth,
  streak: tokens.torch,
  alert: tokens.ember,
}

const ratio = computed(() => Math.min(1, Math.max(0, props.value / props.max)))
const lit = computed(() => Math.round(ratio.value * props.cells))
const trackPx = computed(() => props.cells * 4 + 4)
const tone = computed(() => TONE_HEX[healthTone(ratio.value * 100)])
/** folded bug: the label and `aria-valuenow` must never print a value
 * outside 0..max, even though the fill itself was already clamped. */
const shown = computed(() => Math.min(props.max, Math.max(0, Math.round(props.value))))

/** folded bug: the fill's `steps()` count must track the *change* in lit
 * cells between renders, not the raw cell count — otherwise a 4-cell hop
 * animates over the same number of steps as filling the whole bar. */
const previousLit = ref<number | null>(null)
watch(lit, (_newLit, oldLit) => { previousLit.value = oldLit }, { flush: 'sync' })
const litSteps = computed(() => Math.max(1, Math.abs(lit.value - (previousLit.value ?? lit.value))))

const fillStyle = computed(() => ({
  width: `${lit.value * 4}px`,
  backgroundColor: tone.value,
  ...(isReduced.value ? {} : { transition: `width 300ms steps(${litSteps.value})` }),
}))
</script>

<template>
  <div class="flex items-center gap-2">
    <span class="font-display text-xl leading-6 text-ink-0">{{ label }} {{ shown }}/{{ max }}</span>
    <div
      class="h-3 border-2 border-line-dim bg-ground-2"
      :style="{ width: `${trackPx}px` }"
      role="meter"
      :aria-valuenow="shown"
      aria-valuemin="0"
      :aria-valuemax="max"
      :aria-label="label"
    >
      <i data-fill class="retro-anim block h-full" :style="fillStyle" />
    </div>
  </div>
</template>
