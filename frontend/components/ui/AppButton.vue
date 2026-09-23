<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'danger' | 'ghost'
  loading?: boolean
  disabled?: boolean
  type?: 'button' | 'submit'
  block?: boolean
}>(), { variant: 'primary', loading: false, disabled: false, type: 'button', block: false })

const classes = computed(() => [
  'inline-flex h-12 items-center justify-center gap-2 rounded-btn px-5 font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50',
  props.block ? 'w-full' : '',
  {
    primary: 'bg-growth text-white hover:bg-growth/90',
    danger: 'bg-alert text-white hover:bg-alert/90',
    ghost: 'bg-transparent text-ink hover:bg-ink/5 dark:text-paper dark:hover:bg-paper/10',
  }[props.variant],
])
</script>

<template>
  <button :type="type" :class="classes" :disabled="disabled || loading" :aria-busy="loading || undefined">
    <span v-if="loading" class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
    <span :class="{ 'opacity-0': loading }"><slot /></span>
  </button>
</template>
