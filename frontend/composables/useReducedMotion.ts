import { ref } from 'vue'
import type { Ref } from 'vue'

/**
 * The OS `prefers-reduced-motion` setting (design amend A4). One owner
 * for the media query — kit components read this instead of wiring their
 * own `matchMedia`. Module-level: every caller shares the same ref and
 * there is exactly one `change` listener for the app's lifetime (`ssr:
 * false` is set in `nuxt.config.ts`, so there is no hydration concern).
 * Falls back to `false` when `matchMedia` is missing (older happy-dom).
 */
let osReduced: Ref<boolean> | null = null

export function useReducedMotion(): Readonly<Ref<boolean>> {
  if (!osReduced) {
    osReduced = ref(false)
    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
      const mql = window.matchMedia('(prefers-reduced-motion: reduce)')
      osReduced.value = mql.matches
      mql.addEventListener('change', (e) => {
        if (osReduced) osReduced.value = e.matches
      })
    }
  }
  return osReduced
}
