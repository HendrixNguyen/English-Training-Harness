<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import GlossTerm from '~/components/learn/GlossTerm.vue'
import GlossCard from '~/components/learn/GlossCard.vue'
import { segmentPassage } from '~/utils/gloss'
import type { Gloss } from '~/utils/content'
const props = withDefaults(defineProps<{ passage: string, glossary: Gloss[], mode?: 'item' | 'sheet' }>(), { mode: 'item' })
const cardId = useId()
const openIndex = ref<number | null>(null)
const segments = computed(() => segmentPassage(props.passage, props.glossary))
const hasMarks = computed(() => segments.value.some(s => s.gloss !== undefined))
const openGloss = computed(() => openIndex.value === null ? null : props.glossary[openIndex.value])
let opener: HTMLButtonElement | null = null
function toggle(index: number, event: MouseEvent) {
  opener = event.currentTarget as HTMLButtonElement
  openIndex.value = openIndex.value === index ? null : index
}
function closeGloss(returnFocus = false) {
  openIndex.value = null
  if (returnFocus) opener?.focus()
}
function escape(event: KeyboardEvent) {
  if (event.key === 'Escape' && openIndex.value !== null) {
    event.preventDefault()
    closeGloss(true)
  }
}
defineExpose({ closeGloss })
</script>

<template>
  <div class="space-y-3 bg-ground-1 p-2" @keydown="escape">
    <div class="flex h-11 items-center justify-between">
      <p class="font-display text-base uppercase leading-5 tracking-[0.05em] text-ink-1">Đoạn văn</p>
      <slot name="eyebrow-action" />
    </div>
    <p v-if="mode === 'item' && hasMarks" class="font-body text-sm text-ink-1">Chạm vào từ có gạch chấm để xem nghĩa.</p>
    <RetroPanel>
      <div tabindex="0" role="region" aria-label="Đoạn văn" class="overflow-y-auto overscroll-contain border-y-2 border-line-dim py-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch" :class="mode === 'item' ? 'max-h-[50vh]' : 'max-h-[40vh]'">
        <p class="whitespace-pre-line font-body text-lg leading-[30px] text-ink-0"><template v-for="(segment, i) in segments" :key="i"><GlossTerm v-if="segment.gloss !== undefined" :text="segment.text" :active="openIndex === segment.gloss" :controls="cardId" @toggle="toggle(segment.gloss, $event)" /><template v-else>{{ segment.text }}</template></template></p>
      </div>
      <GlossCard v-if="openGloss" :id="cardId" :gloss="openGloss" @close="closeGloss(true)" />
    </RetroPanel>
  </div>
</template>
