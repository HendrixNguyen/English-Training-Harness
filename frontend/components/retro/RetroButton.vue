<script setup lang="ts">
import { computed } from 'vue'
import { tokens } from '~/tailwind.config'

/**
 * 48-px button (design §5). Depth is a hard, sharp-edged bottom shadow in the
 * variant's `-deep` colour; pressed drops it to 2px with no transition
 * (design §3 "pressed within 100ms: no transitions on press"). `loading`
 * keeps the label in the DOM at `opacity-0` (so the button's width never
 * jumps) and shows stepping dots over it.
 */
const props = withDefaults(defineProps<{
  type?: 'button' | 'submit'
  variant?: 'primary' | 'secondary' | 'danger'
  loading?: boolean
  disabled?: boolean
  block?: boolean
}>(), { type: 'button', variant: 'primary', loading: false, disabled: false, block: false })

const VARIANT = {
  primary: { classes: 'bg-growth text-ground-0', shadow: tokens['growth-deep'] },
  secondary: { classes: 'bg-ground-2 text-ink-0 border-2 border-line-lit', shadow: tokens['line-dim'] },
  danger: { classes: 'bg-ember text-ground-0', shadow: tokens['ember-deep'] },
} as const

const isDisabled = computed(() => props.disabled || props.loading)
const style = computed(() => ({ '--rb-shadow': VARIANT[props.variant].shadow }))
</script>

<template>
  <button
    :type="type"
    class="rb relative inline-flex h-12 min-w-[48px] items-center justify-center rounded-sm px-5 font-display text-[22px] leading-6 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch disabled:opacity-50 disabled:shadow-none"
    :class="[VARIANT[variant].classes, block ? 'w-full' : '']"
    :style="style"
    :disabled="isDisabled"
    :aria-busy="loading ? 'true' : undefined"
    :aria-disabled="disabled ? 'true' : undefined"
  >
    <span data-label :class="{ 'opacity-0': loading }"><slot /></span>
    <span v-if="loading" class="absolute inset-0 flex items-center justify-center" aria-hidden="true">
      <span class="retro-dots font-display text-[22px]">&hellip;</span>
    </span>
  </button>
</template>

<style scoped>
.rb {
  box-shadow: 0 4px 0 0 var(--rb-shadow, transparent);
}
.rb:active:not(:disabled) {
  transform: translateY(2px);
  box-shadow: 0 2px 0 0 var(--rb-shadow, transparent);
}
</style>
