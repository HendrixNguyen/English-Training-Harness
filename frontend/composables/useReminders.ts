import { ref, type Ref } from 'vue'
import { useSettingsStore } from '~/stores/settings'
import { flattenSubscription, pushSupport, urlBase64ToUint8Array, type FlatSubscription } from '~/utils/push'

export type ReminderState = 'off' | 'requesting' | 'on' | 'denied' | 'unsupported' | 'no-key' | 'error' | 'invalid'

interface Deps {
  vapidPublicKey: string
  win?: typeof window
}

/** Design harness/designs/settings.md §4.1 — the reminder card's state machine over the browser Push API. */
export function useReminders(deps?: Deps): {
  state: Ref<ReminderState>
  init(): void
  enable(time: string): Promise<void>
  disable(): Promise<void>
  saveTime(time: string): Promise<void>
} {
  const win = deps?.win ?? window
  const vapid = deps?.vapidPublicKey ?? ''
  const store = useSettingsStore()
  const state = ref<ReminderState>('off')

  async function pushManager(): Promise<PushManager> {
    const reg = await win.navigator.serviceWorker.ready
    return reg.pushManager
  }

  async function currentSubscription(): Promise<{ sub: PushSubscription, flat: FlatSubscription } | null> {
    if (pushSupport(vapid, win) !== 'ok') return null
    const sub = await (await pushManager()).getSubscription()
    const flat = sub ? flattenSubscription(sub.toJSON()) : null
    return sub && flat ? { sub, flat } : null
  }

  function init() {
    const support = pushSupport(vapid, win)
    if (support !== 'ok') {
      state.value = support
      return
    }
    if (win.Notification.permission === 'denied') {
      state.value = 'denied'
      return
    }
    state.value = store.remindersOn ? 'on' : 'off'
  }

  async function enable(time: string) {
    state.value = 'requesting'
    const permission = await win.Notification.requestPermission()
    if (permission !== 'granted') {
      state.value = permission === 'denied' ? 'denied' : 'off'
      return
    }
    let sub: PushSubscription
    try {
      sub = await (await pushManager()).subscribe({
        userVisibleOnly: true,
        // TS 5.9's lib.dom narrows ArrayBufferView's `buffer` to ArrayBuffer while
        // Uint8Array.from() types its result as Uint8Array<ArrayBufferLike>; the
        // runtime value is a plain Uint8Array, which the Push API accepts fine.
        applicationServerKey: urlBase64ToUint8Array(vapid) as BufferSource,
      })
    } catch {
      state.value = 'error'
      return
    }
    const flat = flattenSubscription(sub.toJSON())
    if (!flat || !(await store.saveReminder(time, flat))) {
      // The server never learned about this subscription: do not leave the
      // browser holding one that will never be sent to.
      await sub.unsubscribe()
      state.value = store.saveError === 'invalid' ? 'invalid' : 'error'
      return
    }
    store.setRemindersOn(true)
    state.value = 'on'
  }

  async function disable() {
    const cur = await currentSubscription()
    if (cur) await cur.sub.unsubscribe()
    // No unsubscribe endpoint: the server prunes this endpoint on its next 404/410.
    store.setRemindersOn(false)
    state.value = 'off'
  }

  async function saveTime(time: string) {
    const cur = store.remindersOn ? await currentSubscription() : null
    const ok = await store.saveReminder(time, cur?.flat ?? null)
    if (!ok) {
      state.value = store.saveError === 'invalid' ? 'invalid' : 'error'
      return
    }
    state.value = store.remindersOn ? 'on' : 'off'
  }

  return { state, init, enable, disable, saveTime }
}
