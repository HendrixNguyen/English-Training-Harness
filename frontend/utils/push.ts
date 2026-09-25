/** Client-side Web Push helpers (the service worker's receive side lives in service-worker/push.ts). */

export type PushSupport = 'ok' | 'unsupported' | 'no-key'

/** no-key hides the reminders switch (design §4.1); unsupported explains "Add to Home Screen". */
export function pushSupport(
  vapidPublicKey: string,
  w: Pick<Window, 'navigator'> & { PushManager?: unknown, Notification?: unknown } = window,
): PushSupport {
  if (!vapidPublicKey) return 'no-key'
  if (!('serviceWorker' in w.navigator) || !w.PushManager || !w.Notification) return 'unsupported'
  return 'ok'
}

/** applicationServerKey wants raw bytes; VAPID keys are published base64url. */
export function urlBase64ToUint8Array(b64url: string): Uint8Array {
  const padded = b64url + '='.repeat((4 - (b64url.length % 4)) % 4)
  const b64 = padded.replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(b64)
  return Uint8Array.from(raw, c => c.charCodeAt(0))
}

/** Backend spec §6.4 push_subscription — flat keys, not the browser's nested `keys`. */
export interface FlatSubscription { endpoint: string, p256dh: string, auth: string }

export function flattenSubscription(json: PushSubscriptionJSON): FlatSubscription | null {
  const endpoint = json.endpoint
  const p256dh = json.keys?.p256dh
  const auth = json.keys?.auth
  if (!endpoint || !p256dh || !auth) return null
  return { endpoint, p256dh, auth }
}
