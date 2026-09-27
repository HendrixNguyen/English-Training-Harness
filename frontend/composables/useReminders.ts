import { ref, type Ref } from 'vue'
import { useSettingsStore } from '~/stores/settings'
import { flattenSubscription, pushSupport, urlBase64ToUint8Array, type FlatSubscription } from '~/utils/push'

export type ReminderState = 'off' | 'requesting' | 'on' | 'denied' | 'unsupported' | 'no-key'
/** How the last save went, independent of `state`. Null clears the alert line. */
export type SaveProblem = 'error' | 'invalid' | null

interface Deps {
  vapidPublicKey: string
  win?: typeof window
}

/**
 * Design harness/designs/settings.md §4.1 — the reminder card's state machine over the browser Push API.
 * `state` is what the device can do and whether the switch is on; `problem` is how the last save went.
 * They are independent refs on purpose: a save result must never overwrite `no-key`/`unsupported`/`denied`,
 * and a failed save must never flip an `on` switch to `off` (amend, 2026-09-27).
 */
export function useReminders(deps?: Deps): {
  state: Ref<ReminderState>
  problem: Ref<SaveProblem>
  ready: Ref<boolean>
  init(): Promise<void>
  enable(time: string): Promise<void>
  disable(): Promise<void>
  saveTime(time: string): Promise<void>
} {
  const win = deps?.win ?? window
  const vapid = deps?.vapidPublicKey ?? ''
  const store = useSettingsStore()
  const state = ref<ReminderState>('off')
  const problem = ref<SaveProblem>(null)
  const ready = ref(false)

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

  async function init() {
    try {
      const support = pushSupport(vapid, win)
      if (support !== 'ok') {
        state.value = support
        return
      }
      if (win.Notification.permission === 'denied') {
        state.value = 'denied'
        return
      }
      // Trust localStorage's remindersOn only as far as the browser still agrees:
      // confirm a real subscription before showing `on`.
      const cur = store.remindersOn ? await currentSubscription() : null
      if (store.remindersOn && !cur) store.setRemindersOn(false)
      state.value = cur ? 'on' : 'off'
    } finally {
      ready.value = true
    }
  }

  async function enable(time: string) {
    // Refuse where push cannot work at all: no key/support, or the browser already blocks it.
    if (pushSupport(vapid, win) !== 'ok' || win.Notification.permission === 'denied') return
    problem.value = null
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
      state.value = 'off'
      problem.value = 'error'
      return
    }
    const flat = flattenSubscription(sub.toJSON())
    if (!flat || !(await store.saveReminder(time, flat))) {
      // The server never learned about this subscription: do not leave the
      // browser holding one that will never be sent to. The switch returns to
      // its previous position, off.
      await sub.unsubscribe()
      state.value = 'off'
      problem.value = store.saveError === 'invalid' ? 'invalid' : 'error'
      return
    }
    store.setRemindersOn(true)
    state.value = 'on'
  }

  async function disable() {
    problem.value = null
    const cur = await currentSubscription()
    if (cur) await cur.sub.unsubscribe()
    // No unsubscribe endpoint: the server prunes this endpoint on its next 404/410.
    store.setRemindersOn(false)
    state.value = 'off'
  }

  async function saveTime(time: string) {
    const cur = state.value === 'on' ? await currentSubscription() : null
    if (state.value === 'on' && !cur) {
      // The browser dropped the subscription since init: a fact about the device, the one place
      // saveTime writes `state`.
      store.setRemindersOn(false)
      state.value = 'off'
    }
    const ok = await store.saveReminder(time, cur?.flat ?? null)
    problem.value = ok ? null : (store.saveError === 'invalid' ? 'invalid' : 'error')
  }

  return { state, problem, ready, init, enable, disable, saveTime }
}
