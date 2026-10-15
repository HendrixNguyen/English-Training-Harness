<script setup lang="ts">
import { nextTick, onMounted, onBeforeUnmount, ref } from 'vue'
withDefaults(defineProps<{ title: string; confirmLabel: string; cancelLabel: string; loading?: boolean }>(), { loading: false })
const emit = defineEmits<{ confirm: []; cancel: [] }>()
const dialog = ref<HTMLElement | null>(null)
let previous: HTMLElement | null = null
function key(event: KeyboardEvent) {
 if (event.key === 'Escape') { event.preventDefault(); emit('cancel') }
 if (event.key !== 'Tab') return
 const controls = Array.from(dialog.value?.querySelectorAll<HTMLElement>('button:not(:disabled), [href], input:not(:disabled), [tabindex="0"]') ?? [])
 const first = controls[0]; const last = controls.at(-1)
 if (!first) { event.preventDefault(); dialog.value?.focus(); return }
 if (event.shiftKey && (document.activeElement === first || document.activeElement === dialog.value)) { event.preventDefault(); last?.focus() }
 else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
}
onMounted(async () => { previous = document.activeElement as HTMLElement; await nextTick(); dialog.value?.querySelector<HTMLElement>('button')?.focus() })
onBeforeUnmount(() => previous?.focus())
</script>
<template>
  <div class="fixed inset-0 z-50 flex items-end justify-center bg-ground-0/80 p-4" @keydown="key">
    <div ref="dialog" role="dialog" aria-modal="true" aria-labelledby="signout-title" tabindex="-1" class="w-full max-w-md pb-[env(safe-area-inset-bottom)]">
      <RetroPanel tone="ember" :announce="false">
        <h2 id="signout-title" class="font-display text-[28px]">{{ title }}</h2>
        <p class="my-4 font-body text-base"><slot /></p>
        <div class="flex flex-col gap-3">
          <RetroButton variant="secondary" :disabled="loading" @click="emit('cancel')">{{ cancelLabel }}</RetroButton>
          <RetroButton variant="danger" :loading="loading" @click="emit('confirm')">{{ confirmLabel }}</RetroButton>
        </div>
      </RetroPanel>
    </div>
  </div>
</template>
