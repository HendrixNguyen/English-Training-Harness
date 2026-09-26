<script setup lang="ts">
import { computed } from 'vue'
import { useRetroToast } from '~/composables/useRetroToast'
import RetroPanel from './RetroPanel.vue'

/** One-line toast, one at a time, auto-dismissed after 2s (design §5). */
const { queue } = useRetroToast()
const head = computed(() => queue.value[0])
</script>

<template>
  <div
    v-if="head"
    role="status"
    class="retro-rise fixed inset-x-4 mx-auto max-w-md"
    style="bottom: calc(80px + env(safe-area-inset-bottom))"
  >
    <RetroPanel :tone="head.tone ?? 'plain'" band>
      {{ head.line }}
    </RetroPanel>
  </div>
</template>
