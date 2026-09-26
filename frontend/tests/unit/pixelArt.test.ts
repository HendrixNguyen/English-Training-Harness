import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PixelArt from '~/components/retro/PixelArt.vue'
import { COMPANION, GLYPHS, PALETTE } from '~/utils/pixelArt'
import { tokens } from '~/tailwind.config'

const SPROUT = [
  '................................',
  '................................',
  '................................',
  '................................',
  '................................',
  '................................',
  '................................',
  '................................',
  '................................',
  '...................kkk..........',
  '..................kgggk.........',
  '........kkk......kggigk.........',
  '.......kgggk.....kgggGk.........',
  '......kgggigk....kgGGk..........',
  '......kggggGk.kgGGGk............',
  '.......kgggGGkkgGk..............',
  '.........kkkkkkgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
  '........kkkkkkkkkkkkkkkk........',
  '.......kttttttttttttttttk.......',
  '.......kTTTTTTTTTTTTTTTTk.......',
  '........kTTTTTTTTTTTTTTk........',
  '........kTTTkkTTTTkkTTTk........',
  '........kTTTkiTTTTkiTTTk........',
  '........kTTTTkTTTTkTTTTk........',
  '.........kTTTTkkkkTTTTk.........',
  '.........kTTTTTTTTTTTTk.........',
  '.........kkkkkkkkkkkkkk.........',
]

function usedChars(rows: string[]): Set<string> {
  const set = new Set<string>()
  for (const row of rows) for (const ch of row) set.add(ch)
  return set
}

describe('pixel art data (design §4)', () => {
  const paletteChars = new Set(['.', ...Object.keys(PALETTE)])

  it('every companion stage is square and uses only palette chars', () => {
    for (const stage of Object.keys(COMPANION)) {
      const rows = COMPANION[stage as keyof typeof COMPANION]
      expect(rows).toHaveLength(32)
      for (const row of rows) expect(row).toHaveLength(32)
      for (const ch of usedChars(rows)) expect(paletteChars.has(ch)).toBe(true)
    }
  })

  it('every glyph is square and uses only palette chars', () => {
    for (const name of Object.keys(GLYPHS)) {
      const rows = GLYPHS[name as keyof typeof GLYPHS]
      const n = name === 'cursor' ? 8 : 16
      expect(rows).toHaveLength(n)
      for (const row of rows) expect(row).toHaveLength(n)
      for (const ch of usedChars(rows)) expect(paletteChars.has(ch)).toBe(true)
    }
  })

  it('COMPANION.sprout equals the design §4 sketch, verbatim', () => {
    expect(COMPANION.sprout).toEqual(SPROUT)
  })

  it('rows 22-31 (the pot) are identical across all six stages', () => {
    const stages = Object.keys(COMPANION) as (keyof typeof COMPANION)[]
    const potOf = (rows: string[]) => rows.slice(22, 32)
    const reference = potOf(COMPANION[stages[0]])
    for (const stage of stages) expect(potOf(COMPANION[stage])).toEqual(reference)
  })

  it('PixelArt renders a plausible rect count for sprout and skips "." entirely', () => {
    const palette: Record<string, string> = {}
    for (const [char, role] of Object.entries(PALETTE)) palette[char] = (tokens as Record<string, string>)[role]
    const w = mount(PixelArt, { props: { rows: COMPANION.sprout, palette, size: 128 } })
    const rects = w.findAll('rect')
    expect(rects.length).toBeGreaterThan(40)
    expect(rects.length).toBeLessThan(400)
  })
})
