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

  it('rows 22-25 and 30-31 (the pot rim and base) are identical across all six stages', () => {
    const stages = Object.keys(COMPANION) as (keyof typeof COMPANION)[]
    const rimOf = (rows: string[]) => [...rows.slice(22, 26), ...rows.slice(30, 32)]
    const reference = rimOf(COMPANION[stages[0]])
    for (const stage of stages) expect(rimOf(COMPANION[stage])).toEqual(reference)
  })

  it('rows 26-29 (the face) are identical across the five non-wilted stages', () => {
    const nonWilted = (Object.keys(COMPANION) as (keyof typeof COMPANION)[]).filter(s => s !== 'wilted')
    const faceOf = (rows: string[]) => rows.slice(26, 30)
    const reference = faceOf(COMPANION[nonWilted[0]])
    for (const stage of nonWilted) expect(faceOf(COMPANION[stage])).toEqual(reference)
  })

  it('wilted (folded bug): eyes closed on row 26 (k only at the two eye columns)', () => {
    const row = COMPANION.wilted[26]
    // Interior (excluding the pot's own left/right rim k's): k appears only
    // at the two eye-column pairs, same columns as the open face — a closed
    // eyelid is drawn with one row, not the open face's row26+row27 pair.
    expect(row.slice(9, 23)).toBe('TTTkkTTTTkkTTT')
  })

  it('wilted (folded bug): the mouth line moves up to row 28, cols 14-17', () => {
    expect(COMPANION.wilted[28].slice(14, 18)).toBe('kkkk')
    expect(COMPANION.wilted[28][13]).toBe('T')
    expect(COMPANION.wilted[28][18]).toBe('T')
  })

  it('wilted (folded bug): the mouth corners move down to row 29, cols 13 and 18 (inverted smile)', () => {
    expect(COMPANION.wilted[29][13]).toBe('k')
    expect(COMPANION.wilted[29][18]).toBe('k')
    expect(COMPANION.wilted[29].slice(14, 18)).toBe('TTTT')
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
