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
