import { useAuthStore } from '~/stores/auth'

export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  if (!auth.hydrated) auth.hydrate()

  if (to.path === '/login') {
    // A signed-in user has no business on /login unless Google just sent them back.
    if (auth.isAuthenticated && !to.query.code) return navigateTo('/', { replace: true })
    // A session that is present but expired is dropped here too — with its
    // api-state cache — so the next account never inherits it (fix 2026-09-24).
    if (auth.accessToken !== null && !auth.isAuthenticated) void auth.signOut()
    return
  }
  if (!auth.isAuthenticated) {
    void auth.signOut()
    return navigateTo('/login', { replace: true })
  }
})
