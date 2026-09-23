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
    if (Array.isArray(questions) && questions.length > 0) return { kind: 'questions', questions: questions as Question[] }
  }
  return { kind: 'raw', text: JSON.stringify(contentJson, null, 2) }
}
