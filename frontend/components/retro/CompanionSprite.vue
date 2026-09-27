<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { tokens } from '~/tailwind.config'
import { healthTone, normalizeStage } from '~/utils/plant'
import { COMPANION, PALETTE } from '~/utils/pixelArt'
import PixelArt from './PixelArt.vue'
import { useReducedMotion } from '~/composables/useReducedMotion'

const props = withDefaults(defineProps<{
  stage: string
  health: number
  name?: string
  size?: number
  crop?: 'full' | 'face'
  react?: 'idle' | 'hit' | 'miss' | 'levelup' | 'down'
  reduced?: boolean
}>(), { name: undefined, size: 128, crop: 'full', react: 'idle', reduced: undefined })

const emit = defineEmits<{ reacted: [react: 'hit' | 'miss' | 'levelup'] }>()

const os = useReducedMotion()
/** design amend A4: an explicit prop overrides the OS setting; Task 5
 * wires the reduced static-reaction hold and `reacted` emit. */
const isReduced = computed(() => props.reduced ?? os.value)

const norm = computed(() => normalizeStage(props.stage))

/** design §4 health tint: reuse `healthTone()`, map its v1 tone names
 * (`streak`/`alert`) onto the v2 hues (`torch`/`ember`). */
const TONE_HEX: Record<'growth' | 'streak' | 'alert', string> = {
  growth: tokens.growth,
  streak: tokens.torch,
  alert: tokens.ember,
}

/** An unknown stage forces `ink-2` on both greens, regardless of health —
 * same fallback v1's `PlantSvg` used. */
const pxColor = computed(() => (norm.value.known ? TONE_HEX[healthTone(props.health)] : tokens['ink-2']))

/** design amend A4 reaction table: `hit`/`miss`/`levelup` hold a single
 * static frame for 300ms under reduced motion, then emit `reacted` — one
 * `setTimeout`, restarted on every `react` change and cleared on unmount.
 * `holding` drives the frame; `idle`/`down` never enter it. */
const holding = ref(false)
let holdTimer: ReturnType<typeof setTimeout> | null = null

function clearHoldTimer() {
  if (holdTimer) { clearTimeout(holdTimer); holdTimer = null }
}

/** Folded bug companionsprite-motion-is-off-the-pixel-grid: non-reduced
 * `levelup` is a 3-frame palette swap (torch -> ink-0 -> base tone) stepped
 * on `setTimeout(100)`, replacing the old CSS filter keyframe class this
 * reaction used (deleted from retro.css). `null` is frame 2 / the base tone
 * — no override. All three timers are cleared on every `react` change and
 * on unmount. */
const levelupFrame = ref<0 | 1 | null>(null)
let levelupTimers: ReturnType<typeof setTimeout>[] = []

function clearLevelupTimers() {
  for (const t of levelupTimers) clearTimeout(t)
  levelupTimers = []
}

watch([() => props.react, isReduced], ([react, reduced]) => {
  clearHoldTimer()
  clearLevelupTimers()
  holding.value = false
  levelupFrame.value = null
  if (reduced && (react === 'hit' || react === 'miss' || react === 'levelup')) {
    holding.value = true
    holdTimer = setTimeout(() => {
      holding.value = false
      emit('reacted', react)
    }, 300)
  } else if (!reduced && react === 'levelup') {
    levelupFrame.value = 0
    levelupTimers.push(setTimeout(() => { levelupFrame.value = 1 }, 100))
    levelupTimers.push(setTimeout(() => { levelupFrame.value = null }, 200))
    levelupTimers.push(setTimeout(() => { emit('reacted', 'levelup') }, 300))
  }
}, { immediate: true })

onBeforeUnmount(() => {
  clearHoldTimer()
  clearLevelupTimers()
})

