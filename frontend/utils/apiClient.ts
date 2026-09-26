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
  /** Called once per 401 before the ApiError is thrown (sign out + redirect). */
  onUnauthorized: () => void
  /**
   * Called when a response carries a fresh X-Session-Token (Require's silent
   * renewal, design harness/designs/stay-signed-in.md §5). Only on a 2xx, and
   * only when the token actually changed — never on an error response or an
   * unchanged token.
   */
  onRenew?: (token: string, expiresIn: number) => void
  fetch?: typeof globalThis.fetch
}

export interface ApiClient {
  get<T>(path: string): Promise<T>
  post<T>(path: string, body?: unknown): Promise<T>
}

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

  async function request<T>(method: 'GET' | 'POST', path: string, body?: unknown, isRetry = false): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' }
    const sent = opts.getToken()
    if (sent) headers.Authorization = `Bearer ${sent}`
    if (body !== undefined) headers['Content-Type'] = 'application/json'

    const res = await doFetch(`${opts.baseURL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })

    if (res.ok) {
      const renewed = res.headers.get('X-Session-Token')
      if (renewed && renewed !== opts.getToken()) {
        opts.onRenew?.(renewed, Number(res.headers.get('X-Session-Expires-In')))
      }
    }

    if (res.status === 401) {
      // A renewal from another in-flight request can land between this
      // request being sent and its 401 coming back; retry once with the
      // newest token before giving up (design §5 "One retry").
      if (!isRetry && opts.getToken() !== sent) {
        return request<T>(method, path, body, true)
      }
      opts.onUnauthorized()
      throw new ApiError(401, 'unauthorized')
    }
    if (!res.ok) throw new ApiError(res.status, await errorCode(res))
    return (await res.json()) as T
  }

  return {
    get: <T>(path: string) => request<T>('GET', path),
    post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  }
}
