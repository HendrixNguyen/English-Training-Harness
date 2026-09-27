<script setup lang="ts">
import RetroButton from '~/components/retro/RetroButton.vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import PixelArt from '~/components/retro/PixelArt.vue'
import { GLYPHS } from '~/utils/pixelArt'

/** Every async region's three states (design §5, kept: props/role="status"
 * unchanged — `revivePage`/`onboardingPage` click `[role="status"] button`). */
defineProps<{
  state: 'loading' | 'empty' | 'error'
  message?: string
  action?: string
}>()
defineEmits<{ action: [] }>()
</script>

<template>
  <div v-if="state === 'loading'" class="inline-flex items-center gap-2 bg-ground-1 p-2" aria-busy="true" aria-label="Đang tải">
    <span
      v-for="i in 3"
      :key="i"
      class="retro-dots h-2 w-2 bg-line-lit"
      :style="{ animationDelay: `${(i - 1) * 150}ms` }"
    />
  </div>
  <div v-else role="status">
    <RetroPanel :tone="state === 'error' ? 'ember' : 'plain'">
      <p class="flex items-center gap-2 font-body text-[17px] text-ink-0">
        <PixelArt v-if="state === 'error'" :rows="GLYPHS.cross" :size="16" />
        {{ message }}
      </p>
      <RetroButton v-if="action" :variant="state === 'error' ? 'secondary' : 'primary'" @click="$emit('action')">
        {{ action }}
      </RetroButton>
    </RetroPanel>
  </div>
</template>
