<script setup lang="ts">
defineProps<{ label: string, emoji: string, selected: boolean }>()
defineEmits<{ select: [] }>()
</script>

<template>
  <button
    type="button"
    role="radio"
    :aria-checked="selected"
    class="flex h-24 flex-1 flex-col items-center justify-center gap-1 rounded-sm border-2 text-center font-display text-[22px] leading-6"
    :class="selected ? 'border-growth bg-ground-2' : 'border-line-lit bg-ground-1'"
    @click="$emit('select')"
  >
    <span class="text-2xl" aria-hidden="true">{{ emoji }}</span>
    <span><span v-if="selected" aria-hidden="true">▶ </span>{{ label }}</span>
  </button>
</template>
