import { sanitizeGlossary } from '~/utils/gloss'

export interface Gloss { term: string, meaning_vi: string, definition: string }
export interface Word { term: string, definition: string, example?: string }
export interface Question { id: string, prompt: string, options: Record<string, string>, answer?: string, explanation?: string }
export type Classified
  = { kind: 'words', words: Word[], questions: Question[] }
    | { kind: 'reading', passage: string, glossary: Gloss[], questions: Question[] }
    | { kind: 'questions', questions: Question[] }
    | { kind: 'raw', text: string }
export interface ContentIdentity { content_json: unknown, task_type: string, content_origin?: string, content_revision?: string | null }
export function isRecord(v: unknown): v is Record<string, unknown> { return typeof v === 'object' && v !== null && !Array.isArray(v) }
const nonblank = (v: unknown): v is string => typeof v === 'string' && v.trim().length > 0
const normalize = (v: string) => v.trim().replace(/\s+/gu, ' ').toLowerCase()
function placeholder(v: string): boolean {
  const text = normalize(v)
  return !/[\p{L}\p{N}]/u.test(text) || ['todo', 'tbd'].includes(text.replace(/[.:!?…]+$/u, ''))
    || ['<a complete distinct question>', '<a meaningful sentence explaining the correct answer in context>'].includes(text)
    || ['prompt', 'question', 'question prompt', 'explanation'].some(label => [`<${label}>`, `[${label}]`, `{{${label}}}`, `[insert ${label} here]`, `insert ${label} here`, `${label} goes here`].includes(text))
}
function bodyOf(cj: Record<string, unknown>): Record<string, unknown> { return isRecord(cj.content) ? cj.content : cj }
/** Classification preserves exact study text; provenance validation is a separate gate. */
export function classifyContent(contentJson: unknown, taskType?: string): Classified {
  if (!isRecord(contentJson)) return { kind: 'raw', text: '' }
  const body = bodyOf(contentJson)
  // Legacy shapes predate task typing; retain their complete study material.
  if (contentJson.content_schema !== 1) taskType = undefined
  const questions = Array.isArray(body.questions) ? body.questions as Question[] : []
  if ((!taskType || taskType === 'vocabulary') && Array.isArray(body.words) && body.words.length) return { kind: 'words', words: body.words as Word[], questions }
  if ((!taskType || taskType === 'reading') && nonblank(body.passage)) return { kind: 'reading', passage: body.passage, questions, glossary: sanitizeGlossary(body.glossary, body.passage) }
  if (questions.length) return { kind: 'questions', questions }
  return { kind: 'raw', text: '' }
}
export function questionsOf(content: Classified): Question[] { return content.kind === 'raw' ? [] : content.questions }
function validShape(body: Record<string, unknown>, generated: boolean): boolean {
  if (new TextEncoder().encode(JSON.stringify(body)).length > 32768) return false
  if (body.passage != null && typeof body.passage !== 'string') return false
  for (const key of ['words', 'questions', 'glossary']) if (body[key] != null && !Array.isArray(body[key])) return false
  const words = (body.words ?? []) as unknown[]
  if (!words.every(w => isRecord(w) && nonblank(w.term) && nonblank(w.definition) && (w.example == null || typeof w.example === 'string'))) return false
  const glosses = (body.glossary ?? []) as unknown[]
  if (!glosses.every(g => g === null || isRecord(g) && ['term', 'meaning_vi', 'definition'].every(k => g[k] == null || typeof g[k] === 'string'))) return false
  const questions = (body.questions ?? []) as unknown[]
  const ids = new Set<string>(); const prompts = new Set<string>()
  return questions.every(q => {
    if (!isRecord(q) || !nonblank(q.id) || ids.has(q.id.trim()) || !nonblank(q.prompt) || placeholder(q.prompt) || prompts.has(normalize(q.prompt)) || !isRecord(q.options)) return false
    ids.add(q.id.trim()); prompts.add(normalize(q.prompt))
    const entries = Object.entries(q.options)
    if (entries.length < 2 || entries.length > 4 || (generated && entries.length !== 4) || !entries.every(([k, v]) => /^[A-D]$/u.test(k) && nonblank(v)) || new Set(entries.map(([, v]) => normalize(v as string))).size !== entries.length) return false
    if (q.answer != null && typeof q.answer !== 'string') return false
    if (q.explanation != null && typeof q.explanation !== 'string') return false
    if (q.answer && !Object.hasOwn(q.options, q.answer as string)) return false
    if (q.explanation && (!q.answer || placeholder(q.explanation as string))) return false
    return !generated || (nonblank(q.answer) && nonblank(q.explanation))
  })
}
/** Recognized legacy material only; a schema marker by itself never grants trust. */
export function hasUsableContent(contentJson: unknown): boolean {
  if (!isRecord(contentJson) || contentJson.content_schema === 1) return false
  const body = bodyOf(contentJson)
  if (!validShape(body, false) || classifyContent(contentJson).kind === 'raw') return false
  return !isRecord(contentJson.content) || contentJson.content_origin === 'legacy' || (Array.isArray(body.questions) && body.questions.length > 0)
}
/** Backend-origin + opaque revision + revalidated shape, used for rendering and typed caches. */
export function hasTrustedContent(task: ContentIdentity | null | undefined): boolean {
  if (!task || !nonblank(task.content_revision) || !['generated', 'legacy'].includes(task.content_origin ?? '') || !isRecord(task.content_json)) return false
  const root = task.content_json; const body = bodyOf(root)
  if (task.content_origin === 'legacy') return root.content_schema !== 1 && validShape(body, false) && classifyContent(root).kind !== 'raw'
  if (root.content_schema !== 1 || !isRecord(root.content) || new TextEncoder().encode(JSON.stringify(body)).length > 32768) return false
  const fields = task.task_type === 'vocabulary' ? ['words', 'questions'] : task.task_type === 'reading' ? ['passage', 'glossary', 'questions'] : ['questions']
  const typedBody = Object.fromEntries(fields.filter(key => Object.hasOwn(body, key)).map(key => [key, body[key]]))
  if (!validShape(typedBody, true)) return false
  const levels: Record<string, [number, number]> = { A1: [240, 720], A2: [360, 1040], B1: [520, 1440], B2: [720, 2000], C1: [1000, 2560], C2: [1200, 3040] }
  if (root.content_level != null && typeof root.content_level !== 'string') return false
  const level = root.content_level as string | undefined
  if (level && !levels[level]) return false
  const q = (body.questions ?? []) as Question[]
  if (task.task_type === 'vocabulary') return Array.isArray(body.words) && body.words.length >= 5 && body.words.length <= 8 && q.length >= 3 && q.length <= 5
  if (task.task_type === 'reading') {
    const [min, max] = level ? levels[level]! : [240, 3040]
    const n = typeof body.passage === 'string' ? Array.from(body.passage.trim()).length : 0
    return n >= min && n <= max && q.length >= 3 && q.length <= 5
  }
  return task.task_type === 'practice' && q.length >= 4 && q.length <= 8
}
