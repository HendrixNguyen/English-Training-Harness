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
    signIn(res: SignInResponse, now: number = Date.now()) {
      if (typeof res.access_token !== 'string' || typeof res.expires_in !== 'number') return
      this.accessToken = res.access_token
      this.expiresAt = now + res.expires_in * 1000
      this.user = res.user
      this.hydrated = true
      const p: Persisted = { accessToken: this.accessToken, expiresAt: this.expiresAt, user: res.user }
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
