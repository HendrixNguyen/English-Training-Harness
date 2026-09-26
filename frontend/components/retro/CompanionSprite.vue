<script setup lang="ts">
import { computed } from 'vue'
import { tokens } from '~/tailwind.config'
import { healthTone, normalizeStage } from '~/utils/plant'
import { COMPANION, PALETTE } from '~/utils/pixelArt'
import PixelArt from './PixelArt.vue'

const props = withDefaults(defineProps<{
  stage: string
  health: number
  name?: string
  size?: number
  crop?: 'full' | 'face'
  react?: 'idle' | 'hit' | 'miss' | 'levelup' | 'down'
  reduced?: boolean
}>(), { name: undefined, size: 128, crop: 'full', react: 'idle', reduced: false })

const emit = defineEmits<{ reacted: [react: 'hit' | 'miss' | 'levelup'] }>()

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

const palette = computed(() => {
  const out: Record<string, string> = {}
  for (const [char, roleName] of Object.entries(PALETTE)) out[char] = (tokens as Record<string, string>)[roleName]
  out.g = pxColor.value
  out.G = pxColor.value
  return out
})

const isDown = computed(() => props.react === 'down')
const rows = computed(() => (isDown.value ? COMPANION.wilted : COMPANION[norm.value.stage]))

/** design §4 reaction table. `hit`/`miss`/`levelup` are transient (they
 * emit `reacted` and the caller returns `react` to `idle`); `reduced`
 * collapses every reaction to a single static frame — a frame swap, not a
 * keyframe (design's motion budget: "the reduced variant is designed, not
 * disabled"). */
const animClass = computed(() => {
  if (props.reduced || isDown.value) return ''
  if (props.react === 'idle') return props.stage === 'wilted' ? '' : 'retro-breath'
  return { hit: 'retro-hop', miss: 'retro-shake', levelup: 'retro-flash', down: '' }[props.react]
})

const wrapperStyle = computed(() => (isDown.value ? { transform: 'rotate(90deg)' } : {}))

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
    :data-frame="reduced ? 0 : 1"
    @animationend="onAnimationend"
  >
    <PixelArt
      :rows="rows"
      :palette="palette"
      :size="size"
      :view-box="crop === 'face' ? '8 16 16 16' : undefined"
    />
  </div>
</template>
