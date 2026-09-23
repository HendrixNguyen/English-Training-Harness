import { describe, expect, it } from 'vitest'
import { classifyContent } from '~/utils/content'

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
})
