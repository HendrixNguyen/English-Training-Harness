/**
 * Strips the sliding-session headers (X-Session-Token, X-Session-Expires-In)
 * before a NetworkFirst response for /quests/daily or /pet/status is written
 * to the api-state cache (design harness/designs/stay-signed-in.md §4
 * "Offline"). A cache-served response must never carry a token that is newer
 * than the one currently in the auth store — otherwise an offline reload
 * would look like a renewal happened when it did not. Wired into sw.ts as a
 * workbox `cacheWillUpdate` plugin; split out here (like push.ts) so it is a
 * plain function a unit test can call without booting a service worker.
 */
export function stripSessionHeaders(response: Response): Response {
  const headers = new Headers(response.headers)
  headers.delete('X-Session-Token')
  headers.delete('X-Session-Expires-In')
  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  })
}
