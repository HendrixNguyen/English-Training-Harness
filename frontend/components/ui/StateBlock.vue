<script setup lang="ts">
defineProps<{
  state: 'loading' | 'empty' | 'error'
  message?: string
  action?: string
}>()
defineEmits<{ action: [] }>()
</script>

<template>
  <div v-if="state === 'loading'" class="animate-pulse space-y-3" aria-busy="true" aria-label="Đang tải">
    <div class="h-4 w-2/3 rounded bg-mute/20" />
    <div class="h-4 w-1/2 rounded bg-mute/20" />
    <div class="h-4 w-3/4 rounded bg-mute/20" />
  </div>
  <div v-else class="flex flex-col items-start gap-3" :class="state === 'error' ? 'text-alert' : 'text-mute'" role="status">
    <p class="text-base">
      {{ message }}
    </p>
    <AppButton v-if="action" :variant="state === 'error' ? 'danger' : 'primary'" @click="$emit('action')">
      {{ action }}
    </AppButton>
  </div>
</template>
