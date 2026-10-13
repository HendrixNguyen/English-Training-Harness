import { describe, expect, it, vi } from 'vitest'
import { ApiError, createApiClient } from '~/utils/apiClient'

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function setup(response: Response, token: string | null = 'jwt-1') {
  const fetchMock = vi.fn(async () => response)
  const onUnauthorized = vi.fn()
  const api = createApiClient({
    baseURL: 'http://api.test',
    getToken: () => token,
    onUnauthorized,
    fetch: fetchMock as unknown as typeof fetch,
  })
  return { api, fetchMock, onUnauthorized }
}

describe('createApiClient', () => {
  it('binds cached private reads to a stable owner across token renewal without bearer material in the URL', async () => {
    let token = 'private-token-one'; let owner = 'owner-a'
    const fetchMock = vi.fn<typeof fetch>(async () => jsonResponse(200, {}))
    const api = createApiClient({ baseURL: 'http://api.test', getToken: () => token, getOwner: () => owner, onUnauthorized: vi.fn(), fetch: fetchMock as typeof fetch })
    await api.get('/api/v1/quests/daily'); token = 'private-token-two'; await api.get('/api/v1/quests/daily')
    expect(fetchMock.mock.calls.map(c => c[0])).toEqual(['http://api.test/api/v1/quests/daily?aelp_owner=owner-a', 'http://api.test/api/v1/quests/daily?aelp_owner=owner-a'])
    owner = 'owner-b'; await api.get('/api/v1/pet/status'); await api.get('/api/v1/quests/daily?fresh=true')
    expect(fetchMock.mock.calls[2]?.[0]).toBe('http://api.test/api/v1/pet/status?aelp_owner=owner-b')
    expect(fetchMock.mock.calls[3]?.[0]).toBe('http://api.test/api/v1/quests/daily?fresh=true&aelp_owner=owner-b')
    await api.post('/api/v1/quests/progress', {}); expect(fetchMock.mock.calls[4]?.[0]).toBe('http://api.test/api/v1/quests/progress')
  })
  it('rejects a late prior-owner response before adopting data or signing out the replacement account', async () => {
    let owner = 'a'; let resolve!: (r: Response) => void
    const fetchMock = vi.fn<typeof fetch>(() => new Promise(r => { resolve = r }))
    const onUnauthorized = vi.fn()
    const api = createApiClient({ baseURL: 'http://api.test', getToken: () => 'token', getOwner: () => owner, onUnauthorized, fetch: fetchMock })
    const pending = api.get('/api/v1/pet/status'); owner = 'b'; resolve(jsonResponse(401, {}))
    await expect(pending).rejects.toMatchObject({ code: 'session_changed' }); expect(onUnauthorized).not.toHaveBeenCalled()
  })
  it('prefixes the base URL and sends Authorization: Bearer <jwt>', async () => {
    const { api, fetchMock } = setup(jsonResponse(200, { ok: true }))
    await expect(api.get('/api/v1/quests/daily')).resolves.toEqual({ ok: true })
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('http://api.test/api/v1/quests/daily')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer jwt-1')
    expect(init.method).toBe('GET')
  })

  it('omits the Authorization header when there is no token', async () => {
    const { api, fetchMock } = setup(jsonResponse(200, {}), null)
    await api.post('/api/v1/auth/google', { code: 'c', redirect_uri: 'r' })
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    const headers = init.headers as Record<string, string>
    expect(headers.Authorization).toBeUndefined()
    expect(headers['Content-Type']).toBe('application/json')
    expect(init.body).toBe(JSON.stringify({ code: 'c', redirect_uri: 'r' }))
  })

  it('calls onUnauthorized and throws ApiError(401) on a 401', async () => {
    const { api, onUnauthorized } = setup(jsonResponse(401, { error: 'unauthorized' }))
    await expect(api.get('/api/v1/pet/status')).rejects.toMatchObject({ status: 401, code: 'unauthorized' })
    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  it('maps the {error} envelope to ApiError.code', async () => {
    const { api, onUnauthorized } = setup(jsonResponse(404, { error: 'no_active_roadmap' }))
    const err = await api.get('/api/v1/quests/daily').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 404, code: 'no_active_roadmap' })
    expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it('falls back to unknown_error when the error body is not JSON', async () => {
    const { api } = setup(new Response('<html>502</html>', { status: 502 }))
    await expect(api.get('/healthz')).rejects.toMatchObject({ status: 502, code: 'unknown_error' })
  })
})
