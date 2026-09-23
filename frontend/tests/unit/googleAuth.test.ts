import { describe, expect, it } from 'vitest'
import { GOOGLE_SCOPES, googleAuthUrl, randomState } from '~/utils/googleAuth'

describe('googleAuthUrl (mirrors backend/internal/auth/scopes.go)', () => {
  it('builds the consent URL with the backend scope list, offline access and consent prompt', () => {
    const u = new URL(googleAuthUrl('client-1', 'http://localhost:3000/login', 'state-1'))
    expect(u.origin + u.pathname).toBe('https://accounts.google.com/o/oauth2/v2/auth')
    expect(u.searchParams.get('client_id')).toBe('client-1')
    expect(u.searchParams.get('redirect_uri')).toBe('http://localhost:3000/login')
    expect(u.searchParams.get('response_type')).toBe('code')
    expect(u.searchParams.get('access_type')).toBe('offline')
    expect(u.searchParams.get('prompt')).toBe('consent')
    expect(u.searchParams.get('state')).toBe('state-1')
    expect(u.searchParams.get('scope')).toBe(GOOGLE_SCOPES.join(' '))
  })

  it('requests calendar.events and tasks at first sign-in so the google slice never forces re-consent', () => {
    expect(GOOGLE_SCOPES).toEqual([
      'openid',
      'email',
      'profile',
      'https://www.googleapis.com/auth/calendar.events',
      'https://www.googleapis.com/auth/tasks',
    ])
  })

  it('randomState is 32 hex chars and not repeated', () => {
    const a = randomState()
    expect(a).toMatch(/^[0-9a-f]{32}$/)
    expect(randomState()).not.toBe(a)
  })
})
