import { useAuthStore } from '~/stores/auth'

export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  if (!auth.hydrated) auth.hydrate()

  if (to.path === '/login') {
    // A signed-in user has no business on /login unless Google just sent them back.
    if (auth.isAuthenticated && !to.query.code) return navigateTo('/', { replace: true })
    return
  }
  if (!auth.isAuthenticated) {
    auth.signOut()
    return navigateTo('/login', { replace: true })
  }
})
