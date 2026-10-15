import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import { useAuthStore } from '~/stores/auth'

const post = vi.fn()
vi.mock('~/composables/useApi', () => ({ useApi: () => ({ post }) }))
const navigateTo = vi.fn()
let query: Record<string, string> = {}
vi.stubGlobal('useRoute', () => ({ query }))
vi.stubGlobal('useRuntimeConfig', () => ({ public: { googleClientId: 'test-client' } }))
vi.stubGlobal('navigateTo', navigateTo)
const { default: LoginPage } = await import('~/pages/login.vue')
function mountPage() { return mount(LoginPage, { global: { components: { RetroPanel, RetroButton, CompanionSprite } } }) }

describe('retro login OAuth callback', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    sessionStorage.clear()
    post.mockReset()
    navigateTo.mockReset()
    query = {}
  })
  it('refuses a mismatched state without exchanging the authorization code', async () => {
    sessionStorage.setItem('aelp.oauth_state', 'expected')
    query = { code: 'private-code', state: 'wrong' }
    const history = vi.spyOn(window.history, 'replaceState')
    const w = mountPage(); await flushPromises()
    expect(post).not.toHaveBeenCalled()
    expect(w.get('[role="alert"]').text()).toContain('Phiên đăng nhập không hợp lệ')
    expect(sessionStorage.getItem('aelp.oauth_state')).toBeNull()
    expect(history).toHaveBeenCalledWith(null, '', '/login')
  })
  it('exchanges the matching code once, shows waiting and navigates only after sign-in', async () => {
    let resolve!: (value: unknown) => void
    post.mockImplementation(() => new Promise(r => { resolve = r }))
    sessionStorage.setItem('aelp.oauth_state', 'expected')
    query = { code: 'private-code', state: 'expected' }
    const auth = useAuthStore()
    const signIn = vi.spyOn(auth, 'signIn').mockImplementation(async () => { auth.accessToken = 'session'; auth.expiresAt = Date.now() + 60000 })
    const w = mountPage(); await flushPromises()
    expect(w.get('[role="status"]').text()).toContain('Đang đăng nhập')
    expect(w.get('button').attributes('disabled')).toBeDefined()
    expect(navigateTo).not.toHaveBeenCalled()
    resolve({ access_token: 'session' }); await flushPromises()
    expect(signIn).toHaveBeenCalledOnce()
    expect(post).toHaveBeenCalledWith('/api/v1/auth/google', { code: 'private-code', redirect_uri: `${window.location.origin}/login` })
    expect(navigateTo).toHaveBeenCalledWith('/', { replace: true })
    const second = mountPage(); await flushPromises()
    expect(post).toHaveBeenCalledOnce()
    second.unmount(); w.unmount()
  })
  it('offers the Google button again when exchange fails', async () => {
    post.mockRejectedValue(new Error('network'))
    sessionStorage.setItem('aelp.oauth_state', 'expected')
    query = { code: 'private-code', state: 'expected' }
    const w = mountPage(); await flushPromises()
    expect(w.get('[role="alert"]').text()).toContain('Không đăng nhập được')
    expect(w.get('button').attributes('disabled')).toBeUndefined()
    expect(navigateTo).not.toHaveBeenCalled()
  })
})
