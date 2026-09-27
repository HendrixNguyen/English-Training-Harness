import { ref } from 'vue'

export interface RetroToastItem {
  line: string
  tone?: 'plain' | 'growth' | 'torch' | 'ember'
}

/**
 * Module-level queue (design §5): "one at a time" needs one owner. Pages
 * mount a single `<RetroToast />` (plan 2+) and call `show()` from
 * anywhere via this composable.
 */
const queue = ref<RetroToastItem[]>([])
let timer: ReturnType<typeof setTimeout> | null = null

function clearTimer() {
  if (timer) { clearTimeout(timer); timer = null }
}

function scheduleDismiss() {
  clearTimer()
  timer = setTimeout(dismiss, 2000)
}

function show(line: string, tone: RetroToastItem['tone'] = 'plain') {
  queue.value.push({ line, tone })
  if (queue.value.length === 1) scheduleDismiss()
}

function dismiss() {
  clearTimer()
  queue.value.shift()
  if (queue.value.length > 0) scheduleDismiss()
}

export function useRetroToast() {
  return { queue, show, dismiss }
}
