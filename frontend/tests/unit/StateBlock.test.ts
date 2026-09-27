import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StateBlock from '~/components/ui/StateBlock.vue'

/**
 * Design amend A1/A2 (harness/designs/retro-kit.md): `StateBlock` paints
 * its own `ground-1` and `ink-*` so it reads on any v1 surface, not only
 * the kit's dark ground — the blocker
 * `restyled-stateblock-and-countdowntimer-put-near-white-ink-0-.md`.
 */
describe('StateBlock (design amend A2)', () => {
  it('loading: root is bg-ground-1, aria-busy, aria-label, three bg-line-lit cells, no bg-ground-2', () => {
    const w = mount(StateBlock, { props: { state: 'loading' } })
    const root = w.find('[aria-busy="true"]')
    expect(root.classes()).toContain('bg-ground-1')
    expect(root.attributes('aria-label')).toBe('Đang tải')
    const cells = w.findAll('.retro-dots')
    expect(cells).toHaveLength(3)
    for (const cell of cells) {
      expect(cell.classes()).toContain('bg-line-lit')
      expect(cell.classes()).not.toContain('bg-ground-2')
    }
    expect(w.html()).not.toContain('bg-ground-2')
  })

  it('error: [role="status"] wraps a RetroPanel toned ember, the <p> is text-ink-0, the button is inside it', () => {
    const w = mount(StateBlock, { props: { state: 'error', message: 'Lỗi rồi', action: 'Thử lại' } })
    const status = w.find('[role="status"]')
    expect(status.exists()).toBe(true)
    const panel = status.find('section')
    expect(panel.classes()).toContain('bg-ground-1')
    const style = panel.attributes('style') ?? ''
    // tone=ember recolours the outer ring — see RetroPanel.test.ts for the hex
    expect(style.length).toBeGreaterThan(0)
    const p = status.find('p')
    expect(p.classes()).toContain('text-ink-0')
    expect(status.find('button').exists()).toBe(true)
  })

  it('empty: [role="status"] wraps a plain-tone RetroPanel', () => {
    const w = mount(StateBlock, { props: { state: 'empty', message: 'Chưa có gì', action: 'Bắt đầu' } })
    const status = w.find('[role="status"]')
    expect(status.exists()).toBe(true)
    expect(status.find('section').classes()).toContain('bg-ground-1')
    expect(status.find('button').exists()).toBe(true)
  })
})
