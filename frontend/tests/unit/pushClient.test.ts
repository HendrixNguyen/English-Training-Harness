import { describe, expect, it } from 'vitest'
import { flattenSubscription, pushSupport, urlBase64ToUint8Array } from '~/utils/push'

describe('urlBase64ToUint8Array', () => {
  it('decodes base64url (no padding, -_ alphabet) to the raw bytes', () => {
    // "BAgQ" is base64 for [0x04, 0x08, 0x10]; base64url may drop '=' padding
    expect(Array.from(urlBase64ToUint8Array('BAgQ'))).toEqual([4, 8, 16])
    expect(Array.from(urlBase64ToUint8Array('-_8'))).toEqual([251, 255])
  })
})

describe('flattenSubscription', () => {
  it('maps the browser shape to the backend flat shape', () => {
    expect(flattenSubscription({ endpoint: 'https://push.example/abc', keys: { p256dh: 'BNc5', auth: 'aX8v' } }))
      .toEqual({ endpoint: 'https://push.example/abc', p256dh: 'BNc5', auth: 'aX8v' })
  })
  it('returns null when any field is missing', () => {
    expect(flattenSubscription({ endpoint: 'https://push.example/abc', keys: { p256dh: 'BNc5' } })).toBeNull()
    expect(flattenSubscription({ keys: { p256dh: 'a', auth: 'b' } })).toBeNull()
  })
})

describe('pushSupport', () => {
  const full = { navigator: { serviceWorker: {} } as Navigator, PushManager: function () {}, Notification: function () {} }
  it('is ok when serviceWorker, PushManager and Notification exist and a key is configured', () => {
    expect(pushSupport('BKey', full)).toBe('ok')
  })
  it('is no-key when the VAPID public key is empty, whatever the browser', () => {
    expect(pushSupport('', full)).toBe('no-key')
  })
  it('is unsupported without PushManager (iOS Safari outside an installed PWA) or without serviceWorker', () => {
    expect(pushSupport('BKey', { ...full, PushManager: undefined })).toBe('unsupported')
    expect(pushSupport('BKey', { ...full, navigator: {} as Navigator })).toBe('unsupported')
  })
})
