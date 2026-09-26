<script setup lang="ts">
import { computed } from 'vue'
import { healthTone, normalizeStage } from '~/utils/plant'

const props = withDefaults(defineProps<{ stage: string, health: number, size?: number, grow?: boolean }>(), { size: 160, grow: false })

const norm = computed(() => normalizeStage(props.stage))
const toneClass = computed(() => {
  if (!norm.value.known) return 'text-mute'
  if (norm.value.stage === 'wilted') return 'text-alert'
  return { growth: 'text-growth', streak: 'text-streak', alert: 'text-alert' }[healthTone(props.health)]
})
const label = computed(() => `Cây đang ở giai đoạn ${norm.value.stage}, máu ${Math.max(0, props.health)}%`)
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 160 160"
    role="img"
    :aria-label="label"
    :class="toneClass"
    class="mx-auto block"
  >
    <!-- pot and soil: identical in every stage so the plant reads as one plant growing -->
    <path d="M52 112h56l-8 32H60z" class="fill-ink dark:fill-paper/80" />
    <ellipse cx="80" cy="112" rx="30" ry="5" class="fill-ink/70 dark:fill-paper/50" />

    <g data-plant :class="{ 'plant-grow': grow }">
      <Transition name="stage" mode="out-in" type="transition">
        <g v-if="norm.stage === 'seed'" key="seed" data-stage="seed">
          <path d="M62 112c0-8 8-12 18-12s18 4 18 12z" class="fill-ink/50" />
          <circle cx="80" cy="106" r="3" class="fill-ink" />
        </g>

        <g v-else-if="norm.stage === 'sprout'" key="sprout" data-stage="sprout" class="plant-sway">
          <path d="M80 110V78" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
          <path d="M80 86c-18 0-26-12-26-22 16 0 26 8 26 22z" fill="currentColor" />
          <path d="M80 80c18 0 26-12 26-22-16 0-26 8-26 22z" fill="currentColor" />
        </g>

        <g v-else-if="norm.stage === 'sapling'" key="sapling" data-stage="sapling" class="plant-sway">
          <path d="M80 110V50" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
          <path d="M80 96c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
          <path d="M80 84c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
          <path d="M80 72c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
          <path d="M80 60c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
          <path d="M80 50c-8-6-10-14-8-20 8 4 10 12 8 20z" fill="currentColor" />
        </g>

        <g v-else-if="norm.stage === 'flowering'" key="flowering" data-stage="flowering" class="plant-sway">
          <path d="M80 110V44" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
          <path d="M80 96c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
          <path d="M80 84c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
          <path d="M80 72c-16 0-22-10-22-18 14 0 22 7 22 18z" fill="currentColor" />
          <path d="M80 60c16 0 22-10 22-18-14 0-22 7-22 18z" fill="currentColor" />
          <circle cx="58" cy="56" r="5" class="fill-streak" />
          <circle cx="102" cy="44" r="5" class="fill-streak" />
          <circle cx="80" cy="40" r="6" class="fill-streak" />
        </g>

        <g v-else-if="norm.stage === 'fruitful'" key="fruitful" data-stage="fruitful" class="plant-sway">
          <path d="M80 110V40" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" />
          <path d="M80 98c-20 0-28-12-28-22 18 0 28 9 28 22z" fill="currentColor" />
          <path d="M80 86c20 0 28-12 28-22-18 0-28 9-28 22z" fill="currentColor" />
          <path d="M80 72c-20 0-28-12-28-22 18 0 28 9 28 22z" fill="currentColor" />
          <path d="M80 60c20 0 28-12 28-22-18 0-28 9-28 22z" fill="currentColor" />
          <circle cx="56" cy="60" r="5" class="fill-streak" />
          <circle cx="104" cy="48" r="5" class="fill-streak" />
          <circle cx="66" cy="84" r="7" class="fill-streak" />
          <circle cx="96" cy="70" r="7" class="fill-streak" />
        </g>

        <g v-else key="wilted" data-stage="wilted" class="plant-droop origin-[80px_110px]">
          <path d="M80 110c0-24 6-40 14-52" stroke="currentColor" stroke-width="5" stroke-linecap="round" fill="none" opacity="0.7" />
          <path d="M84 94c-14 4-22-4-24-12 12-2 20 4 24 12z" fill="currentColor" opacity="0.6" />
          <path d="M90 76c14 2 20-6 20-14-12 0-18 6-20 14z" fill="currentColor" opacity="0.6" />
          <path d="M94 62c-6 8-14 10-20 8 4-8 12-12 20-8z" fill="currentColor" opacity="0.5" />
          <path d="M46 116c8-6 16-6 22-2-6 4-14 6-22 2z" fill="currentColor" opacity="0.5" />
        </g>
      </Transition>
    </g>
  </svg>
</template>

<style scoped>
@media (prefers-reduced-motion: no-preference) {
  .plant-sway {
    transform-origin: 80px 110px;
    animation: sway 4s ease-in-out infinite;
  }
  .plant-droop {
    animation: droop 600ms ease-out forwards;
  }
  .stage-enter-active,
  .stage-leave-active {
    transition: opacity 300ms ease;
  }
  .stage-enter-from,
  .stage-leave-to {
    opacity: 0;
  }
  .plant-grow {
    transform-origin: 80px 110px;
    animation: grow 800ms cubic-bezier(.2, .8, .2, 1) 1;
  }
}
@media (prefers-reduced-motion: reduce) {
  .plant-droop {
    transform: rotate(12deg);
  }
}
@keyframes sway {
  0%, 100% { transform: rotate(-2deg); }
  50% { transform: rotate(2deg); }
}
@keyframes droop {
  from { transform: rotate(0deg); }
  to { transform: rotate(12deg); }
}
@keyframes grow {
  0% { transform: scale(1); }
  45% { transform: scale(1.06); }
  100% { transform: scale(1); }
}
</style>
