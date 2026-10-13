import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore, type SignInResponse } from '~/stores/auth'
import { createApiClient, isCachedApiResponse } from '~/utils/apiClient'
import { classifyContent } from '~/utils/content'
import { initialSession, readDraft, writeDraft } from '~/utils/learningSession'
import { API_STATE_CACHE } from '~/utils/session'
import { installSeededCaches } from './fakeCaches'

const session: SignInResponse = { access_token: 'original-session-token', token_type: 'Bearer', expires_in: 86400, user: { id: 'lifecycle-owner', email: 'owner@test.example', full_name: 'Owner', cefr_current: 'B1' } }
const scope = { owner: session.user.id, task: 'reading', revision: 'rev' }
const content = classifyContent({ questions: [{ id: 'q', prompt: 'Choose a word.', options: { A: 'One', B: 'Two' } }] })
function deferred<T>() { let resolve!: (v: T) => void; const promise = new Promise<T>(r => { resolve = r }); return { promise, resolve } }
function response(status: number, body: unknown = {}) { return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } }) }
async function setup() {
  const auth = useAuthStore(); await auth.signIn(session)
  const transport = deferred<Response>()
  const fetchMock = vi.fn<typeof fetch>(() => transport.promise)
  const onUnauthorized = vi.fn(() => { void auth.signOut() })
  const api = createApiClient({ baseURL: 'http://api.test', getToken: () => auth.accessToken, getOwner: () => auth.user?.id ?? null, getSessionEpoch: () => auth.sessionEpoch, onUnauthorized, fetch: fetchMock })
  return { auth, transport, api, onUnauthorized, fetchMock }
}
async function replace(auth: ReturnType<typeof useAuthStore>) {
  await auth.signOut(); await auth.signIn({ ...session, access_token: 'replacement-session-token' })
  writeDraft(scope, { ...initialSession(content), pending: 'B' })
}
beforeEach(() => { localStorage.clear(); setActivePinia(createPinia()) })
afterEach(() => vi.unstubAllGlobals())

