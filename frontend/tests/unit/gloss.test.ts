import { describe, expect, it } from 'vitest'
import { sanitizeGlossary, segmentPassage } from '~/utils/gloss'
const gloss = (term: string) => ({ term, meaning_vi: 'đàm phán', definition: 'reach agreement' })
const marked = (passage: string, terms: string[]) => segmentPassage(passage, terms.map(gloss)).filter(s => 'gloss' in s).map(s => s.text)
describe('lossless glossary segmentation', () => {
  it('marks first boundary occurrences, inflections and punctuation losslessly', () => {
    const passage = 'Negotiated, then negotiate. start art!\nXin chào — café.'
    const segments = segmentPassage(passage, ['negotiate', 'art', 'café'].map(gloss))
    expect(segments.map(s => s.text).join('')).toBe(passage)
    expect(segments.filter(s => 'gloss' in s).map(s => s.text)).toEqual(['Negotiated', 'art', 'café'])
    expect(marked('éart 2art art-work’s', ['art'])).toEqual(['art-work’s'])
  })
  it('escapes patterns, matches whitespace, and places longer overlapping terms first', () => {
    expect(marked('C++ and C++', ['C++'])).toEqual(['C++'])
    expect(marked('New\n  York then New York; York.', ['York', 'New York'])).toEqual(['New\n  York', 'York'])
    expect(marked('new york', ['new', 'new york'])).toEqual(['new york'])
  })
  it('finds an overlapping valid phrase after rejecting a Unicode word-internal occurrence', () => {
    for (const passage of ['shad had had enough.', 'éhad had had enough.', '2had had had enough.', '𝒔had had had enough.']) {
      const glossary = [gloss('had had')]
      const start = passage.indexOf('had had', passage.indexOf('had had') + 1)
      const segments = segmentPassage(passage, glossary)
      expect(segments).toEqual([{ text: passage.slice(0, start) }, { text: 'had had', gloss: 0 }, { text: ' enough.' }])
      expect(segments.map(segment => segment.text).join('')).toBe(passage)
      expect(sanitizeGlossary(glossary, passage)).toEqual(glossary)
    }
  })
  it('drops malformed, blank, duplicate, absent and unplaceable entries and caps eight', () => {
    const passage = 'one two three four five six seven eight nine'
    const entries = [null, gloss('absent'), { ...gloss('one'), definition: ' ' }, gloss(' one '), gloss('ONE'), ...'two three four five six seven eight nine'.split(' ').map(gloss)]
    const clean = sanitizeGlossary(entries, passage)
    expect(clean).toHaveLength(8)
    expect(clean[0]).toEqual(gloss('one'))
    expect(sanitizeGlossary([gloss('new'), gloss('new york')], 'new york')).toEqual([gloss('new york')])
    expect(sanitizeGlossary(undefined, passage)).toEqual([])
  })
})
