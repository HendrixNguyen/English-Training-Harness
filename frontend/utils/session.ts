/** Name of the service worker's NetworkFirst cache for /quests/daily and /pet/status (sw.ts). */
export const API_STATE_CACHE = 'api-state'

/** Drop cached per-user API responses; called on sign-out and on a 401. */
export async function clearApiCache(): Promise<void> {
  if (typeof caches === 'undefined') return
  await caches.delete(API_STATE_CACHE)
}
