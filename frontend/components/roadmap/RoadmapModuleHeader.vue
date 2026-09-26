<script setup lang="ts">
import type { RoadmapModule } from '~/stores/roadmap'

defineProps<{ module: RoadmapModule, met: number }>()
</script>

<template>
  <div class="flex items-start gap-3 pt-6 first:pt-0">
    <span aria-hidden="true" class="mt-1 size-3 shrink-0 rounded-full bg-ink dark:bg-paper" />
    <div class="min-w-0 flex-1">
      <div class="flex items-baseline justify-between gap-2">
        <p class="min-w-0">
          <span class="text-xs font-semibold uppercase tracking-wider text-mute">Tuần {{ module.week }} · </span>
          <span class="text-base font-semibold">{{ module.title }}</span>
        </p>
        <span class="shrink-0 text-[13px] tabular-nums text-mute">{{ met }}/7</span>
      </div>
      <p class="text-[13px] leading-[18px] text-mute">
        {{ module.focus }}
      </p>
    </div>
  </div>
</template>
