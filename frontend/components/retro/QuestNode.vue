<script setup lang="ts">
import { computed } from 'vue'
import { tokens } from '~/tailwind.config'
import { GLYPHS, PALETTE } from '~/utils/pixelArt'
import type { QuestTask } from '~/stores/quest'
import PixelArt from './PixelArt.vue'

/** The hub path tile (design §5). Icon by `task.task_type`: book =
 * vocabulary, scroll = reading, sword = practice. */
const props = withDefaults(defineProps<{
  task: QuestTask
  index: number
  state: 'done' | 'current' | 'open' | 'locked'
  connector?: 'lit' | 'dim' | 'none'
  reduced?: boolean
}>(), { connector: 'none', reduced: false })

const emit = defineEmits<{ enter: [id: string] }>()

const TASK_GLYPH: Record<string, keyof typeof GLYPHS> = { vocabulary: 'book', reading: 'scroll', practice: 'sword' }
const taskGlyph = computed(() => TASK_GLYPH[props.task.task_type] ?? 'book')

function hexPalette(...chars: string[]): Record<string, string> {
  const out: Record<string, string> = {}
  for (const c of chars) out[c] = (tokens as Record<string, string>)[PALETTE[c]]
  return out
}

const border = computed(() => ({
  done: 'border-line-lit',
  current: 'border-growth',
  open: 'border-line-lit',
  locked: 'border-line-dim text-ink-2',
})[props.state])

const actionWord = computed(() => ({ done: 'Đã xong', current: 'Vào', open: 'Vào', locked: 'Khoá' })[props.state])
const disabled = computed(() => props.state === 'locked' || props.state === 'done')

function onClick() {
  if (props.state === 'open' || props.state === 'current') emit('enter', props.task.id)
}
</script>

<template>
  <li class="flex flex-col items-center">
    <div class="flex w-full items-center gap-3">
      <span class="w-2 shrink-0" aria-hidden="true">
        <PixelArt v-if="state === 'current' && !reduced" :rows="GLYPHS.cursor" :palette="hexPalette('l')" :size="16" class="retro-blink" />
      </span>
      <button
        type="button"
        class="relative flex h-14 w-14 shrink-0 items-center justify-center border-2 bg-ground-1"
        :class="border"
        :aria-current="state === 'current' ? 'step' : undefined"
        :aria-disabled="disabled ? 'true' : undefined"
        @click="onClick"
      >
        <PixelArt :rows="GLYPHS[taskGlyph]" :palette="hexPalette('k', 'l')" :size="32" />
        <span v-if="state === 'done'" data-glyph="star" class="absolute -right-1 -top-1">
          <PixelArt :rows="GLYPHS.star" :palette="hexPalette('t', 'T')" :size="16" />
        </span>
        <span v-if="state === 'locked'" data-glyph="padlock" class="absolute -right-1 -top-1">
          <PixelArt :rows="GLYPHS.padlock" :palette="hexPalette('k', 'd')" :size="16" />
        </span>
      </button>
      <div class="min-w-0 flex-1">
        <p class="truncate font-body text-[17px]" :class="state === 'locked' ? 'text-ink-2' : 'text-ink-0'">
          {{ index + 1 }}. {{ task.title }}
          <span class="text-sm text-ink-1">({{ task.duration_minutes }} phút)</span>
        </p>
      </div>
      <span class="shrink-0 font-display text-xl leading-6" :class="state === 'locked' ? 'text-ink-2' : 'text-ink-0'">{{ actionWord }}</span>
    </div>
    <div class="h-4 w-1">
      <span
        v-if="connector !== 'none'"
        data-connector
        class="block h-full w-full"
        :class="connector === 'lit' ? 'bg-growth' : 'bg-line-dim'"
      />
    </div>
  </li>
</template>
