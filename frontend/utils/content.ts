import { sanitizeGlossary } from '~/utils/gloss'

export interface Gloss {
  term: string
  meaning_vi: string
  definition: string
}

export interface Word {
  term: string
  definition: string
}
export interface Question {
  id: string
  prompt: string
  options: Record<string, string>
}

export type Classified
  = { kind: 'words', words: Word[] }
    | { kind: 'reading', passage: string, glossary: Gloss[], questions: Question[] }
    | { kind: 'questions', questions: Question[] }
    | { kind: 'raw', text: string }

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function pick(cj: Record<string, unknown>, key: string): unknown {
  if (Array.isArray(cj[key])) return cj[key]
  const inner = cj.content
  if (isRecord(inner) && Array.isArray(inner[key])) return inner[key]
  return undefined
}

/** Picks a renderer for a task's content_json without ever returning nothing. */
export function classifyContent(contentJson: unknown): Classified {
  if (isRecord(contentJson)) {
    const words = pick(contentJson, 'words')
    if (Array.isArray(words) && words.length > 0) return { kind: 'words', words: words as Word[] }
    const questions = pick(contentJson, 'questions')
    const inner = isRecord(contentJson.content) ? contentJson.content : {}
    const passage = typeof contentJson.passage === 'string' && contentJson.passage.trim() ? contentJson.passage : inner.passage
    if (typeof passage === 'string' && passage.trim()) return { kind: 'reading', passage, questions: Array.isArray(questions) ? questions as Question[] : [], glossary: sanitizeGlossary(pick(contentJson, 'glossary'), passage) }
    if (Array.isArray(questions) && questions.length > 0) return { kind: 'questions', questions: questions as Question[] }
  }
  return { kind: 'raw', text: JSON.stringify(contentJson, null, 2) }
}

/** A roadmap outline has no learner-facing exercise content, even if it has a title. */
export function hasUsableContent(contentJson: unknown): boolean {
  const content = classifyContent(contentJson)
  const nonblank = (value: unknown) => typeof value === 'string' && value.trim().length > 0
  const validQuestion = (question: unknown) => isRecord(question) && nonblank(question.id) && nonblank(question.prompt) && isRecord(question.options) && Object.keys(question.options).length > 0 && Object.values(question.options).every(nonblank)
  if (content.kind === 'words') return content.words.every(word => isRecord(word) && nonblank(word.term) && nonblank(word.definition))
  if (content.kind === 'reading' || content.kind === 'questions') return content.questions.every(validQuestion)
  return false
}
