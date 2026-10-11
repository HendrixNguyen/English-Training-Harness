import type { Gloss } from '~/utils/content'

export type Segment = { text: string, gloss?: never } | { text: string, gloss: number }

/** Place longer terms first, skipping collisions but retaining the original text verbatim. */
export function segmentPassage(passage: string, glossary: Gloss[]): Segment[] {
  const marks: { start: number, end: number, gloss: number }[] = []
  glossary.map((entry, gloss) => ({ entry, gloss })).sort((a, b) => b.entry.term.length - a.entry.term.length).forEach(({ entry, gloss }) => {
    const term = entry.term.trim()
    if (!term) return
    const pattern = term.split(/\s+/u).map(part => part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('\\s+')
    const regex = new RegExp(`${pattern}[\\p{L}\\p{N}'’-]*`, 'giu')
    for (const match of passage.matchAll(regex)) {
      const start = match.index
      const end = start + match[0].length
      const previous = Array.from(passage.slice(0, start)).at(-1)
      if (previous && /[\p{L}\p{N}]/u.test(previous)) continue
      if (marks.some(mark => start < mark.end && end > mark.start)) continue
      marks.push({ start, end, gloss })
      break
    }
  })
  marks.sort((a, b) => a.start - b.start)
  const segments: Segment[] = []
  let offset = 0
  for (const mark of marks) {
    if (mark.start > offset) segments.push({ text: passage.slice(offset, mark.start) })
    segments.push({ text: passage.slice(mark.start, mark.end), gloss: mark.gloss })
    offset = mark.end
  }
  if (offset < passage.length || segments.length === 0) segments.push({ text: passage.slice(offset) })
  return segments
}

export function sanitizeGlossary(raw: unknown, passage: string): Gloss[] {
  if (!Array.isArray(raw)) return []
  const seen = new Set<string>()
  const valid: Gloss[] = []
  for (const value of raw) {
    if (typeof value !== 'object' || value === null) continue
    const entry = value as Record<string, unknown>
    if (![entry.term, entry.meaning_vi, entry.definition].every(v => typeof v === 'string' && v.trim())) continue
    const gloss = { term: (entry.term as string).trim(), meaning_vi: (entry.meaning_vi as string).trim(), definition: (entry.definition as string).trim() }
    const key = gloss.term.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    if (segmentPassage(passage, [gloss]).some(s => s.gloss !== undefined)) valid.push(gloss)
  }
  // Resolve overlaps before capping; terms that cannot be placed never consume a mark.
  const placed = new Set(segmentPassage(passage, valid).flatMap(s => s.gloss === undefined ? [] : [s.gloss]))
  return valid.filter((_, i) => placed.has(i)).slice(0, 8)
}
