<template>
  <section class="rounded-card border border-ink/10 bg-white p-4 dark:border-paper/10 dark:bg-ink">
    <h2 v-if="title" class="mb-3 text-xs font-semibold uppercase tracking-wider text-mute">
      {{ title }}
    </h2>
    <slot />
  </section>
</template>

<script setup lang="ts">
defineProps<{ title?: string }>()
</script>
