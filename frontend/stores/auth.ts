import { defineStore } from 'pinia'
import { clearApiCache } from '~/utils/session'

/** Backend spec §6.1 `user`. */
export interface AuthUser {
  id: string
  email: string
  full_name: string
  cefr_current: string | null
}

/** Backend spec §6.1 `POST /api/v1/auth/google` 200 body — the contract, not the merged handler. */
export interface SignInResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: AuthUser
}

interface Persisted {
  accessToken: string
  expiresAt: number
  user: AuthUser
}

export const AUTH_STORAGE_KEY = 'aelp.auth'

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    accessToken: null as string | null,
    expiresAt: null as number | null,
    user: null as AuthUser | null,
    hydrated: false,
  }),
  getters: {
    isAuthenticated: s => s.accessToken !== null && s.expiresAt !== null && s.expiresAt > Date.now(),
    initial: s => (s.user?.full_name?.trim().charAt(0) || '?').toUpperCase(),
  },
  actions: {
    hydrate() {
      this.hydrated = true
      const raw = storageOrNull()?.getItem(AUTH_STORAGE_KEY)
      if (!raw) return
      try {
        const p = JSON.parse(raw) as Partial<Persisted>
        if (typeof p.accessToken === 'string' && typeof p.expiresAt === 'number' && p.user) {
          this.accessToken = p.accessToken
          this.expiresAt = p.expiresAt
          this.user = p.user
        }
      } catch {
        storageOrNull()?.removeItem(AUTH_STORAGE_KEY)
      }
    },
    /**
     * Stores the new session, then drops any previous account's cached API
     * responses — the mirror of signOut(). State and storage are written
     * synchronously; the returned promise is the cache clear, and /login
     * awaits it before navigating so the hub never reads a stale entry.
     */
    signIn(res: SignInResponse, now: number = Date.now()): Promise<void> {
      if (typeof res.access_token !== 'string' || typeof res.expires_in !== 'number') return Promise.resolve()
      this.accessToken = res.access_token
      this.expiresAt = now + res.expires_in * 1000
      this.user = res.user
      this.hydrated = true
      const p: Persisted = { accessToken: this.accessToken, expiresAt: this.expiresAt, user: res.user }
      storageOrNull()?.setItem(AUTH_STORAGE_KEY, JSON.stringify(p))
      return clearApiCache()
    },
    /**
     * Adopts a token Require renewed silently (design harness/designs/stay-signed-in.md
     * §5): updates the session in place and rewrites storage, but never
     * touches the api-state cache — a renewal must be invisible, not a
     * fresh sign-in. Ignored when there is no signed-in user or expiresIn
     * is not a positive, finite number.
     */
    renew(token: string, expiresIn: number, now: number = Date.now()): void {
      if (!Number.isFinite(expiresIn) || expiresIn <= 0) return
      if (this.user === null) return
      this.accessToken = token
      this.expiresAt = now + expiresIn * 1000
      const p: Persisted = { accessToken: this.accessToken, expiresAt: this.expiresAt, user: this.user }
      storageOrNull()?.setItem(AUTH_STORAGE_KEY, JSON.stringify(p))
    },
    /** Drops the session, then the per-user service worker cache — every sign-out path goes through here. */
    signOut(): Promise<void> {
      this.accessToken = null
      this.expiresAt = null
      this.user = null
      storageOrNull()?.removeItem(AUTH_STORAGE_KEY)
      return clearApiCache()
    },
  },
})
