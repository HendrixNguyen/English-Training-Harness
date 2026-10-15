<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import Badge from '~/components/retro/Badge.vue'
defineProps<{ streak?: number | null, pulse?: boolean }>()
const auth = useAuthStore()
</script>

<template>
  <header class="flex min-h-16 items-center justify-between gap-2 py-3 text-ink-0">
    <NuxtLink to="/settings" class="flex min-h-12 min-w-0 items-center gap-2" :aria-label="`Cài đặt tài khoản ${auth.user?.full_name || ''}`">
      <span class="flex size-12 shrink-0 items-center justify-center border-2 border-line-lit bg-ground-2 font-display text-2xl">{{ auth.initial }}</span>
      <span class="truncate font-display text-[22px]">{{ auth.user?.full_name }}</span>
    </NuxtLink>
    <NuxtLink to="/roadmap" class="flex min-h-12 shrink-0 items-center gap-1 text-sm" aria-label="Xem hành trình">
      <span v-if="streak !== null && streak !== undefined" data-streak-chip :class="{ 'streak-pulse': pulse }" class="flex items-center gap-1">
        <Badge kind="streak" :count="streak" :earned="streak > 0" />
        <span class="sr-only">Chuỗi ngày: {{ streak }}</span>
      </span>
      <span>Hành trình</span>
    </NuxtLink>
  </header>
</template>

<style scoped>
.streak-pulse { outline: 2px solid theme('colors.torch'); outline-offset: 2px; }
</style>
