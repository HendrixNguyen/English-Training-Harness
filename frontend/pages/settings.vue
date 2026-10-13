<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useAuthStore } from '~/stores/auth'
import { useSettingsStore } from '~/stores/settings'
import { pushCapability, subscribeDevice, type PushState } from '~/utils/devicePush'
import { googleAuthUrl, randomState } from '~/utils/googleAuth'
const settings = useSettingsStore()
const auth = useAuthStore()
const config = useRuntimeConfig()
const time = ref('')
const online = ref(true)
const push = ref<PushState>('off')
const confirm = ref(false)
const signingOut = ref(false)
const owner = () => `${auth.user?.id ?? ''}:${auth.sessionEpoch}`
watch(() => settings.preferences, value => { if (value) time.value = value.notification_time.slice(0, 5) }, { immediate: true })
watch(owner, () => { time.value = ''; push.value = 'off'; confirm.value = false })
function connection() { online.value = navigator.onLine }
async function load() {
 await settings.load()
 if (settings.preferences) push.value = pushCapability(settings.preferences.push_configured, config.public.vapidPublicKey)
}
onMounted(() => { connection(); window.addEventListener('online', connection); window.addEventListener('offline', connection); void load() })
onBeforeUnmount(() => { window.removeEventListener('online', connection); window.removeEventListener('offline', connection) })
const valid = computed(() => /^([01]\d|2[0-3]):[0-5]\d$/.test(time.value))
const pushLine = computed(() => ({ unsupported: 'Thiết bị này chưa nhắc được qua trình duyệt.', 'no-key': 'Nhắc qua trình duyệt chưa được cấu hình. Cậu vẫn lưu được giờ nhắc.', denied: 'Trình duyệt đang chặn thông báo. Cho phép trong cài đặt trang web rồi thử lại.', off: 'Chưa bật nhắc trên thiết bị này.', requesting: 'Đang xin phép trình duyệt…', on: 'Đã đăng ký nhắc trên thiết bị này.', error: 'Chưa đăng ký được thông báo. Thử lại.' })[push.value])
async function enablePush() {
 if (!online.value || !settings.preferences || push.value === 'requesting') return
 const capability = pushCapability(settings.preferences.push_configured, config.public.vapidPublicKey)
 if (capability !== 'off') { push.value = capability; return }
 const ticket = owner(); const current = () => owner() === ticket
 push.value = 'requesting'
 try {
  const subscription = await subscribeDevice(config.public.vapidPublicKey, current)
  if (!current()) return
  if (!subscription) { push.value = Notification.permission === 'denied' ? 'denied' : 'off'; return }
  const saved = await settings.save(time.value, subscription)
  if (current()) push.value = saved ? 'on' : 'error'
 } catch { if (current()) push.value = 'error' }
}
function reauthorizeGoogle() {
 const state = randomState()
 sessionStorage.setItem('aelp.oauth_state', state)
 window.location.assign(googleAuthUrl(config.public.googleClientId, `${window.location.origin}/login`, state))
}
async function signOut() {
 if (signingOut.value) return
 signingOut.value = true
 try { await auth.signOut(); await navigateTo('/login') } finally { signingOut.value = false }
}
</script>
<template>
  <main class="retro-screen mx-auto max-w-md px-4 pb-8 font-body text-base">
    <NuxtLink to="/" class="flex min-h-12 items-center text-line-lit">‹ Trại</NuxtLink>
    <p class="font-display text-base text-ink-2">Quán trọ</p>
    <h1 class="mb-6 font-display text-[28px]">Cài đặt</h1>
    <p v-if="!online" role="status" class="mb-4 text-ink-1">Cần mạng để lưu cài đặt.</p>
    <RetroPanel speaker="Nhắc học" :announce="false" class="mb-8">
      <StateBlock v-if="settings.loading && !settings.preferences" state="loading" />
      <StateBlock v-else-if="!settings.preferences" state="error" message="Chưa đọc được giờ nhắc đã lưu." action="Thử lại" @action="load" />
      <form v-else class="flex flex-col gap-4" @submit.prevent="settings.save(time)">
        <label for="reminder-time" class="text-ink-1">Giờ nhắc</label>
        <input id="reminder-time" v-model="time" type="time" required class="min-h-12 w-full rounded-[2px] border-2 border-line-lit bg-ground-2 px-3 font-display text-[32px] text-ink-0 focus:outline focus:outline-2 focus:outline-torch">
        <p class="text-sm text-ink-1">Giờ đã lưu: {{ settings.preferences.notification_time.slice(0, 5) }} · {{ settings.preferences.timezone }}. Lần tới: {{ new Date(settings.preferences.next_reminder_at).toLocaleString('vi-VN', { timeZone: settings.preferences.timezone }) }}.</p>
        <p role="status" class="text-sm text-ink-1">{{ pushLine }}</p>
        <p v-if="settings.error" role="alert" class="text-sm text-ember">{{ settings.error === 'invalid_request' ? 'Giờ nhắc không hợp lệ.' : 'Chưa lưu được giờ nhắc. Thử lại.' }}</p>
        <RetroButton type="submit" block :loading="settings.saving" :disabled="!online || !valid || push === 'requesting'">Lưu giờ nhắc</RetroButton>
        <RetroButton v-if="push === 'off' || push === 'error' || push === 'requesting'" variant="secondary" :loading="push === 'requesting'" :disabled="!online || !valid || settings.saving" @click="enablePush">Bật nhắc trên thiết bị này</RetroButton>
      </form>
    </RetroPanel>
    <RetroPanel speaker="Lịch Google" :announce="false" class="mb-8">
      <p class="mb-4">Đưa 30 phút mỗi ngày vào lịch và nhiệm vụ Google của cậu.</p>
      <p v-if="settings.googleCount !== null" role="status" class="mb-4 text-sm text-growth">Đã đồng bộ · {{ settings.googleCount }} nhiệm vụ.</p>
      <p v-if="settings.googleError" role="alert" class="mb-4 text-sm text-ink-1">{{ settings.googleError === 'reauth_required' ? 'Google cần cậu cho phép lại để ghi lịch và nhiệm vụ.' : settings.googleError === 'google_unavailable' ? 'Google chưa phản hồi. Thử lại sau ít phút.' : 'Không đồng bộ được. Thử lại.' }}</p>
      <RetroButton v-if="settings.googleError === 'reauth_required'" variant="secondary" :disabled="!online" @click="reauthorizeGoogle">Cho phép lại với Google</RetroButton>
      <RetroButton v-else variant="secondary" block :loading="settings.syncing" :disabled="!online || settings.googleCount !== null" @click="settings.sync()">{{ settings.googleCount !== null ? 'Đã đồng bộ' : settings.googleError ? 'Thử lại' : 'Đồng bộ với Google' }}</RetroButton>
    </RetroPanel>
    <RetroButton variant="danger" @click="confirm = true">Rời quán · Đăng xuất</RetroButton>
    <ConfirmSheet v-if="confirm" title="Đăng xuất?" confirm-label="Đăng xuất" cancel-label="Ở lại" :loading="signingOut" @cancel="!signingOut && (confirm = false)" @confirm="signOut">Cậu sẽ cần đăng nhập lại để tiếp tục học.</ConfirmSheet>
  </main>
</template>
