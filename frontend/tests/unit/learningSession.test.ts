import { beforeEach, describe, expect, it, vi } from 'vitest'
import { classifyContent } from '~/utils/content'
import { initialSession, readDraft, writeDraft, clearOwnerDrafts, validateSession } from '~/utils/learningSession'
const content = classifyContent({ questions: [1, 2].map(n => ({ id: `q${n}`, prompt: `Which item follows ${n}?`, options: { A: 'First', B: 'Second' }, answer: 'B', explanation: 'The second item follows.' })) })
const scope = { owner: 'owner', task: 'task', revision: 'revision' }
describe('revision-scoped learning drafts', () => {
  beforeEach(() => { localStorage.clear(); clearOwnerDrafts(scope.owner) })
  it('keeps pending selection separate from submitted answers and restores feedback/lookback/duration', () => {
    const s = initialSession(content)
    s.pending = 'A'; writeDraft(scope, s)
    expect(readDraft(scope, content)?.answers).toEqual({})
    s.phase = 'feedback'; s.answers = { q1: 'A' }; s.passageOpen = true
    writeDraft(scope, s)
    expect(readDraft(scope, content)).toEqual(s)
    s.phase = 'summary'; s.cursor = 2; s.pending = null; s.answers.q2 = 'B'; s.duration = 47; s.uncertain = true
    writeDraft(scope, s); expect(readDraft(scope, content)).toEqual(s)
    expect(readDraft({ ...scope, owner: 'other' }, content)).toBeNull()
    expect(readDraft({ ...scope, task: 'other' }, content)).toBeNull()
    expect(readDraft({ ...scope, revision: 'changed' }, content)).toBeNull()
  })
  it('rejects skipped cursors, invalid options, impossible feedback and partial summary', () => {
    const s = initialSession(content)
    expect(validateSession({ ...s, cursor: 1 }, content)).toBeNull()
    expect(validateSession({ ...s, pending: 'Z' }, content)).toBeNull()
    expect(validateSession({ ...s, phase: 'feedback' }, content)).toBeNull()
    expect(validateSession({ ...s, phase: 'summary' }, content)).toBeNull()
  })
  it('storage denial still keeps drafts in memory and account clearing removes them', () => {
    const denied = vi.spyOn(localStorage, 'setItem').mockImplementation(() => { throw Error('denied') })
    const s = initialSession(content); s.pending = 'B'
    expect(writeDraft(scope, s)).toBe(false)
    expect(readDraft(scope, content)).toEqual(s)
    clearOwnerDrafts(scope.owner)
    expect(readDraft(scope, content)).toBeNull()
    denied.mockRestore()
  })
})

describe('draft storage transitions', () => {
  it('uses the latest in-memory selection if storage becomes unwritable after an earlier draft', () => {
    clearOwnerDrafts(scope.owner); localStorage.clear()
    const s = initialSession(content); s.pending = 'A'; writeDraft(scope, s)
    const denied = vi.spyOn(localStorage, 'setItem').mockImplementation(() => { throw Error('denied') })
    s.pending = 'B'; writeDraft(scope, s)
    expect(readDraft(scope, content)?.pending).toBe('B')
    denied.mockRestore()
  })
})

describe('draft shape isolation', () => {
  it('rejects malformed/foreign draft bodies and corrupt durations without blocking a fresh session', () => {
    clearOwnerDrafts(scope.owner); localStorage.clear()
    const key = 'aelp.learning-draft.v1:' + JSON.stringify([scope.owner, scope.task, scope.revision])
    localStorage.setItem(key, '{broken'); expect(readDraft(scope, content)).toBeNull()
    localStorage.setItem(key, JSON.stringify({ ...scope, owner: 'foreign', session: initialSession(content) })); expect(readDraft(scope, content)).toBeNull()
    const summary = { ...initialSession(content), phase: 'summary', cursor: 2, answers: { q1: 'A', q2: 'B' }, duration: 47.5 }
    expect(validateSession(summary, content)).toBeNull()
    expect(validateSession({ ...summary, duration: 3601 }, content)).toBeNull()
    expect(validateSession({ ...summary, duration: 47, answers: { q1: 'A', unknown: 'B' } }, content)).toBeNull()
  })
})
