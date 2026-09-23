import { describe, expect, it } from 'vitest'
import { parsePushPayload } from '~/service-worker/push'

describe('parsePushPayload (idea: "shows the notify payload and opens its url")', () => {
  it('reads title, body and url from a JSON payload', () => {
    expect(parsePushPayload('{"title":"Đến giờ học","body":"Cây đang khát","url":"/"}')).toEqual({
      title: 'Đến giờ học',
      body: 'Cây đang khát',
      url: '/',
    })
  })

  it('falls back to a default notification for an empty or non-JSON payload', () => {
    expect(parsePushPayload(null)).toEqual({ title: 'Học 30 phút', body: 'Cây của bạn đang chờ bạn.', url: '/' })
    expect(parsePushPayload('plain text')).toEqual({ title: 'Học 30 phút', body: 'plain text', url: '/' })
  })

  it('only opens same-origin urls', () => {
    expect(parsePushPayload('{"url":"https://evil.example/x"}').url).toBe('/')
    expect(parsePushPayload('{"url":"/revive"}').url).toBe('/revive')
  })
})
