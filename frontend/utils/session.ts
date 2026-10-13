/** Name of the service worker's NetworkFirst cache for /quests/daily and /pet/status (sw.ts). */
import { getActivePinia, setActivePinia } from 'pinia'
export const API_STATE_CACHE = 'api-state'

/** Drop cached per-user API responses; called on sign-out and on a 401. */
export async function clearApiCache(): Promise<void> {
  if (typeof caches === 'undefined') return
  await caches.delete(API_STATE_CACHE)
}

/** Central auth cleanup without importing API-backed stores into authentication. */
const registered = new WeakSet<object>()
const privateMemory = new Set<WeakRef<{ invalidate(): void }>>()
export function registerPrivateMemory(store: { invalidate(): void }) {
  if (registered.has(store)) return
  registered.add(store)
  privateMemory.add(new WeakRef(store))
}
export function invalidatePrivateMemory() {
  // Pinia actions activate their owning instance. Restore the caller's instance
  // after invalidating stores from older mounts so the new owner's stores cannot
  // accidentally bind to an older session's Pinia tree.
  const callerPinia = getActivePinia()
  try {
    for (const reference of privateMemory) {
      const store = reference.deref()
      if (store) store.invalidate()
      else privateMemory.delete(reference)
    }
  } finally {
    if (callerPinia) setActivePinia(callerPinia)
  }
}
