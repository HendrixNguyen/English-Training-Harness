<script setup lang="ts">
defineProps<{ text: string, tone: 'growth' | 'streak' }>()
</script>

<template>
  <span
    :data-growth-chip="tone"
    class="inline-flex rounded-full px-3 py-1 text-sm font-semibold tabular-nums"
    :class="tone === 'growth' ? 'bg-growth/15 text-growth' : 'bg-streak/15 text-streak'"
  >{{ text }}</span>
</template>