const palette = computed(() => {
  const out: Record<string, string> = {}
  for (const [char, roleName] of Object.entries(PALETTE)) out[char] = (tokens as Record<string, string>)[roleName]
  out.g = pxColor.value
  out.G = pxColor.value
  if (holding.value && props.react === 'levelup') {
    for (const char of Object.keys(PALETTE)) if (char !== 'k') out[char] = tokens.torch
  }
  if (levelupFrame.value === 0) {
    for (const char of Object.keys(PALETTE)) if (char !== 'k') out[char] = tokens.torch
  } else if (levelupFrame.value === 1) {
    for (const char of Object.keys(PALETTE)) if (char !== 'k') out[char] = tokens['ink-0']
  }
  return out
})

const isDown = computed(() => props.react === 'down')
const rows = computed(() => (isDown.value ? COMPANION.wilted : COMPANION[norm.value.stage]))
/** design retro-kit.md §4 `down`: the rotation lives on the drawing itself
 * (PixelArt's `transform`), not the wrapper — so it survives the wrapper's
 * own `--px`-based transforms (folded bug). */
const downTransform = 'rotate(90 16 16) translate(0 7)'

/** design §4 reaction table. `hit`/`miss`/`levelup` are transient (they
 * emit `reacted` and the caller returns `react` to `idle`); `reduced`
 * collapses every reaction to a single static frame — a frame swap, not a
 * keyframe (design's motion budget: "the reduced variant is designed, not
 * disabled"). */
const animClass = computed(() => {
  if (isReduced.value || isDown.value) return ''
  if (props.react === 'idle') return props.stage === 'wilted' ? '' : 'retro-breath'
  // levelup is now a JS-stepped palette swap (see `levelupFrame`), not a CSS class.
  return { hit: 'retro-hop', miss: 'retro-shake', levelup: '', down: '' }[props.react]
})

/** Folded bug companionsprite-motion-is-off-the-pixel-grid: `--px` is the
 * sprite's own pixel unit (design's "unit" — 4px at 128px/full, 3px at
 * 48px/face, …), computed from the snapped size PixelArt itself renders at,
 * not the raw `size` prop (which may not land on a whole multiple of the
 * grid). Every consumer of the companion (SpeechBox at 48px/face, the hub
 * at 64/128px/full) gets a `--px` that matches what is actually drawn. */
const scale = computed(() => {
  const unit = props.crop === 'face' ? 16 : 32
  const snapped = Math.max(unit, Math.floor(props.size / unit) * unit)
  return snapped / unit
})

const wrapperStyle = computed(() => {
  const style: Record<string, string> = { '--px': String(scale.value) }
  if (holding.value && props.react === 'hit') style.transform = 'translateY(calc(var(--px) * -2px))'
  return style
})

/** `hit`/`levelup` show frame 1 while holding; `miss`'s static frame is
 * the base (design amend A4). Non-reduced keeps the legacy `1`. */
const dataFrame = computed(() => {
  if (!isReduced.value) return 1
  return holding.value && (props.react === 'hit' || props.react === 'levelup') ? 1 : 0
})

const label = computed(() => {
  const named = props.name ? `${props.name}, ` : ''
  const down = isDown.value ? ', đã gục' : ''
  return `${named}giai đoạn ${norm.value.stage}, ${Math.max(0, props.health)} HP${down}`
})

function onAnimationend() {
  if (props.react === 'hit' || props.react === 'miss' || props.react === 'levelup') emit('reacted', props.react)
}
</script>

<template>
  <div
    class="inline-block"
    :class="animClass"
    :style="wrapperStyle"
    role="img"
    :aria-label="label"
    :data-stage="norm.stage"
    :data-react="react"
    :data-frame="dataFrame"
    @animationend="onAnimationend"
  >
    <PixelArt
      :rows="rows"
      :palette="palette"
      :size="size"
      :view-box="crop === 'face' ? '8 16 16 16' : undefined"
      :transform="isDown ? downTransform : undefined"
    />
  </div>
</template>
