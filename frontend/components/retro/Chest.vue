<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { GLYPHS } from '~/utils/pixelArt'
import PixelArt from './PixelArt.vue'
import { useReducedMotion } from '~/composables/useReducedMotion'

/** Reward reveal (design §5): the closed chest swaps to open (one 200ms
 * timer, cleared on unmount; at once under reduced motion) and the
 * contents list appears. */
const props = withDefaults(defineProps<{
  items: { icon: string, label: string }[]
  open: boolean
  reduced?: boolean
}>(), { reduced: undefined })

const emit = defineEmits<{ opened: [] }>()

const os = useReducedMotion()
const isReduced = computed(() => props.reduced ?? os.value)

const revealed = ref(false)
let timer: ReturnType<typeof setTimeout> | null = null

function clearTimer() {
  if (timer) { clearTimeout(timer); timer = null }
}

watch(() => props.open, (open) => {
  clearTimer()
  if (!open) { revealed.value = false; return }
  if (isReduced.value) {
    revealed.value = true
    emit('opened')
    return
  }
  timer = setTimeout(() => {
    revealed.value = true
    emit('opened')
  }, 200)
}, { immediate: true })

onBeforeUnmount(clearTimer)
</script>

<template>
  <div class="flex flex-col items-center gap-3">
    <PixelArt
      v-if="!revealed"
      data-glyph="chestClosed"
      :rows="GLYPHS.chestClosed"
      :size="64"
    />
    <PixelArt
      v-else
      data-glyph="chestOpen"
      :rows="GLYPHS.chestOpen"
      :size="64"
    />
    <ul v-if="revealed" aria-live="polite" class="space-y-1">
      <li v-for="(item, i) in items" :key="i" class="font-display text-[22px] leading-6 text-torch">
        {{ item.icon }} {{ item.label }}
      </li>
    </ul>
  </div>
</template>
