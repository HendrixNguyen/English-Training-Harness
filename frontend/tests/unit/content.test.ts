import { describe, expect, it } from 'vitest'
import { classifyContent, hasUsableContent, hasTrustedContent } from '~/utils/content'

describe('classifyContent (design §2.4)', () => {
  it('finds a §6.2 vocabulary list at the top level or under content', () => {
    const words = [{ term: 'Inquire', definition: 'To ask for information' }]
    expect(classifyContent({ words })).toEqual({ kind: 'words', words, questions: [] })
    expect(classifyContent({ type: 'vocabulary', content: { words } })).toEqual({ kind: 'words', words, questions: [] })
  })

  it('finds a question list', () => {
    const questions = [{ id: 'q1', prompt: 'Choose…', options: { A: 'x', B: 'y' } }]
    expect(classifyContent({ content: { questions } })).toEqual({ kind: 'questions', questions })
  })

  it('classifies outlines as unavailable without exposing JSON', () => {
    const seed = { title: 'Day 1 vocabulary', duration_minutes: 10, day: 1, task: 'vocabulary' }
    expect(classifyContent(seed)).toEqual({ kind: 'raw', text: '' })
    expect(classifyContent(null)).toEqual({ kind: 'raw', text: '' })
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
    expect(hasUsableContent({ content: { passage: 'Plain text', questions: [] } })).toBe(false)
    expect(hasUsableContent({ passage: 'Plain text', questions: [] })).toBe(true)
    expect(hasUsableContent({ content_origin: 'legacy', content: { passage: 'Plain text' } })).toBe(true)
  })

})

describe('learning content trust', () => {
  it('preserves vocabulary examples and associated questions', () => {
    const words = [{ term: 'inquire', definition: 'ask', example: 'I inquire about the plan.' }]
    const questions = [{ id: 'q1', prompt: 'Which word means ask?', options: { A: 'inquire', B: 'reply' } }]
    expect(classifyContent({ content: { words, questions } })).toEqual({ kind: 'words', words, questions })
  })
  it('does not trust an unmarked nested study outline or a one-option quiz', () => {
    expect(hasUsableContent({ content: { passage: 'A draft passage.' } })).toBe(false)
    expect(hasUsableContent({ content: { words: [{ term: 'ask', definition: 'inquire' }] } })).toBe(false)
    expect(hasUsableContent({ questions: [{ id: 'q', prompt: 'Which term?', options: { A: 'ask' } }] })).toBe(false)
  })
})


describe('server-generated trust', () => {
  const qs = Array.from({ length: 4 }, (_, i) => ({ id: `q${i}`, prompt: `Which form completes item ${i}?`, options: { A: 'am', B: 'is', C: 'are', D: 'be' }, answer: 'B', explanation: 'The singular form is required.' }))
  const task = (body: unknown) => ({ task_type: 'practice', content_origin: 'generated', content_revision: 'server-revision', content_json: { content_schema: 1, content_level: 'B1', content: body } })
  it('requires server identity and validates count, options, distinct questions and explanation', () => {
    expect(hasTrustedContent(task({ questions: qs }))).toBe(true)
    expect(hasTrustedContent({ ...task({ questions: qs }), content_revision: null })).toBe(false)
    expect(hasTrustedContent({ ...task({ questions: qs }), content_origin: 'outline' })).toBe(false)
    expect(hasTrustedContent(task({ questions: qs.slice(0, 3) }))).toBe(false)
    expect(hasTrustedContent(task({ questions: qs.map(q => ({ ...q, explanation: 'TODO' })) }))).toBe(false)
    expect(hasTrustedContent(task({ questions: qs.map(q => ({ ...q, prompt: qs[0]!.prompt })) }))).toBe(false)
    expect(hasTrustedContent(task({ questions: qs.map(q => ({ ...q, options: { ...q.options, B: ' AM ' } })) }))).toBe(false)
  })
  it('validates the actual task-type contract while permitting unknown fields as the server does', () => {
    const value = task({ questions: qs, passage: 42, words: 'unknown field' })
    expect(hasTrustedContent(value)).toBe(true)
    expect(classifyContent(value.content_json, value.task_type).kind).toBe('questions')
    const reading = { ...value, task_type: 'reading', content_json: { content_schema: 1, content_level: 'A1', content: { passage: 'A clear passage. '.repeat(20), questions: qs.slice(0, 3), glossary: [null, { term: null }] } } }
    expect(hasTrustedContent(reading)).toBe(true)
    expect(hasTrustedContent({ ...reading, content_json: { ...reading.content_json, content_level: 'C2' } })).toBe(false)
  })
})
