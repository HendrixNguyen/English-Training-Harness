<script setup lang="ts">
defineProps<{ text: string, active: boolean, controls: string }>()
defineEmits<{ toggle: [event: MouseEvent] }>()
</script>

<template>
  <button
    type="button"
    class="font-[inherit] leading-[inherit] text-ink-0 underline decoration-2 decoration-line-lit underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch"
    :class="active ? 'bg-ground-2 decoration-solid' : 'decoration-dotted'"
    :aria-label="`Giải nghĩa ${text}`"
    :aria-expanded="active"
    :aria-controls="controls"
    @click="$emit('toggle', $event)"
  >{{ text }}</button>
</template>
