<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '~/stores/auth'

defineProps<{ streak?: number | null }>()

const auth = useAuthStore()
const menuOpen = ref(false)

async function signOut() {
  await auth.signOut()
  await navigateTo('/login', { replace: true })
}
</script>

<template>
  <header class="flex items-center justify-between py-3">
    <div class="relative flex items-center gap-3">
      <button
        type="button"
        class="flex size-10 items-center justify-center rounded-full bg-growth font-semibold text-white"
        aria-haspopup="menu"
        :aria-expanded="menuOpen"
        aria-label="Tài khoản"
        @click="menuOpen = !menuOpen"
      >
        {{ auth.initial }}
      </button>
      <span class="font-semibold">{{ auth.user?.full_name }}</span>
      <div v-if="menuOpen" role="menu" class="absolute left-0 top-12 z-10 w-44 rounded-card border border-ink/10 bg-white p-1 shadow-sm dark:border-paper/10 dark:bg-ink">
        <NuxtLink to="/settings" role="menuitem" class="block rounded-btn px-3 py-2 hover:bg-ink/5 dark:hover:bg-paper/10" @click="menuOpen = false">
          Cài đặt
        </NuxtLink>
        <button type="button" role="menuitem" class="block w-full rounded-btn px-3 py-2 text-left hover:bg-ink/5 dark:hover:bg-paper/10" @click="signOut">
          Đăng xuất
        </button>
      </div>
    </div>
    <div class="flex items-center gap-3">
      <NuxtLink to="/roadmap" class="text-sm text-mute underline-offset-2 hover:underline">
        Lộ trình
      </NuxtLink>
      <span v-if="streak !== null && streak !== undefined" class="rounded-full bg-streak/15 px-3 py-1 text-sm font-semibold text-streak">
        🔥 Streak: {{ streak }} ngày
      </span>
    </div>
  </header>
</template>
