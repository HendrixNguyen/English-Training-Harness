import { describe, expect, it } from 'vitest'
import { classifyContent, hasUsableContent } from '~/utils/content'

describe('classifyContent (design §2.4)', () => {
  it('finds a §6.2 vocabulary list at the top level or under content', () => {
    const words = [{ term: 'Inquire', definition: 'To ask for information' }]
    expect(classifyContent({ words })).toEqual({ kind: 'words', words })
    expect(classifyContent({ type: 'vocabulary', content: { words } })).toEqual({ kind: 'words', words })
  })

  it('finds a question list', () => {
    const questions = [{ id: 'q1', prompt: 'Choose…', options: { A: 'x', B: 'y' } }]
    expect(classifyContent({ content: { questions } })).toEqual({ kind: 'questions', questions })
  })

  it('falls back to raw JSON for anything else (the demo seed shape)', () => {
    const seed = { title: 'Day 1 vocabulary', duration_minutes: 10, day: 1, task: 'vocabulary' }
    expect(classifyContent(seed)).toEqual({ kind: 'raw', text: JSON.stringify(seed, null, 2) })
    expect(classifyContent(null)).toEqual({ kind: 'raw', text: 'null' })
  })
  it('recognizes nested reading while preserving exact whitespace and sanitizing optional glosses', () => {
    const passage = '  A contract.\nNext line.  '
    const questions = [{ id: 'q1', prompt: 'Choose', options: { A: 'Yes' } }]
    expect(classifyContent({ content: { passage, questions } })).toEqual({ kind: 'reading', passage, questions, glossary: [] })
    expect(classifyContent({ passage: ' ', content: { passage, questions } }).kind).toBe('reading')
    expect(classifyContent({ passage: ' ', questions }).kind).toBe('questions')
  })
  it('distinguishes usable cached content from outlines and malformed exercise arrays', () => {
    expect(hasUsableContent({ content_schema: 1, title: 'outline' })).toBe(false)
    expect(hasUsableContent({ words: [null] })).toBe(false)
    expect(hasUsableContent({ questions: [{ id: 'q1', prompt: 'Choose', options: {} }] })).toBe(false)
    expect(hasUsableContent({ content: { passage: 'Plain text', questions: [] } })).toBe(true)
  })

})
