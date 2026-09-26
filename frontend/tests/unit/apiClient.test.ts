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

describe('createApiClient — silent renewal (design harness/designs/stay-signed-in.md §5)', () => {
  function renewedResponse(status: number, body: unknown, extra: Record<string, string> = {}) {
    return new Response(JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json', ...extra },
    })
  }

  it('calls onRenew with X-Session-Token and X-Session-Expires-In on a 2xx', async () => {
    const fetchMock = vi.fn(async () => renewedResponse(200, { ok: true }, {
      'X-Session-Token': 'jwt-2',
      'X-Session-Expires-In': '86400',
    }))
    const onUnauthorized = vi.fn()
    const onRenew = vi.fn()
    const api = createApiClient({
      baseURL: 'http://api.test',
      getToken: () => 'jwt-1',
      onUnauthorized,
      onRenew,
      fetch: fetchMock as unknown as typeof fetch,
    })

    await api.get('/api/v1/quests/daily')

    expect(onRenew).toHaveBeenCalledTimes(1)
    expect(onRenew).toHaveBeenCalledWith('jwt-2', 86400)
  })

  it('ignores X-Session-Token on an error response and when it matches the current token', async () => {
    const onRenew = vi.fn()

    const errorFetch = vi.fn(async () => renewedResponse(404, { error: 'no_active_roadmap' }, { 'X-Session-Token': 'jwt-2' }))
    const apiOnError = createApiClient({
      baseURL: 'http://api.test',
      getToken: () => 'jwt-1',
      onUnauthorized: vi.fn(),
      onRenew,
      fetch: errorFetch as unknown as typeof fetch,
    })
    await apiOnError.get('/api/v1/quests/daily').catch(() => {})
    expect(onRenew).not.toHaveBeenCalled()

    const sameTokenFetch = vi.fn(async () => renewedResponse(200, { ok: true }, { 'X-Session-Token': 'jwt-1' }))
    const apiSameToken = createApiClient({
      baseURL: 'http://api.test',
      getToken: () => 'jwt-1',
      onUnauthorized: vi.fn(),
      onRenew,
      fetch: sameTokenFetch as unknown as typeof fetch,
    })
    await apiSameToken.get('/api/v1/quests/daily')
    expect(onRenew).not.toHaveBeenCalled()
  })

  it('retries a 401 once with the newest token when a renewal landed since the request was sent', async () => {
    let currentToken = 'jwt-1'
    let calls = 0
    const fetchMock = vi.fn(async () => {
      calls++
      if (calls === 1) {
        // A concurrent request's renewal lands while this one is in flight.
        currentToken = 'jwt-2'
        return renewedResponse(401, { error: 'unauthorized' })
      }
      return renewedResponse(200, { ok: true })
    })
    const onUnauthorized = vi.fn()
    const api = createApiClient({
      baseURL: 'http://api.test',
      getToken: () => currentToken,
      onUnauthorized,
      fetch: fetchMock as unknown as typeof fetch,
    })

    await expect(api.get('/api/v1/quests/daily')).resolves.toEqual({ ok: true })

    expect(fetchMock).toHaveBeenCalledTimes(2)
    const [, secondInit] = fetchMock.mock.calls[1] as unknown as [string, RequestInit]
    expect((secondInit.headers as Record<string, string>).Authorization).toBe('Bearer jwt-2')
    expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it('does not retry a 401 when no renewal happened', async () => {
    const fetchMock = vi.fn(async () => renewedResponse(401, { error: 'unauthorized' }))
    const onUnauthorized = vi.fn()
    const api = createApiClient({
      baseURL: 'http://api.test',
      getToken: () => 'jwt-1',
      onUnauthorized,
      fetch: fetchMock as unknown as typeof fetch,
    })

    await expect(api.get('/api/v1/quests/daily')).rejects.toMatchObject({ status: 401, code: 'unauthorized' })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })
})
