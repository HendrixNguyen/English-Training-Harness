<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import RetroPanel from './RetroPanel.vue'
import CompanionSprite from './CompanionSprite.vue'

/**
 * `RetroPanel` with the companion's portrait and the line typing in at
 * 30ms/char (design §5). A tap anywhere settles it early; the full line is
 * always in a visually-hidden `aria-live` span so a screen reader hears it
 * once, whichever pace it appeared at.
 */
const props = withDefaults(defineProps<{
  line: string
  name?: string
  stage?: string
  health?: number
  reduced?: boolean
}>(), { name: undefined, stage: 'sprout', health: 100, reduced: false })

const emit = defineEmits<{ settled: [] }>()

const visibleChars = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

function clearTimer() {
  if (timer) { clearInterval(timer); timer = null }
}

function startTyping() {
  clearTimer()
  if (props.reduced) {
    visibleChars.value = props.line.length
    if (props.line.length > 0) emit('settled')
    return
  }
  visibleChars.value = 0
  if (props.line.length === 0) return
  timer = setInterval(() => {
    visibleChars.value += 1
    if (visibleChars.value >= props.line.length) {
      clearTimer()
      emit('settled')
    }
  }, 30)
}

watch(() => props.line, startTyping, { immediate: true })
onBeforeUnmount(clearTimer)

const typed = computed(() => props.line.slice(0, visibleChars.value))
const settled = computed(() => visibleChars.value >= props.line.length)

function revealAll() {
  if (settled.value) return
  clearTimer()
  visibleChars.value = props.line.length
  emit('settled')
}
</script>

<template>
  <div @click="revealAll">
    <RetroPanel :speaker="name">
      <template #portrait>
        <CompanionSprite crop="face" :size="48" :stage="stage" :health="health" />
      </template>
      <p class="font-body text-[17px] text-ink-0">
        <span data-typed aria-hidden="true">{{ typed }}</span>
        <span data-live-line class="sr-only" aria-live="polite">{{ line }}</span>
      </p>
    </RetroPanel>
  </div>
</template>

<style scoped>
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
