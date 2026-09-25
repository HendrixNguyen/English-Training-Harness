<script setup lang="ts">
defineProps<{ modelValue: boolean, disabled?: boolean, busy?: boolean, labelledby: string }>()
const emit = defineEmits<{ 'update:modelValue': [boolean] }>()
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :aria-disabled="disabled || busy || undefined"
    :aria-busy="busy || undefined"
    :aria-labelledby="labelledby"
    class="relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors duration-200 motion-reduce:transition-none disabled:cursor-not-allowed"
    :class="modelValue ? 'bg-growth' : 'bg-mute/30'"
    :disabled="disabled || busy"
    @click="emit('update:modelValue', !modelValue)"
  >
    <span
      class="inline-block size-5 rounded-full bg-white shadow-sm transition-transform duration-200 motion-reduce:transition-none"
      :class="modelValue ? 'translate-x-[22px]' : 'translate-x-0.5'"
      aria-hidden="true"
    />
  </button>
</template>
