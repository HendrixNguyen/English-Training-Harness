import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QuestNode from '~/components/retro/QuestNode.vue'
import { tokens } from '~/tailwind.config'
import type { QuestTask } from '~/stores/quest'

const task: QuestTask = { id: 't1', task_type: 'vocabulary', title: 'Từ vựng', duration_minutes: 10, is_completed: false, content_json: null }

describe('QuestNode (design §5)', () => {
  it('done: border-line-lit, star glyph, "Đã xong", aria-disabled', () => {
    const w = mount(QuestNode, { props: { task, index: 0, state: 'done' } })
    const button = w.find('button')
    expect(button.classes().join(' ')).toContain('border-line-lit')
    expect(w.find('[data-glyph="star"]').exists()).toBe(true)
    expect(w.text()).toContain('Đã xong')
    expect(button.attributes('aria-disabled')).toBe('true')
  })

  it('current: border-growth, "Vào", aria-current=step', () => {
    const w = mount(QuestNode, { props: { task, index: 0, state: 'current' } })
    const button = w.find('button')
    expect(button.classes().join(' ')).toContain('border-growth')
    expect(w.text()).toContain('Vào')
    expect(button.attributes('aria-current')).toBe('step')
  })

  it('open: border-line-lit, "Vào"', () => {
    const w = mount(QuestNode, { props: { task, index: 0, state: 'open' } })
    expect(w.find('button').classes().join(' ')).toContain('border-line-lit')
    expect(w.text()).toContain('Vào')
  })

  it('locked: ink-2, padlock glyph, "Khoá", aria-disabled', () => {
    const w = mount(QuestNode, { props: { task, index: 0, state: 'locked' } })
    expect(w.find('button').classes().join(' ')).toContain('text-ink-2')
    expect(w.find('[data-glyph="padlock"]').exists()).toBe(true)
    expect(w.text()).toContain('Khoá')
    expect(w.find('button').attributes('aria-disabled')).toBe('true')
  })

  it('locked icon dims every non-k char to ink-2 (design A3); k stays ground-0', () => {
    const w = mount(QuestNode, { props: { task, index: 0, state: 'locked' } })
    const style = w.find('button svg').attributes('style') ?? ''
    expect(style).toContain(`--px-l: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-i: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-d: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-T: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-k: ${tokens['ground-0']}`)
  })

  it('emits enter with the task id for open/current, never for locked/done', async () => {
    for (const state of ['open', 'current'] as const) {
      const w = mount(QuestNode, { props: { task, index: 0, state } })
      await w.find('button').trigger('click')
      expect(w.emitted('enter')).toEqual([['t1']])
    }
    for (const state of ['locked', 'done'] as const) {
      const w = mount(QuestNode, { props: { task, index: 0, state } })
      await w.find('button').trigger('click')
      expect(w.emitted('enter')).toBeUndefined()
    }
  })

  it('the tile button carries text-ink-0, or text-ink-2 when locked (design amend A2)', () => {
    const open = mount(QuestNode, { props: { task, index: 0, state: 'open' } })
    expect(open.find('button').classes()).toContain('text-ink-0')
    const locked = mount(QuestNode, { props: { task, index: 0, state: 'locked' } })
    expect(locked.find('button').classes()).toContain('text-ink-2')
    expect(locked.find('button').classes()).not.toContain('text-ink-0')
  })

  it('carries the torch focus classes (design amend A5)', () => {
    const w = mount(QuestNode, { props: { task, index: 0, state: 'open' } })
    const classes = w.find('button').classes()
    for (const c of ['focus-visible:outline', 'focus-visible:outline-2', 'focus-visible:outline-offset-2', 'focus-visible:outline-torch', 'focus-visible:ring-0', 'focus-visible:ring-offset-0']) {
      expect(classes).toContain(c)
    }
  })

  it('open/current get the pressed inset classes; locked/done do not (design amend A5)', () => {
    for (const state of ['open', 'current'] as const) {
      const w = mount(QuestNode, { props: { task, index: 0, state } })
      const classes = w.find('button').classes()
      expect(classes).toContain('active:translate-y-[2px]')
      expect(classes).toContain('active:border-line-dim')
    }
    for (const state of ['locked', 'done'] as const) {
      const w = mount(QuestNode, { props: { task, index: 0, state } })
      const classes = w.find('button').classes()
      expect(classes).not.toContain('active:translate-y-[2px]')
      expect(classes).not.toContain('active:border-line-dim')
    }
  })

  it('draws the connector lit/dim/none', () => {
    const lit = mount(QuestNode, { props: { task, index: 0, state: 'open', connector: 'lit' } })
    expect(lit.find('[data-connector]').classes()).toContain('bg-growth')
    const dim = mount(QuestNode, { props: { task, index: 0, state: 'open', connector: 'dim' } })
    expect(dim.find('[data-connector]').classes()).toContain('bg-line-dim')
    const none = mount(QuestNode, { props: { task, index: 0, state: 'open', connector: 'none' } })
    expect(none.find('[data-connector]').exists()).toBe(false)
  })
})
