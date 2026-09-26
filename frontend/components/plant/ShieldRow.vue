<script setup lang="ts">
import { computed } from 'vue'
import { MAX_SHIELDS, localDateYmd, shieldSpentLine } from '~/utils/plant'

const props = defineProps<{ shields: number, lastUsedOn: string | null, today?: string }>()

const held = computed(() => Math.min(MAX_SHIELDS, Math.max(0, Math.floor(props.shields))))
const caption = computed(() => shieldSpentLine(props.lastUsedOn, props.today ?? localDateYmd()))
/** design §4.1: held slots first; the first empty slot is "spent" while the caption shows. */
const slots = computed<Array<'held' | 'spent' | 'empty'>>(() =>
  Array.from({ length: MAX_SHIELDS }, (_, i) => i < held.value ? 'held' : i === held.value && caption.value ? 'spent' : 'empty'))
const label = computed(() => `Khiên: ${held.value} trên ${MAX_SHIELDS}` + (caption.value ? `, một chiếc vừa đỡ cho ngày ${caption.value.slice(-6, -1)}` : ''))
</script>

<template>
  <div>
    <div class="flex items-center gap-3" role="img" :aria-label="label">
      <span class="text-sm text-mute" aria-hidden="true">Khiên:</span>
      <span class="flex items-center gap-2" aria-hidden="true">
        <svg
          v-for="(state, i) in slots"
          :key="i"
          :data-shield="state"
          viewBox="0 0 20 20"
          class="size-5 transition-colors duration-200 motion-reduce:transition-none"
          :class="{ 'text-streak': state === 'held', 'text-mute': state !== 'held' }"
        >
          <path
            d="M10 2 L17 5 V10 C17 14.5 13.5 17.5 10 18.5 C6.5 17.5 3 14.5 3 10 V5 Z"
            :fill="state === 'held' ? 'currentColor' : state === 'spent' ? 'currentColor' : 'none'"
            :fill-opacity="state === 'spent' ? 0.25 : 1"
            stroke="currentColor"
            :stroke-opacity="state === 'empty' ? 0.3 : 1"
            stroke-width="1.5"
            stroke-linejoin="round"
          />
          <path v-if="state === 'spent'" d="M6.5 10.5 L9 13 L13.5 7.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </span>
    </div>
    <p v-if="caption" class="mt-1 text-sm text-mute">{{ caption }}</p>
  </div>
</template>
