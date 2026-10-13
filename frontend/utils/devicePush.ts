import type { DeviceSubscription } from '~/stores/settings'
export type PushState = 'unsupported' | 'no-key' | 'denied' | 'off' | 'requesting' | 'on' | 'error'
export function pushCapability(configured: boolean, key: string): PushState {
 if (typeof Notification === 'undefined' || typeof PushManager === 'undefined' || !navigator.serviceWorker) return 'unsupported'
 if (!configured || !key) return 'no-key'
 return Notification.permission === 'denied' ? 'denied' : 'off'
}
export async function subscribeDevice(key: string, current: () => boolean): Promise<DeviceSubscription | null> {
 if (await Notification.requestPermission() !== 'granted' || !current()) return null
 let timer: ReturnType<typeof setTimeout> | undefined
 try {
  const registration = await Promise.race([navigator.serviceWorker.ready, new Promise<never>((_, reject) => { timer = setTimeout(() => reject(new Error('worker_timeout')), 5000) })])
  if (!current()) return null
  const encoded = atob(key.replace(/-/g, '+').replace(/_/g, '/'))
  const applicationServerKey = Uint8Array.from(encoded, char => char.charCodeAt(0))
  let subscription = await registration.pushManager.getSubscription()
  if (!current()) return null
  if (!subscription) subscription = await registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey })
  if (!current()) return null
  const json = subscription.toJSON()
  if (!json.endpoint || !json.keys?.p256dh || !json.keys.auth) throw new Error('invalid_subscription')
  return { endpoint: json.endpoint, p256dh: json.keys.p256dh, auth: json.keys.auth }
 } finally { if (timer) clearTimeout(timer) }
}
