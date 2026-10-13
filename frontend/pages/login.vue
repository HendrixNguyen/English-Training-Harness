<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useApi } from '~/composables/useApi'
import { useAuthStore, type SignInResponse } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
import { googleAuthUrl, randomState } from '~/utils/googleAuth'

const STATE_KEY = 'aelp.oauth_state'

const config = useRuntimeConfig()
const route = useRoute()
const auth = useAuthStore()

const busy = ref(false)
const error = ref<string | null>(null)
const redirectUri = computed(() => `${window.location.origin}/login`)

function startSignIn() {
  if (busy.value) return
  error.value = null
  busy.value = true
  const state = randomState()
  sessionStorage.setItem(STATE_KEY, state)
  window.location.assign(googleAuthUrl(config.public.googleClientId, redirectUri.value, state))
}

async function finishSignIn(code: string, state: string) {
  const expected = sessionStorage.getItem(STATE_KEY)
  sessionStorage.removeItem(STATE_KEY)
  if (!expected || expected !== state) {
    error.value = 'Phiên đăng nhập không hợp lệ. Thử lại.'
    return
  }
  busy.value = true
  try {
    const res = await useApi().post<SignInResponse>('/api/v1/auth/google', { code, redirect_uri: redirectUri.value })
    await auth.signIn(res)
    if (!auth.isAuthenticated) {
      // The backend answered 200 but not in the §6.1 shape (see the plan's merge blocker).
      error.value = 'Máy chủ trả về phiên đăng nhập không hợp lệ. Thử lại sau.'
      return
    }
    await navigateTo('/', { replace: true })
  } catch (e) {
    error.value = e instanceof ApiError && e.code === 'google_auth_failed'
      ? 'Google không xác nhận được tài khoản. Thử lại.'
      : 'Không đăng nhập được. Thử lại.'
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  const { code, state } = route.query
  if (typeof code === 'string' && typeof state === 'string') {
    window.history.replaceState(null, '', '/login') // never keep the code in the address bar
    void finishSignIn(code, state)
  }
})
</script>

<template>
  <main class="retro-screen mx-auto max-w-md px-4 flex min-h-screen flex-col items-center justify-center gap-6 text-center">
    <CompanionSprite stage="seed" :health="100" :size="128" />
    <h1 class="font-display text-[34px] leading-9">Học 30 phút</h1>
    <p class="font-body text-[17px] text-ink-1">Mỗi ngày một phòng. Một người bạn.</p>
    <p v-if="busy" role="status" class="text-sm text-ink-1">Đang đăng nhập…</p>
    <RetroPanel v-if="error" :announce="false" tone="ember" class="w-full text-left">
      <p role="alert" class="font-body text-[17px]">{{ error }}</p>
    </RetroPanel>
    <p class="font-body text-sm text-ink-2">Bằng cách tiếp tục, cậu cho phép ứng dụng đọc lịch và nhiệm vụ Google để lên lịch học.</p>
    <div class="retro-bottom-bar">
      <RetroButton block :loading="busy" @click="startSignIn">Đăng nhập bằng Google</RetroButton>
    </div>
  </main>
</template>
