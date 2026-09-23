<script setup lang="ts">
defineProps<{ label: string, emoji: string, selected: boolean }>()
defineEmits<{ select: [] }>()
</script>

<template>
  <button
    type="button"
    role="radio"
    :aria-checked="selected"
    class="flex h-24 flex-1 flex-col items-center justify-center gap-1 rounded-card border-2 text-center font-semibold"
    :class="selected ? 'border-growth bg-growth/10' : 'border-ink/15 dark:border-paper/15'"
    @click="$emit('select')"
  >
    <span class="text-2xl" aria-hidden="true">{{ emoji }}</span>
    <span>{{ label }}</span>
  </button>
</template>
