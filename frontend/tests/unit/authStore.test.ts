import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AUTH_STORAGE_KEY, useAuthStore, type SignInResponse } from '~/stores/auth'
import { API_STATE_CACHE } from '~/utils/session'
import { installSeededCaches } from './fakeCaches'

const spec61: SignInResponse = {
  access_token: 'eyJ.test',
  token_type: 'Bearer',
  expires_in: 86400,
  user: { id: 'u1', email: 'user@example.com', full_name: 'Nguyen Hendrix', cefr_current: 'B1' },
}

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('signIn stores the §6.1 access_token and computes expiresAt from expires_in', () => {
    const auth = useAuthStore()
    auth.signIn(spec61, 1_000_000)
    expect(auth.accessToken).toBe('eyJ.test')
    expect(auth.expiresAt).toBe(1_000_000 + 86400 * 1000)
    expect(auth.user?.full_name).toBe('Nguyen Hendrix')
    expect(JSON.parse(localStorage.getItem(AUTH_STORAGE_KEY) ?? '{}')).toEqual({
      accessToken: 'eyJ.test',
      expiresAt: 1_000_000 + 86400 * 1000,
      user: spec61.user,
    })
  })

  it('does not accept the pre-spec {token} shape', () => {
    const auth = useAuthStore()
    // The merged handler's shape; the frontend targets the spec, not the handler.
    auth.signIn({ token: 'legacy', user: spec61.user } as unknown as SignInResponse, 0)
    expect(auth.accessToken).toBeNull()
    expect(auth.isAuthenticated).toBe(false)
  })

  it('hydrate restores a persisted session and isAuthenticated respects expiry', () => {
    localStorage.setItem(
      AUTH_STORAGE_KEY,
      JSON.stringify({ accessToken: 't', expiresAt: Date.now() + 60_000, user: spec61.user }),
    )
    const auth = useAuthStore()
    auth.hydrate()
    expect(auth.isAuthenticated).toBe(true)

    localStorage.setItem(
      AUTH_STORAGE_KEY,
      JSON.stringify({ accessToken: 't', expiresAt: Date.now() - 1, user: spec61.user }),
    )
    setActivePinia(createPinia())
    const expired = useAuthStore()
    expired.hydrate()
    expect(expired.isAuthenticated).toBe(false)
  })

  it('hydrate tolerates garbage in storage', () => {
    localStorage.setItem(AUTH_STORAGE_KEY, '{not json')
    const auth = useAuthStore()
    auth.hydrate()
    expect(auth.hydrated).toBe(true)
    expect(auth.isAuthenticated).toBe(false)
  })

  it('signOut clears state and storage (what the 401 hook calls)', () => {
    const auth = useAuthStore()
    auth.signIn(spec61, Date.now())
    auth.signOut()
    expect(auth.accessToken).toBeNull()
    expect(auth.user).toBeNull()
    expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull()
  })

  it('signOut deletes the service worker api-state cache and nothing else', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const auth = useAuthStore()
    auth.signIn(spec61, Date.now())

    await auth.signOut()

    expect(await caches.has(API_STATE_CACHE)).toBe(false)
    expect(await caches.keys()).toEqual(['assets'])
    expect(auth.isAuthenticated).toBe(false)
  })

  it('signOut still resolves where CacheStorage does not exist', async () => {
    vi.stubGlobal('caches', undefined)
    const auth = useAuthStore()
    auth.signIn(spec61, Date.now())
    await expect(auth.signOut()).resolves.toBeUndefined()
    expect(auth.accessToken).toBeNull()
  })

  it('signIn drops the previous account\'s api-state cache and keeps assets', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const auth = useAuthStore()

    await auth.signIn(spec61, Date.now())

    expect(await caches.has(API_STATE_CACHE)).toBe(false)
    expect(await caches.keys()).toEqual(['assets'])
    expect(auth.isAuthenticated).toBe(true)
  })
})
