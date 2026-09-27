import { describe, expect, it } from 'vitest'
import { tokens } from '~/tailwind.config'

/** 15-line relative-luminance WCAG contrast helper (design §6). */
function srgbToLinear(c: number): number {
  const s = c / 255
  return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}
function relativeLuminance(hex: string): number {
  const h = hex.replace('#', '')
  const r = Number.parseInt(h.slice(0, 2), 16)
  const g = Number.parseInt(h.slice(2, 4), 16)
  const b = Number.parseInt(h.slice(4, 6), 16)
  return 0.2126 * srgbToLinear(r) + 0.7152 * srgbToLinear(g) + 0.0722 * srgbToLinear(b)
}
function contrastRatio(a: string, b: string): number {
  const l1 = relativeLuminance(a)
  const l2 = relativeLuminance(b)
  const [lighter, darker] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (lighter + 0.05) / (darker + 0.05)
}

describe('design tokens v2 (design §1)', () => {
  it('registers every v2 hex under its token name', () => {
    expect(tokens['ground-0']).toBe('#0B0A1F')
    expect(tokens['ground-1']).toBe('#151434')
    expect(tokens['ground-2']).toBe('#1F1D4A')
    expect(tokens['line-lit']).toBe('#C9C4F4')
    expect(tokens['line-dim']).toBe('#3B3A78')
    expect(tokens['ink-0']).toBe('#F4F1FF')
    expect(tokens['ink-1']).toBe('#B7B3DC')
    expect(tokens['ink-2']).toBe('#8783B5')
    expect(tokens.growth).toBe('#3DE1B0')
    expect(tokens['growth-deep']).toBe('#178A69')
    expect(tokens.torch).toBe('#F2A83B')
    expect(tokens['torch-deep']).toBe('#B8641E')
    expect(tokens.ember).toBe('#FF5A4E')
    expect(tokens['ember-deep']).toBe('#B3261E')
  })

  it('pins the v1-only aliases to their v1 hex (A8)', () => {
    expect(tokens.alert).toBe('#EF4444')
    expect(tokens.streak).toBe('#F59E0B')
    expect(tokens.mute).toBe('#64748B')
    expect(tokens.alert).not.toBe(tokens.ember)
    expect(tokens.streak).not.toBe(tokens.torch)
    expect(tokens.mute).not.toBe(tokens['ink-2'])
  })

  it('v1 text pairs meet the A8 floor', () => {
    const pairs: [string, string, number, string][] = [
      [tokens['ground-0'], tokens.growth, 4.5, 'both · text on the growth fill'],
      [tokens['ground-0'], tokens.alert, 4.5, 'both · text on the alert fill'],
      [tokens.mute, tokens.paper, 4.5, 'light · caption on html'],
      [tokens.mute, '#FFFFFF', 4.5, 'light · caption in AppCard'],
      [tokens.ink, '#FFFFFF', 4.5, 'light · small done text'],
      [tokens.ink, tokens.paper, 4.5, 'light · small done text'],
      [tokens['growth-deep'], '#FFFFFF', 3, 'light · large green text, glyph, ring'],
      [tokens['growth-deep'], tokens.paper, 3, 'light · large green text, glyph, ring'],
      [tokens.growth, tokens['paper-dark'], 4.5, 'dark · green text'],
      [tokens.growth, tokens.ink, 4.5, 'dark · green text in AppCard'],
      [tokens.streak, tokens['paper-dark'], 4.5, 'dark · streak text'],
      [tokens.streak, tokens.ink, 4.5, 'dark · streak text in AppCard'],
      [tokens.alert, tokens['paper-dark'], 4.5, 'dark · bare error text'],
    ]
    for (const [fg, bg, min, label] of pairs) {
      expect(contrastRatio(fg, bg), label).toBeGreaterThanOrEqual(min)
    }
  })

  it('v1 pre-existing sub-AA pairs do not regress below main (A8)', () => {
    expect(contrastRatio(tokens.mute, tokens['paper-dark'])).toBeGreaterThanOrEqual(3.7)
    expect(contrastRatio(tokens.mute, tokens.ink)).toBeGreaterThanOrEqual(3.0)
    expect(contrastRatio(tokens.streak, '#FFFFFF')).toBeGreaterThanOrEqual(2.0)
    expect(contrastRatio(tokens.alert, '#FFFFFF')).toBeGreaterThanOrEqual(3.7)
  })

  it('keeps the v1-only tokens until plan 6 removes them', () => {
    expect(tokens.paper).toBe('#F8FAFC')
    expect(tokens['paper-dark']).toBe('#0F172A')
    expect(tokens.ink).toBe('#1E293B')
  })

  it('every WCAG pair in design §6 is at least 4.5:1', () => {
    const onGround1: [string, string][] = [
      [tokens['ink-0'], tokens['ground-1']],
      [tokens['ink-1'], tokens['ground-1']],
      [tokens['ink-2'], tokens['ground-1']],
      [tokens['line-lit'], tokens['ground-1']],
      [tokens.growth, tokens['ground-1']],
      [tokens.torch, tokens['ground-1']],
      [tokens.ember, tokens['ground-1']],
    ]
    for (const [fg, bg] of onGround1) {
      expect(contrastRatio(fg, bg)).toBeGreaterThanOrEqual(4.5)
    }
    expect(contrastRatio(tokens['ground-0'], tokens.growth)).toBeGreaterThanOrEqual(4.5)
    expect(contrastRatio(tokens['ground-0'], tokens.torch)).toBeGreaterThanOrEqual(4.5)
    expect(contrastRatio(tokens['ground-0'], tokens.ember)).toBeGreaterThanOrEqual(4.5)
    expect(contrastRatio(tokens['ink-0'], tokens['ember-deep'])).toBeGreaterThanOrEqual(4.5)
  })
})
