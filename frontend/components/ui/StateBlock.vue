<script setup lang="ts">
import { tokens } from '~/tailwind.config'
import RetroButton from '~/components/retro/RetroButton.vue'
import PixelArt from '~/components/retro/PixelArt.vue'
import { GLYPHS, PALETTE } from '~/utils/pixelArt'

/** Every async region's three states (design §5, kept: props/role="status"
 * unchanged — `revivePage`/`onboardingPage` click `[role="status"] button`). */
defineProps<{
  state: 'loading' | 'empty' | 'error'
  message?: string
  action?: string
}>()
defineEmits<{ action: [] }>()

function hexPalette(...chars: string[]): Record<string, string> {
  const out: Record<string, string> = {}
  for (const c of chars) out[c] = (tokens as Record<string, string>)[PALETTE[c]]
  return out
}
</script>

<template>
  <div v-if="state === 'loading'" class="flex items-center gap-2" aria-busy="true" aria-label="Đang tải">
    <span
      v-for="i in 3"
      :key="i"
      class="retro-dots h-2 w-2 bg-ground-2"
      :style="{ animationDelay: `${(i - 1) * 150}ms` }"
    />
  </div>
  <div v-else class="flex flex-col items-start gap-3" role="status">
    <p class="flex items-center gap-2 font-body text-[17px] text-ink-0">
      <PixelArt v-if="state === 'error'" :rows="GLYPHS.cross" :palette="hexPalette('e')" :size="16" />
      {{ message }}
    </p>
    <RetroButton v-if="action" :variant="state === 'error' ? 'secondary' : 'primary'" @click="$emit('action')">
      {{ action }}
    </RetroButton>
  </div>
</template>
