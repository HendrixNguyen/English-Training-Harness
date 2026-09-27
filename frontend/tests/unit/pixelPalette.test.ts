import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Badge from '~/components/retro/Badge.vue'
import Chest from '~/components/retro/Chest.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import MapNode from '~/components/retro/MapNode.vue'
import QuestNode from '~/components/retro/QuestNode.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { tokens } from '~/tailwind.config'
import { PLANT_STAGES } from '~/utils/plant'
import type { QuestTask } from '~/stores/quest'

/**
 * Design A3: `PixelArt` always defines every `--px-*` variable from
 * `PALETTE` before overlaying a caller's overrides, so no `<rect>` in any
 * kit component or state can reference an undefined CSS variable (which
 * the browser resolves to black — the `retro-kit-screens` bug).
 */

const task: QuestTask = { id: 't1', task_type: 'vocabulary', title: 'Từ vựng', duration_minutes: 10, is_completed: false, content_json: null }

/** Every `rect[fill^="var(--px-"]` in `wrapper` must have its variable
 * present and non-empty on the nearest ancestor `<svg style>`. */
function assertEveryRectVarDefined(html: string, label: string) {
  const svgBlocks = [...html.matchAll(/<svg[^>]*style="([^"]*)"[^>]*>([\s\S]*?)<\/svg>/g)]
  for (const [, style, inner] of svgBlocks) {
    const vars = new Set<string>()
    for (const decl of style.split(';')) {
      const m = decl.match(/(--px-[^:]+):\s*(.*)/)
      if (m) vars.add(m[1].trim())
    }
    const rectFills = [...inner.matchAll(/fill="var\((--px-[^)]+)\)"/g)].map(m => m[1])
    for (const varName of rectFills) {
      expect(vars.has(varName), `${label}: ${varName} missing from svg style (${style})`).toBe(true)
      const decl = style.split(';').find(d => d.trim().startsWith(`${varName}:`))
      expect(decl, `${label}: ${varName} has no declaration`).toBeDefined()
      const value = decl!.split(':')[1]?.trim()
      expect(value, `${label}: ${varName} is empty`).toBeTruthy()
    }
  }
}

describe('PixelArt palette fallback (design A3)', () => {
  it.each(['done', 'current', 'open', 'locked'] as const)('QuestNode state=%s defines every --px-* var', (state) => {
    const w = mount(QuestNode, { props: { task, index: 0, state } })
    assertEveryRectVarDefined(w.html(), `QuestNode ${state}`)
  })

  it('QuestNode locked icon dims every non-k char to ink-2, k stays ground-0', () => {
    const w = mount(QuestNode, { props: { task, index: 0, state: 'locked' } })
    const svg = w.find('button svg')
    const style = svg.attributes('style') ?? ''
    expect(style).toContain(`--px-l: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-i: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-d: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-T: ${tokens['ink-2']}`)
    expect(style).toContain(`--px-k: ${tokens['ground-0']}`)
  })

  it.each(['cleared', 'today', 'partial', 'missed', 'locked'] as const)('MapNode state=%s defines every --px-* var', (state) => {
    const w = mount(MapNode, { props: { day: 3, state, title: 'x' } })
    assertEveryRectVarDefined(w.html(), `MapNode ${state}`)
  })

  it.each([false, true])('Chest open=%s defines every --px-* var', async (open) => {
    const w = mount(Chest, { props: { items: [{ icon: '🔥', label: 'x' }], open } })
    assertEveryRectVarDefined(w.html(), `Chest open=${open}`)
  })

  it.each([true, false])('Badge earned=%s defines every --px-* var', (earned) => {
    const w = mount(Badge, { props: { kind: 'streak', count: 1, earned } })
    assertEveryRectVarDefined(w.html(), `Badge earned=${earned}`)
  })

  it.each(PLANT_STAGES)('CompanionSprite stage=%s defines every --px-* var', (stage) => {
    const w = mount(CompanionSprite, { props: { stage, health: 80 } })
    assertEveryRectVarDefined(w.html(), `CompanionSprite ${stage}`)
  })

  it('CompanionSprite react=down defines every --px-* var', () => {
    const w = mount(CompanionSprite, { props: { stage: 'wilted', health: 0, react: 'down' } })
    assertEveryRectVarDefined(w.html(), 'CompanionSprite down')
  })

  it('CompanionSprite unknown stage defines every --px-* var', () => {
    const w = mount(CompanionSprite, { props: { stage: 'cactus', health: 80 } })
    assertEveryRectVarDefined(w.html(), 'CompanionSprite unknown')
  })

  it.each(['loading', 'empty', 'error'] as const)('StateBlock state=%s defines every --px-* var', (state) => {
    const w = mount(StateBlock, { props: { state, message: 'x', action: 'Thử lại' } })
    assertEveryRectVarDefined(w.html(), `StateBlock ${state}`)
  })
})