describe('API session lifetime with actual auth and Pinia', () => {
  it('checks the caller lifetime after settled fetch when same-token relogin is queued before 401 cleanup', async () => {
    const { auth, transport, api, onUnauthorized } = await setup()
    const pending = api.get('/api/v1/pet/status'); transport.resolve(response(401))
    queueMicrotask(() => { void auth.signOut(); void auth.signIn(session); writeDraft(scope, { ...initialSession(content), pending: 'B' }) })
    await expect(pending).rejects.toMatchObject({ code: 'session_changed' })
    expect(auth.sessionEpoch).toBe(3); expect(auth.isAuthenticated).toBe(true); expect(onUnauthorized).not.toHaveBeenCalled(); expect(readDraft(scope, content)?.pending).toBe('B')
  })

  it.each([200, 503])('checks the caller lifetime after settled %s body when replacement is queued between helper and caller', async status => {
    const { auth, transport, api, onUnauthorized } = await setup()
    const body = deferred<unknown>(); const started = deferred<boolean>(); const res = response(status)
    vi.spyOn(res, 'json').mockImplementation(() => { started.resolve(true); return body.promise })
    const pending = api.get('/api/v1/pet/status'); transport.resolve(res); await started.promise
    const boundary = () => { void auth.signOut(); void auth.signIn({ ...session, access_token: 'replacement-session-token' }); writeDraft(scope, { ...initialSession(content), pending: 'B' }) }
    body.resolve(status === 200 ? { plant_name: 'OLD SESSION' } : { error: 'old_session_error' })
    if (status === 200) queueMicrotask(boundary)
    else queueMicrotask(() => queueMicrotask(boundary)) // errorCode adds its own body await
    await expect(pending).rejects.toMatchObject({ code: 'session_changed' })
    expect(auth.sessionEpoch).toBe(3); expect(auth.isAuthenticated).toBe(true); expect(onUnauthorized).not.toHaveBeenCalled(); expect(readDraft(scope, content)?.pending).toBe('B')
  })

  it.each([401, 200])('rejects old %s after same-owner logout/relogin without clearing the replacement draft/session', async status => {
    const { auth, transport, api, onUnauthorized } = await setup()
    const epoch = auth.sessionEpoch; const pending = api.get('/api/v1/pet/status')
    await replace(auth); expect(auth.sessionEpoch).toBe(epoch + 2)
    transport.resolve(response(status, { plant_name: 'Old session pet' }))
    await expect(pending).rejects.toMatchObject({ code: 'session_changed' })
    expect(onUnauthorized).not.toHaveBeenCalled(); expect(auth.isAuthenticated).toBe(true); expect(readDraft(scope, content)?.pending).toBe('B')
  })

  it.each([200, 503])('rejects a session boundary during awaited %s body parsing', async status => {
    const { auth, transport, api, onUnauthorized } = await setup()
    const body = deferred<unknown>(); const started = deferred<boolean>()
    const res = response(status); vi.spyOn(res, 'json').mockImplementation(() => { started.resolve(true); return body.promise })
    const pending = api.get('/api/v1/pet/status'); transport.resolve(res); await started.promise
    await replace(auth); body.resolve(status === 200 ? { plant_name: 'Old pet' } : { error: 'old_session_error' })
    await expect(pending).rejects.toMatchObject({ code: 'session_changed' })
    expect(onUnauthorized).not.toHaveBeenCalled(); expect(auth.isAuthenticated).toBe(true); expect(readDraft(scope, content)?.pending).toBe('B')
  })

  it('ignores a previous-token 401 after same-owner renewal while keeping the new token and draft', async () => {
    const { auth, transport, api, onUnauthorized } = await setup()
    writeDraft(scope, { ...initialSession(content), pending: 'A' })
    const epoch = auth.sessionEpoch; const pending = api.get('/api/v1/pet/status')
    await auth.signIn({ ...session, access_token: 'renewed-session-token' }); expect(auth.sessionEpoch).toBe(epoch)
    transport.resolve(response(401))
    await expect(pending).rejects.toMatchObject({ code: 'session_changed' })
    expect(onUnauthorized).not.toHaveBeenCalled(); expect(auth.isAuthenticated).toBe(true); expect(auth.accessToken).toBe('renewed-session-token'); expect(readDraft(scope, content)?.pending).toBe('A')
  })

  it('accepts same-session success after token renewal with stable cache identity, provenance and draft continuity', async () => {
    const { auth, transport, api, onUnauthorized, fetchMock } = await setup()
    writeDraft(scope, { ...initialSession(content), pending: 'A' })
    const cache = await installSeededCaches(API_STATE_CACHE)
    const epoch = auth.sessionEpoch; const pending = api.get('/api/v1/quests/daily')
    await auth.signIn({ ...session, access_token: 'renewed-session-token' }); expect(auth.sessionEpoch).toBe(epoch)
    const res = response(200, { tasks: [] }); res.headers.set('X-Aelp-Cached', 'true'); transport.resolve(res)
    const accepted = await pending as object
    expect(accepted).toEqual({ tasks: [] }); expect(isCachedApiResponse(accepted)).toBe(true)
    expect(fetchMock.mock.calls[0]?.[0]).toBe(`http://api.test/api/v1/quests/daily?aelp_owner=${session.user.id}`)
    expect(await cache.has(API_STATE_CACHE)).toBe(true); expect(readDraft(scope, content)?.pending).toBe('A'); expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it('still clears the current session and its draft when its current-token request returns 401', async () => {
    const { auth, transport, api, onUnauthorized } = await setup()
    writeDraft(scope, { ...initialSession(content), pending: 'A' })
    const pending = api.get('/api/v1/pet/status'); transport.resolve(response(401))
    await expect(pending).rejects.toMatchObject({ code: 'unauthorized' })
    expect(onUnauthorized).toHaveBeenCalledTimes(1); expect(auth.isAuthenticated).toBe(false); expect(readDraft(scope, content)).toBeNull()
  })
})
