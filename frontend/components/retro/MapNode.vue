<script setup lang="ts">
import { computed } from 'vue'
import { tokens } from '~/tailwind.config'
import { GLYPHS, PALETTE } from '~/utils/pixelArt'
import PixelArt from './PixelArt.vue'

/** World-map tile (design §5). */
const props = withDefaults(defineProps<{
  day: number
  state: 'cleared' | 'today' | 'partial' | 'missed' | 'locked'
  title: string
  expanded?: boolean
}>(), { expanded: undefined })

const emit = defineEmits<{ select: [day: number] }>()

function hexPalette(...chars: string[]): Record<string, string> {
  const out: Record<string, string> = {}
  for (const c of chars) out[c] = (tokens as Record<string, string>)[PALETTE[c]]
  return out
}

const WORD: Record<typeof props.state, string> = {
  cleared: 'đã xong',
  today: 'hôm nay',
  partial: 'một phần',
  missed: 'bỏ lỡ',
  locked: 'khoá',
}

const border = computed(() => ({
  cleared: 'border-torch',
  today: 'border-growth bg-ground-2',
  partial: 'border-line-lit',
  missed: 'border-ember',
  locked: 'border-line-dim text-ink-2',
})[props.state])

const label = computed(() => `Ngày ${props.day}: ${props.title}, ${WORD[props.state]}`)
const disabled = computed(() => props.state === 'locked')

function onClick() {
  if (!disabled.value) emit('select', props.day)
}
</script>

<template>
  <button
    type="button"
    class="relative flex h-10 w-10 items-center justify-center border-2 bg-ground-1"
    :class="border"
    :aria-label="label"
    :aria-current="state === 'today' ? 'step' : undefined"
    :aria-disabled="disabled ? 'true' : undefined"
    :aria-expanded="expanded"
    @click="onClick"
  >
    <span class="font-display text-xl leading-6">{{ day }}</span>

    <span v-if="state === 'cleared'" data-glyph="star" class="absolute -right-1 -top-1">
      <PixelArt :rows="GLYPHS.star" :palette="hexPalette('t', 'T')" :size="12" />
    </span>

    <span v-if="state === 'missed'" data-glyph="ring" class="absolute -right-1 -top-1">
      <PixelArt :rows="GLYPHS.ring" :palette="hexPalette('d')" :size="12" />
    </span>

    <span v-if="state === 'partial'" data-partial-fill class="absolute inset-x-0 bottom-0 h-1/2 bg-growth" aria-hidden="true" />

    <span v-if="state === 'locked'" class="absolute inset-0 bg-ground-2/60" aria-hidden="true" />

    <slot name="sprite" />
  </button>
</template>
