import { readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import AppHeader from '~/components/AppHeader.vue'
import QuestRow from '~/components/quest/QuestRow.vue'
import RoadmapNode from '~/components/roadmap/RoadmapNode.vue'
import AppButton from '~/components/ui/AppButton.vue'
import SegmentedProgress from '~/components/ui/SegmentedProgress.vue'
import type { QuestTask } from '~/stores/quest'
import type { RoadmapNode as RoadmapNodeData } from '~/utils/roadmap'

describe('v1 text on growth/alert fills is ground-0; v1 green text follows the scheme (design A9)', () => {
  it('AppButton primary is bg-growth text-ground-0, never text-white', () => {
    const w = mount(AppButton, { props: { variant: 'primary' } })
    expect(w.classes()).toContain('bg-growth')
    expect(w.classes()).toContain('text-ground-0')
    expect(w.classes()).not.toContain('text-white')
  })

  it('AppButton danger is bg-alert text-ground-0, never text-white', () => {
    const w = mount(AppButton, { props: { variant: 'danger' } })
    expect(w.classes()).toContain('bg-alert')
    expect(w.classes()).toContain('text-ground-0')
    expect(w.classes()).not.toContain('text-white')
  })

  it('AppHeader avatar is text-ground-0', () => {
    setActivePinia(createPinia())
    const w = mount(AppHeader, { props: { streak: 3 } })
    expect(w.find('[aria-label="Tài khoản"]').classes()).toContain('text-ground-0')
  })

  it('QuestRow next-task "Học" link is text-ground-0', () => {
    const task: QuestTask = { id: 't1', task_type: 'vocabulary', title: 'Task', duration_minutes: 10, is_completed: false, content_json: null }
    const w = mount(QuestRow, { props: { task, index: 1, state: 'next' } })
    expect(w.find(`[aria-label="Học: ${task.title}"]`).classes()).toContain('text-ground-0')
  })

  it('QuestRow done glyph is text-growth-deep dark:text-growth', () => {
    const task: QuestTask = { id: 't1', task_type: 'vocabulary', title: 'Task', duration_minutes: 10, is_completed: true, content_json: null }
    const w = mount(QuestRow, { props: { task, index: 1, state: 'done' } })
    const glyph = w.find('span[aria-hidden="true"]')
    expect(glyph.classes()).toContain('text-growth-deep')
    expect(glyph.classes()).toContain('dark:text-growth')
  })

  it('QuestRow done label is text-ink dark:text-growth', () => {
    const task: QuestTask = { id: 't1', task_type: 'vocabulary', title: 'Task', duration_minutes: 10, is_completed: true, content_json: null }
    const w = mount(QuestRow, { props: { task, index: 1, state: 'done' } })
    const label = w.findAll('span').find(s => s.text() === 'Xong')!
    expect(label.classes()).toContain('text-ink')
    expect(label.classes()).toContain('dark:text-growth')
  })

  it('RoadmapNode today state is bg-growth text-ground-0', () => {
    const node: RoadmapNodeData = { day: 9, week: 2, state: 'today' }
    const w = mount(RoadmapNode, { props: { node }, global: { stubs: { NuxtLink: true } } })
    expect(w.classes()).toContain('bg-growth')
    expect(w.classes()).toContain('text-ground-0')
  })

  it('SegmentedProgress met is text-growth-deep dark:text-growth', () => {
    const w = mount(SegmentedProgress, { props: { valueSeconds: 1800, label: 'x', met: true } })
    const counter = w.find('.font-display')
    expect(counter.classes()).toContain('text-growth-deep')
    expect(counter.classes()).toContain('dark:text-growth')
  })

  it('no class string under components/ or pages/ has a growth/alert fill and text-white together (Acceptance amend 2, item 3)', () => {
    const roots = ['components', 'pages']
    const offenders: string[] = []
    for (const root of roots) {
      const dir = resolve(process.cwd(), root)
      const entries = readdirSync(dir, { recursive: true }) as string[]
      for (const entry of entries) {
        if (!entry.endsWith('.vue'))
          continue
        const path = resolve(dir, entry)
        const content = readFileSync(path, 'utf-8')
        const classAttrs = content.match(/class="[^"]*"/g) ?? []
        for (const attr of classAttrs) {
          const hasFill = /bg-(growth|alert)(?!\/)/.test(attr)
          const hasWhite = /text-white/.test(attr)
          if (hasFill && hasWhite)
            offenders.push(`${root}/${entry}: ${attr}`)
        }
      }
    }
    expect(offenders).toEqual([])
  })
})
