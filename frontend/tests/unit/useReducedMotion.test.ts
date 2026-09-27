import { afterEach, describe, expect, it, vi } from 'vitest'

/**
 * Design amend A4: `useReducedMotion()` is the one owner of the
 * `prefers-reduced-motion` media query. Each test re-imports the module
 * after `vi.resetModules()` so the module-level ref does not leak between
 * cases (the composable itself is a real singleton in the app).
 */
describe('useReducedMotion (design amend A4)', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.resetModules()
  })

  it('reads matchMedia(...).matches on first call', async () => {
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })))
    const { useReducedMotion } = await import('~/composables/useReducedMotion')
    expect(useReducedMotion().value).toBe(true)
  })

  it('flips when the OS setting changes', async () => {
    const listeners: ((e: { matches: boolean }) => void)[] = []
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: (_event: string, cb: (e: { matches: boolean }) => void) => listeners.push(cb),
      removeEventListener: vi.fn(),
    })))
    const { useReducedMotion } = await import('~/composables/useReducedMotion')
    const reduced = useReducedMotion()
    expect(reduced.value).toBe(false)
    for (const cb of listeners) cb({ matches: true })
    expect(reduced.value).toBe(true)
  })

  it('falls back to false when matchMedia is missing', async () => {
    vi.stubGlobal('matchMedia', undefined)
    const { useReducedMotion } = await import('~/composables/useReducedMotion')
    expect(useReducedMotion().value).toBe(false)
  })
})
