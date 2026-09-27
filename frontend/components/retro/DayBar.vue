<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { tokens } from '~/tailwind.config'
import { minutesOf, percentOf, segmentFills } from '~/utils/progress'
import { useReducedMotion } from '~/composables/useReducedMotion'

/**
 * The 30-minute day as `segments` HpBar-style tracks (design §5). Revive
 * passes `segments=1 segmentSeconds=900 cells=40` for its single
 * 15-minute challenge bar.
 */
const props = withDefaults(defineProps<{
  valueSeconds: number
  segments?: number
  segmentSeconds?: number
  met?: boolean
  cells?: number
  reduced?: boolean
  /** folded bug: the group had no accessible name. */
  label?: string
}>(), { segments: 3, segmentSeconds: 600, met: false, cells: 20, reduced: undefined, label: 'Phòng hôm nay' })

const os = useReducedMotion()
const isReduced = computed(() => props.reduced ?? os.value)

const target = computed(() => props.segments * props.segmentSeconds)
const fills = computed(() => segmentFills(props.valueSeconds, props.segmentSeconds, props.segments))
const minutes = computed(() => minutesOf(Math.min(props.valueSeconds, target.value)))
const totalMinutes = computed(() => minutesOf(target.value))
const pct = computed(() => percentOf(props.valueSeconds, target.value))
const segmentPx = computed(() => props.cells * 4)

function litOf(fill: number) {
  return Math.round(fill * props.cells)
}

/** folded bug: same steps()-tracks-the-delta rule as HpBar (design amend),
 * kept per segment index since each segment fills independently. */
const previousLits = ref<number[]>([])
watch(fills, (_newFills, oldFills) => {
  previousLits.value = (oldFills ?? []).map(litOf)
}, { flush: 'sync' })

function segStyle(fill: number, index: number) {
  const lit = litOf(fill)
  const prev = previousLits.value[index] ?? lit
  const steps = Math.max(1, Math.abs(lit - prev))
  return {
    width: `${lit * 4}px`,
    backgroundColor: tokens.growth,
    ...(isReduced.value ? {} : { transition: `width 300ms steps(${steps})` }),
  }
}
</script>

<template>
  <div>
    <div class="flex items-baseline justify-end">
      <span class="font-display text-xl leading-6 text-ink-0">{{ minutes }}/{{ totalMinutes }}</span>
    </div>
    <div
      class="mt-2 flex gap-1"
      role="progressbar"
      :aria-valuenow="Math.min(valueSeconds, target)"
      aria-valuemin="0"
      :aria-valuemax="target"
      :aria-label="label"
    >
      <div
        v-for="(fill, i) in fills"
        :key="i"
        class="h-3 border-2 border-line-dim bg-ground-2"
        :style="{ width: `${segmentPx}px` }"
      >
        <i data-segment class="retro-anim block h-full" :style="segStyle(fill, i)" />
      </div>
    </div>
    <p class="mt-1 text-right text-sm" :class="met ? 'text-growth' : 'text-ink-1'">
      <template v-if="met">Phòng hôm nay đã xong</template>
      <template v-else>{{ pct }}%</template>
    </p>
  </div>
</template>
