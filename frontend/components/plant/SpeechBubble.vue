<template>
  <p class="relative mx-auto mt-3 max-w-xs rounded-card border border-ink/10 bg-white px-4 py-2 text-center font-display text-base dark:border-paper/10 dark:bg-ink" aria-live="polite">
    💬 "{{ line }}"
  </p>
</template>

<script setup lang="ts">
defineProps<{ line: string }>()
</script>
