import { describe, expect, it } from 'vitest'
import { tokens } from '~/tailwind.config'

describe('design tokens (Frontend spec §6.1)', () => {
  it('registers the four spec colours under their token names', () => {
    expect(tokens.growth).toBe('#10B981')
    expect(tokens.streak).toBe('#F59E0B')
    expect(tokens.alert).toBe('#EF4444')
    expect(tokens.ink).toBe('#1E293B')
  })
})
