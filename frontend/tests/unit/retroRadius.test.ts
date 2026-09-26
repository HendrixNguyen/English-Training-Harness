import { readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const DIR = resolve(process.cwd(), 'components/retro')
const FORBIDDEN = /rounded-(?!none|sm)|blur|bg-gradient|scale-|ease-|dark:/

describe('retro kit radius/motion guard (design §1, §5)', () => {
  const files = readdirSync(DIR).filter(f => f.endsWith('.vue'))

  it('found the kit components to check', () => {
    expect(files.length).toBeGreaterThan(0)
  })

  it.each(files)('%s uses no radius above 2px, no blur, no gradient, no scale tween, no easing curve, no dark: variant', (file) => {
    const content = readFileSync(resolve(DIR, file), 'utf-8')
    const hit = content.match(FORBIDDEN)
    expect(hit, hit ? `found "${hit[0]}" in ${file}` : undefined).toBeNull()
  })
})
