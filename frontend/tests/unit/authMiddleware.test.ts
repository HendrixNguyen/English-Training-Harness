import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { RouteLocationNormalized } from 'vue-router'
import { AUTH_STORAGE_KEY } from '~/stores/auth'
import { API_STATE_CACHE } from '~/utils/session'
import { installSeededCaches } from './fakeCaches'

const navigateTo = vi.fn()
vi.stubGlobal('defineNuxtRouteMiddleware', <T>(fn: T) => fn)
vi.stubGlobal('navigateTo', navigateTo)

const { default: guard } = await import('~/middleware/auth.global')

const user = { id: 'u1', email: 'user@example.com', full_name: 'Nguyen Hendrix', cefr_current: 'B1' }
const to = { path: '/', query: {} } as RouteLocationNormalized

function persistSession(expiresAt: number) {
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify({ accessToken: 't', expiresAt, user }))
}

describe('middleware/auth.global — expired session (the daily sign-out path)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    navigateTo.mockReset()
  })

  it('drops the per-user api-state cache, keeps the assets cache, and redirects to /login', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    persistSession(Date.now() - 1) // expires_in elapsed since the last visit

    guard(to, to)

    await vi.waitFor(async () => expect(await caches.has(API_STATE_CACHE)).toBe(false))
    expect(await caches.has('assets')).toBe(true)
    expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull()
    expect(navigateTo).toHaveBeenCalledWith('/login', { replace: true })
  })

  it('leaves the cache alone while the session is valid', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    persistSession(Date.now() + 60_000)

    guard(to, to)
    await Promise.resolve()

    expect(await caches.has(API_STATE_CACHE)).toBe(true)
    expect(navigateTo).not.toHaveBeenCalled()
  })

  it('drops an expired session and its cache when /login is the first route, without redirecting', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    persistSession(Date.now() - 1) // a bookmark straight to /login on a shared device
    const login = { path: '/login', query: {} } as RouteLocationNormalized

    guard(login, login)

    await vi.waitFor(async () => expect(await caches.has(API_STATE_CACHE)).toBe(false))
    expect(await caches.has('assets')).toBe(true)
    expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull()
    expect(navigateTo).not.toHaveBeenCalled()
  })

  it('leaves /login alone when there is no session at all', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const login = { path: '/login', query: {} } as RouteLocationNormalized

    guard(login, login)
    await Promise.resolve()

    expect(await caches.has(API_STATE_CACHE)).toBe(true) // nothing to drop; signIn() will
    expect(navigateTo).not.toHaveBeenCalled()
  })
})

describe('middleware/auth.global — trailing slash from a static host (Pages 308 → /login/, fix 2026-09-25)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    navigateTo.mockReset()
  })

  it('treats /login/?code=…&state=… like /login: no redirect, no sign-out, so the Google code survives', async () => {
    const caches = await installSeededCaches(API_STATE_CACHE)
    const login = { path: '/login/', query: { code: 'c', state: 's' } } as unknown as RouteLocationNormalized

    guard(login, login)
    await Promise.resolve()

    expect(navigateTo).not.toHaveBeenCalled()
    expect(await caches.has(API_STATE_CACHE)).toBe(true) // signOut() was not called
  })

  it('sends a signed-in user on /login/ home, exactly as on /login', () => {
    persistSession(Date.now() + 60_000)
    const login = { path: '/login/', query: {} } as RouteLocationNormalized

    guard(login, login)

    expect(navigateTo).toHaveBeenCalledWith('/', { replace: true })
  })

  it('still guards a protected route written with a trailing slash', () => {
    const roadmap = { path: '/roadmap/', query: {} } as RouteLocationNormalized

    guard(roadmap, roadmap)

    expect(navigateTo).toHaveBeenCalledWith('/login', { replace: true })
  })
})
