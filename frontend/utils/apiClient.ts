/**
 * The one HTTP client. Every backend handler answers errors as
 * {"error": "<code>"}; that code becomes ApiError.code so screens can branch
 * on `no_active_roadmap`, `exercise_not_found`, `pet_not_wilted`, ... without
 * parsing bodies themselves.
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
  ) {
    super(`${status} ${code}`)
    this.name = 'ApiError'
  }
}

export interface ApiClientOptions {
  baseURL: string
  getToken: () => string | null
  /** Stable account ID used only to namespace private worker cache entries. */
  getOwner?: () => string | null
  /** Request lifetime boundary; changes on logout/relogin, stays stable on renewal. */
  getSessionEpoch?: () => number
  /** Called for a current-session, current-token 401 (sign out + redirect). */
  onUnauthorized: () => void
  fetch?: typeof globalThis.fetch
}

export interface ApiClient {
  get<T>(path: string): Promise<T>
  post<T>(path: string, body?: unknown): Promise<T>
}

const cachedResponses = new WeakSet<object>()
/** Transport provenance, separate from the server's learning DTO contract. */
export function isCachedApiResponse(value: object): boolean { return cachedResponses.has(value) }

async function errorCode(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: unknown }
    return typeof body.error === 'string' ? body.error : 'unknown_error'
  } catch {
    return 'unknown_error'
  }
}

export function createApiClient(opts: ApiClientOptions): ApiClient {
  const doFetch = opts.fetch ?? globalThis.fetch

  async function request<T>(method: 'GET' | 'POST', path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' }
    const token = opts.getToken()
    if (token) headers.Authorization = `Bearer ${token}`
    if (body !== undefined) headers['Content-Type'] = 'application/json'

    const url = new URL(`${opts.baseURL}${path}`)
    const owner = opts.getOwner?.()
    const epoch = opts.getSessionEpoch?.()
    const assertSession = () => {
      if ((opts.getOwner && owner !== opts.getOwner()) || (opts.getSessionEpoch && epoch !== opts.getSessionEpoch())) throw new ApiError(409, 'session_changed')
    }
    async function inSession<R>(run: () => Promise<R>): Promise<R> {
      try { return await run() } finally { assertSession() }
    }
    if (method === 'GET' && token && owner && /\/api\/v1\/(quests\/daily|pet\/status)$/.test(url.pathname)) url.searchParams.set('aelp_owner', owner)
    let res: Response
    try {
      res = await inSession(() => doFetch(url.toString(), {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      }))
    } catch (error) {
      assertSession()
      throw error
    }
    // The helper's guard precedes this continuation by a microtask. Check in
    // this frame before handling credentials or adopting a returned payload.
    assertSession()

    if (res.status === 401) {
      // Renewal keeps the session epoch. An old token's rejection must not
      // discard the renewed session; successful same-session reads still work.
      if (token !== opts.getToken()) throw new ApiError(409, 'session_changed')
      opts.onUnauthorized()
      throw new ApiError(401, 'unauthorized')
    }
    if (!res.ok) {
      const code = await inSession(() => errorCode(res))
      assertSession()
      throw new ApiError(res.status, code)
    }
    let value: T
    try {
      value = await inSession(() => res.json()) as T
    } catch (error) {
      assertSession()
      throw error
    }
    assertSession()
    if (res.headers.get('X-Aelp-Cached') === 'true' && typeof value === 'object' && value !== null) cachedResponses.add(value)
    return value
  }

  return {
    get: <T>(path: string) => request<T>('GET', path),
    post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  }
}
