import { useAuthStore } from '~/stores/auth'

// Static hosts may canonicalise a route to its trailing-slash form — Cloudflare
// Pages answered `/login?code=…` with `308 /login/?code=…` (2026-09-25) — and
// `/login/` must still be the login page, or a returning Google user lands in
// the protected branch below and loses the code.
function normalisePath(path: string): string {
  return path.replace(/\/+$/, '') || '/'
}

export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  if (!auth.hydrated) auth.hydrate()

  if (normalisePath(to.path) === '/login') {
    // A signed-in user has no business on /login unless Google just sent them back.
    if (auth.isAuthenticated && !to.query.code) return navigateTo('/', { replace: true })
    // A session that is present but expired is dropped here too — with its
    // api-state cache — so the next account never inherits it (fix 2026-09-24).
    if (auth.accessToken !== null && !auth.isAuthenticated) void auth.signOut()
    return
  }
  if (!auth.isAuthenticated) {
    // A stored-but-expired token gets one sentence on /login explaining why;
    // no token at all (first visit, cleared storage) gets the plain screen.
    const expired = auth.accessToken !== null
    void auth.signOut()
    if (expired) return navigateTo({ path: '/login', query: { reason: 'expired' } }, { replace: true })
    return navigateTo('/login', { replace: true })
  }
})
