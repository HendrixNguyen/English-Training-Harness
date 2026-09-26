<script setup lang="ts">
import { computed, useId } from 'vue'
import { tokens } from '~/tailwind.config'

/**
 * The dialogue box (design §5). `speaker` renders a tab `<h2>` overlapping
 * the top line and turns the panel into an `aria-live="polite"` region;
 * `tone` recolours the 4-px outer ring. `band`/`fog` are one class each —
 * cheap to carry now even though only the revive/roadmap screens (plans
 * 4-6) use them.
 */
const props = withDefaults(defineProps<{
  speaker?: string
  tone?: 'plain' | 'growth' | 'torch' | 'ember'
  band?: boolean
  fog?: boolean
}>(), { speaker: undefined, tone: 'plain', band: false, fog: false })

const headingId = useId()

const outerColor = computed(() => (props.tone === 'plain' ? tokens['line-dim'] : tokens[props.tone]))
const ringStyle = computed(() => ({
  boxShadow: `0 0 0 2px ${tokens['ground-0']}, 0 0 0 4px ${outerColor.value}`,
}))
</script>

<template>
  <section
    class="relative"
    :class="[band ? 'p-2' : 'border-2 border-line-lit bg-ground-1 p-4', speaker ? 'mt-3' : '']"
    :style="ringStyle"
    :aria-labelledby="speaker ? headingId : undefined"
    :aria-live="speaker ? 'polite' : undefined"
  >
    <h2
      v-if="speaker"
      :id="headingId"
      class="absolute -top-3 left-3 bg-ground-1 px-2 font-display text-[22px] leading-6 text-ink-0"
    >
      {{ speaker }}
    </h2>
    <div class="relative flex gap-4">
      <div v-if="$slots.portrait" class="shrink-0">
        <slot name="portrait" />
      </div>
      <div class="min-w-0 flex-1">
        <slot />
      </div>
    </div>
    <div v-if="fog" class="pointer-events-none absolute inset-0 bg-ground-2/60" />
  </section>
</template>
