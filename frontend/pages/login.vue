<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useApi } from '~/composables/useApi'
import { useAuthStore, type SignInResponse } from '~/stores/auth'
import { ApiError } from '~/utils/apiClient'
import { googleAuthUrl, randomState } from '~/utils/googleAuth'
import { loginNotice } from '~/utils/loginReason'

const STATE_KEY = 'aelp.oauth_state'

const config = useRuntimeConfig()
const route = useRoute()
const auth = useAuthStore()

const busy = ref(false)
const error = ref<string | null>(null)
const redirectUri = computed(() => `${window.location.origin}/login`)
const notice = computed(() => loginNotice(route.query.reason))

function startSignIn() {
  error.value = null
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
  <main class="mx-auto flex min-h-screen max-w-md flex-col items-center justify-center gap-6 px-6 text-center">
    <PlantSvg stage="sprout" :health="100" :size="96" />
    <h1 class="font-display text-3xl">
      Chào mừng bạn! 🌱
    </h1>
    <p class="text-mute">
      Học 30 phút mỗi ngày, nuôi một cái cây.
    </p>

    <AppCard v-if="notice" data-testid="login-expired" class="text-left text-sm text-ink">
      <span aria-hidden="true">⏳</span> {{ notice }}
    </AppCard>

    <div v-if="busy" class="flex items-center gap-2 text-mute" role="status">
      <span class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
      Đang đăng nhập…
    </div>
    <AppButton v-else block @click="startSignIn">
      Đăng nhập bằng Google
    </AppButton>

    <p v-if="error" class="w-full rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
      {{ error }}
    </p>

    <p class="text-xs text-mute">
      Bằng cách tiếp tục, bạn cho phép ứng dụng đọc lịch và nhiệm vụ Google của bạn để lên lịch học.
    </p>
  </main>
</template>
