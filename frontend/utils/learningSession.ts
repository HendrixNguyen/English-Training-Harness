import { isRecord, questionsOf, type Classified } from '~/utils/content'
export interface DraftScope { owner: string, task: string, revision: string }
export interface LearningSession {
  phase: 'study' | 'selection' | 'feedback' | 'summary'
  cursor: number
  pending: string | null
  answers: Record<string, string>
  passageOpen: boolean
  duration: number | null
  uncertain: boolean
}
const prefix = 'aelp.learning-draft.v1:'
const memory = new Map<string, LearningSession>()
const keyOf = (s: DraftScope) => prefix + JSON.stringify([s.owner, s.task, s.revision])
const clone = (s: LearningSession): LearningSession => ({ ...s, answers: { ...s.answers } })
export function initialSession(content: Classified): LearningSession {
  return { phase: content.kind === 'reading' || content.kind === 'words' ? 'study' : 'selection', cursor: 0, pending: null, answers: {}, passageOpen: false, duration: null, uncertain: false }
}
export function validateSession(value: unknown, content: Classified): LearningSession | null {
  if (!isRecord(value) || !['study', 'selection', 'feedback', 'summary'].includes(String(value.phase)) || !Number.isInteger(value.cursor) || (value.cursor as number) < 0 || !isRecord(value.answers) || typeof value.passageOpen !== 'boolean' || typeof value.uncertain !== 'boolean' || !(value.duration === null || (Number.isInteger(value.duration) && (value.duration as number) >= 1 && (value.duration as number) <= 3600))) return null
  const qs = questionsOf(content); const cursor = value.cursor as number; const answers = value.answers
  if (Object.entries(answers).some(([id, opt]) => { const q = qs.find(q => q.id === id); return !q || typeof opt !== 'string' || !Object.hasOwn(q.options, opt) })) return null
  const count = Object.keys(answers).length
  if (value.phase === 'study') {
    if (!(content.kind === 'reading' ? cursor === 0 : content.kind === 'words' && cursor < content.words.length) || count !== 0 || value.pending !== null) return null
  } else if (value.phase === 'summary') {
    if (count !== qs.length || value.pending !== null || cursor !== qs.length) return null
  } else {
    const q = qs[cursor]
    if (!q || qs.slice(0, cursor).some(q => !Object.hasOwn(answers, q.id)) || qs.slice(cursor + 1).some(q => Object.hasOwn(answers, q.id))) return null
    if (value.pending !== null && (typeof value.pending !== 'string' || !Object.hasOwn(q.options, value.pending))) return null
    if (value.phase === 'feedback' ? count !== cursor + 1 || answers[q.id] !== value.pending : count !== cursor || Object.hasOwn(answers, q.id)) return null
  }
  if ((value.duration !== null || value.uncertain) && value.phase !== 'summary') return null
  return clone(value as unknown as LearningSession)
}
export function readDraft(scope: DraftScope, content: Classified): LearningSession | null {
  const key = keyOf(scope)
  let value: unknown = memory.get(key)
  try { const raw = value === undefined ? localStorage.getItem(key) : null; if (raw) { const body = JSON.parse(raw); value = isRecord(body) && body.owner === scope.owner && body.task === scope.task && body.revision === scope.revision ? body.session : null } } catch { /* memory fallback */ }
  const valid = validateSession(value, content)
  if (!valid) { clearDraft(scope); return null }
  return valid
}
export function writeDraft(scope: DraftScope, session: LearningSession): boolean {
  const key = keyOf(scope); memory.set(key, clone(session))
  try { localStorage.setItem(key, JSON.stringify({ ...scope, session })); return true } catch { return false }
}
export function clearDraft(scope: DraftScope) {
  const key = keyOf(scope); memory.delete(key)
  try { localStorage.removeItem(key) } catch { /* denied */ }
}
export function clearOwnerDrafts(owner: string, except?: DraftScope) {
  const matches = (key: string) => { try { return key.startsWith(prefix) && JSON.parse(key.slice(prefix.length))[0] === owner && (!except || key !== keyOf(except)) } catch { return false } }
  for (const key of memory.keys()) if (matches(key)) memory.delete(key)
  try { for (let i = localStorage.length - 1; i >= 0; i--) { const key = localStorage.key(i); if (key && matches(key)) localStorage.removeItem(key) } } catch { /* denied */ }
}
export function clearTaskDrafts(scope: DraftScope) {
  const matches = (key: string) => { try { const [owner, task] = JSON.parse(key.slice(prefix.length)); return key.startsWith(prefix) && owner === scope.owner && task === scope.task && key !== keyOf(scope) } catch { return false } }
  for (const key of memory.keys()) if (matches(key)) memory.delete(key)
  try { for (let i = localStorage.length - 1; i >= 0; i--) { const key = localStorage.key(i); if (key && matches(key)) localStorage.removeItem(key) } } catch { /* denied */ }
}
