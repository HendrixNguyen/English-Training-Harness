import { describe, expect, it } from 'vitest'
import { stripSessionHeaders } from '~/service-worker/apiStateCache'

describe('stripSessionHeaders (design harness/designs/stay-signed-in.md §4 "Offline")', () => {
  it('removes X-Session-Token and X-Session-Expires-In before a quests/daily response is cached', async () => {
    const response = new Response(JSON.stringify({ day_number: 3 }), {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
        'X-Session-Token': 'eyJ.renewed',
        'X-Session-Expires-In': '86400',
      },
    })

    const stripped = stripSessionHeaders(response)

    expect(stripped.headers.get('X-Session-Token')).toBeNull()
    expect(stripped.headers.get('X-Session-Expires-In')).toBeNull()
    expect(stripped.headers.get('Content-Type')).toBe('application/json')
    expect(stripped.status).toBe(200)
    await expect(stripped.json()).resolves.toEqual({ day_number: 3 })
  })

  it('passes a response with no session headers through unchanged', async () => {
    const response = new Response(JSON.stringify({ health_points: 80 }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })

    const stripped = stripSessionHeaders(response)

    expect(stripped.headers.get('X-Session-Token')).toBeNull()
    await expect(stripped.json()).resolves.toEqual({ health_points: 80 })
  })
})
