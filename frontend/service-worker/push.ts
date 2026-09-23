export interface PushPayload {
  title: string
  body: string
  url: string
}

const DEFAULT: PushPayload = { title: 'Học 30 phút', body: 'Cây của bạn đang chờ bạn.', url: '/' }

/** Notify slice payload → notification. Unknown or cross-origin urls open the dashboard. */
export function parsePushPayload(text: string | null): PushPayload {
  if (!text) return { ...DEFAULT }
  try {
    const p = JSON.parse(text) as Partial<PushPayload>
    const url = typeof p.url === 'string' && p.url.startsWith('/') ? p.url : '/'
    return {
      title: typeof p.title === 'string' && p.title ? p.title : DEFAULT.title,
      body: typeof p.body === 'string' && p.body ? p.body : DEFAULT.body,
      url,
    }
  } catch {
    return { ...DEFAULT, body: text }
  }
}
