<script setup lang="ts">
import PixelArt from '~/components/retro/PixelArt.vue'
import { GLYPHS } from '~/utils/pixelArt'
import { tokens } from '~/tailwind.config'
import type { Gloss } from '~/utils/content'
defineProps<{ gloss: Gloss, id: string }>()
defineEmits<{ close: [] }>()
</script>

<template>
  <div :id="id" class="mt-3 rounded-none border-2 border-line-lit bg-ground-2 p-3 font-body text-ink-0" role="region" aria-live="polite" aria-label="Giải nghĩa">
    <div class="flex items-start justify-between gap-2">
      <p class="text-[17px] font-bold">{{ gloss.term }}</p>
      <button type="button" class="flex h-11 w-11 shrink-0 items-center justify-center rounded-sm border-2 border-line-lit bg-ground-1 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch" aria-label="Đóng giải nghĩa" @click="$emit('close')">
        <PixelArt :rows="GLYPHS.cross" :size="16" :palette="{ e: tokens['ink-0'] }" />
      </button>
    </div>
    <p class="text-[17px]">{{ gloss.meaning_vi }}</p>
    <p class="text-sm text-ink-1">{{ gloss.definition }}</p>
  </div>
</template>
