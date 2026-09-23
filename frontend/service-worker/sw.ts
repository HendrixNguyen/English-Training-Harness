/// <reference lib="webworker" />
import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching'
import { registerRoute } from 'workbox-routing'
import { NetworkFirst, StaleWhileRevalidate } from 'workbox-strategies'
import { parsePushPayload } from './push'

declare let self: ServiceWorkerGlobalScope

// App shell: routes, fonts, plant SVGs, icons (Frontend spec §3 "Static Assets").
precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()

// §3 "User Progress & Pet Status: NetworkFirst" — synchronised state when
// online, last-known state when not. Cache name must match utils/session.ts.
registerRoute(
  ({ url, request }) => request.method === 'GET' && /\/api\/v1\/(quests\/daily|pet\/status)$/.test(url.pathname),
  new NetworkFirst({ cacheName: 'api-state', networkTimeoutSeconds: 5 }),
)

// §3 "Static Assets & Exercises: StaleWhileRevalidate" for anything the
// precache manifest did not cover (hashed chunks loaded later, fonts).
registerRoute(
  ({ request }) => ['style', 'script', 'font', 'image'].includes(request.destination),
  new StaleWhileRevalidate({ cacheName: 'assets' }),
)

self.addEventListener('push', (event) => {
  const payload = parsePushPayload(event.data?.text() ?? null)
  event.waitUntil(
    self.registration.showNotification(payload.title, {
      body: payload.body,
      icon: '/pwa-192x192.png',
      data: { url: payload.url },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = (event.notification.data as { url?: string } | undefined)?.url ?? '/'
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(async (clients) => {
      const existing = clients.find(c => 'focus' in c)
      if (existing) {
        await existing.focus()
        await existing.navigate(url)
        return
      }
      await self.clients.openWindow(url)
    }),
  )
})
