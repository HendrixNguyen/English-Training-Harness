<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useReminders } from '~/composables/useReminders'
import { useSettingsStore } from '~/stores/settings'
import { googleAuthUrl, OAUTH_STATE_KEY, randomState } from '~/utils/googleAuth'

const config = useRuntimeConfig()
const settings = useSettingsStore()
const reminders = useReminders({ vapidPublicKey: config.public.vapidPublicKey })

// Transient (unpersisted) alerts from a previous visit — e.g. a sync error, or
// landing back here after the Google re-consent hand-off — should not linger
// across a fresh visit to this page; reset before the first render.
settings.saveError = null
settings.syncError = null

const time = ref('20:00')
onMounted(() => {
  settings.hydrate()
  time.value = settings.notificationTime
  reminders.init()
})

const timeChanged = computed(() => time.value !== settings.notificationTime)
const switchOn = computed(() => reminders.state.value === 'on' || reminders.state.value === 'requesting')
const showSwitch = computed(() => !['unsupported', 'no-key'].includes(reminders.state.value))

/** Design §4.1 status lines, one per state. */
const reminderLine = computed(() => ({
  off: 'Đang tắt · Bạn sẽ không nhận thông báo.',
  requesting: 'Đang xin phép trình duyệt…',
  on: `Đang nhắc lúc ${settings.notificationTime} mỗi ngày.`,
  denied: 'Trình duyệt đang chặn thông báo. Mở cài đặt trang web, cho phép Thông báo, rồi thử lại.',
  unsupported: 'Thiết bị này chưa nhắc được qua trình duyệt. Trên iPhone: Chia sẻ → Thêm vào Màn hình chính, rồi mở lại.',
  'no-key': '',
  error: 'Không lưu được. Thử lại.',
  invalid: 'Giờ nhắc không hợp lệ.',
} as Record<string, string>)[reminders.state.value] ?? '')
const reminderIsAlert = computed(() => reminders.state.value === 'error' || reminders.state.value === 'invalid')

function toggle(on: boolean) {
  if (on) void reminders.enable(time.value)
  else void reminders.disable()
}
function saveTime() {
  void reminders.saveTime(time.value)
}

/** Design §4.2. */
const syncLine = computed(() => {
  if (settings.syncing) return 'Đang đồng bộ…'
  if (settings.syncError === 'reauth_required') return 'Google cần bạn cho phép lại để ghi lịch và nhiệm vụ.'
  if (settings.syncError === 'google_unavailable') return 'Google chưa phản hồi. Thử lại sau ít phút.'
  if (settings.syncError === 'other') return 'Không đồng bộ được. Thử lại.'
  if (settings.lastSync) {
    const at = new Date(settings.lastSync.at)
    return `Đã đồng bộ · ${settings.lastSync.tasksCreatedCount} nhiệm vụ · hôm nay ${at.getHours().toString().padStart(2, '0')}:${at.getMinutes().toString().padStart(2, '0')}`
  }
  return ''
})
const syncButton = computed(() => {
  if (settings.syncError === 'reauth_required') return { label: 'Cho phép lại với Google', variant: 'primary' as const }
  if (settings.syncError === 'google_unavailable') return { label: 'Thử lại', variant: 'primary' as const }
  if (settings.syncError === 'other') return { label: 'Thử lại', variant: 'danger' as const }
  if (settings.lastSync) return { label: 'Đồng bộ lại', variant: 'ghost' as const }
  return { label: 'Đồng bộ với Google', variant: 'primary' as const }
})
function onSync() {
  if (settings.syncError === 'reauth_required') {
    // Same hand-off as /login: prompt=consent re-grants calendar.events + tasks.
    const state = randomState()
    sessionStorage.setItem(OAUTH_STATE_KEY, state)
    window.location.assign(googleAuthUrl(config.public.googleClientId, `${window.location.origin}/login`, state))
    return
  }
  void settings.syncGoogle()
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader />
    <h1 class="mt-2 font-display text-[28px] leading-8">Cài đặt</h1>

    <AppCard title="Nhắc học mỗi ngày" class="mt-4">
      <SpeechBubble :line="`Mình sẽ nhắc bạn lúc ${time}.`" />
      <label class="mt-3 block">
        <span class="sr-only">Giờ nhắc</span>
        <input
          v-model="time"
          type="time"
          class="h-12 w-full rounded-btn border border-ink/15 bg-transparent px-4 text-center font-display text-2xl tabular-nums dark:border-paper/15"
        >
      </label>
      <div v-if="showSwitch" class="mt-4 flex items-center justify-between border-t border-ink/10 pt-4 dark:border-paper/10">
        <span id="reminder-label" class="font-semibold">Bật nhắc học</span>
        <AppSwitch
          :model-value="switchOn"
          :disabled="reminders.state.value === 'denied'"
          :busy="reminders.state.value === 'requesting'"
          labelledby="reminder-label"
          @update:model-value="toggle"
        />
      </div>
      <p v-if="reminderLine" class="mt-2 text-sm" :class="reminderIsAlert ? 'text-alert' : 'text-mute'" role="status">
        {{ reminderLine }}
      </p>
      <AppButton v-if="timeChanged" class="mt-4" block :loading="settings.saving" @click="saveTime">
        Lưu giờ nhắc
      </AppButton>
    </AppCard>

    <AppCard title="Google Lịch & Nhiệm vụ" class="mt-4">
      <p class="text-mute">
        Tạo một sự kiện học 30 phút lặp mỗi ngày và một danh sách 28 nhiệm vụ.
      </p>
      <AppButton class="mt-4" block :variant="syncButton.variant" :loading="settings.syncing" @click="onSync">
        {{ syncButton.label }}
      </AppButton>
      <p v-if="syncLine" class="mt-2 text-sm" :class="settings.syncError && settings.syncError !== 'reauth_required' ? 'text-alert' : 'text-mute'" role="status">
        {{ syncLine }}
      </p>
    </AppCard>
  </main>
</template>
