<script setup lang="ts">
import { computed } from 'vue'
import { tokens } from '~/tailwind.config'
import { GLYPHS, PALETTE } from '~/utils/pixelArt'
import PixelArt from './PixelArt.vue'

/** 32-px pixel emblem (design §5). `earned` draws every colour as
 * authored; an unearned badge dims every non-outline char to `line-dim`
 * and hides the `k` outline so only the muted silhouette shows. */
const props = withDefaults(defineProps<{
  kind: 'streak' | 'shield' | 'star'
  count?: number
  earned?: boolean
}>(), { count: undefined, earned: true })

const KIND_GLYPH = { streak: 'flame', shield: 'shield', star: 'star' } as const
const KIND_LABEL = { streak: 'chuỗi', shield: 'khiên', star: 'sao' } as const

const rows = computed(() => GLYPHS[KIND_GLYPH[props.kind]])

const palette = computed(() => {
  const out: Record<string, string> = {}
  for (const [char, roleName] of Object.entries(PALETTE)) {
    if (props.earned) {
      out[char] = (tokens as Record<string, string>)[roleName]
    } else {
      out[char] = char === 'k' ? 'transparent' : tokens['line-dim']
    }
  }
  return out
})

const label = computed(() => (props.kind === 'streak' ? `${KIND_LABEL.streak} ${props.count ?? 0} ngày` : KIND_LABEL[props.kind]))
</script>

<template>
  <span class="inline-flex items-center gap-1" role="img" :aria-label="label">
    <PixelArt :rows="rows" :palette="palette" :size="32" />
    <span v-if="count !== undefined" class="font-display text-xl leading-6" :class="earned ? 'text-torch' : 'text-line-dim'">x{{ count }}</span>
  </span>
</template>
