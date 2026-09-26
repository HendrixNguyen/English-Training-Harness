import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import tailwindConfig from '~/tailwind.config'

const root = resolve(process.cwd())
const packageJson = JSON.parse(readFileSync(resolve(root, 'package.json'), 'utf-8'))
const nuxtConfigSource = readFileSync(resolve(root, 'nuxt.config.ts'), 'utf-8')

describe('retro kit fonts (design §2)', () => {
  it('package.json carries the two fontsource packages and drops the v1 faces', () => {
    const deps = { ...packageJson.dependencies, ...packageJson.devDependencies }
    expect(deps['@fontsource/vt323']).toBeTruthy()
    expect(deps['@fontsource/nunito']).toBeTruthy()
    expect(deps['@fontsource-variable/fraunces']).toBeUndefined()
    expect(deps['@fontsource/source-sans-3']).toBeUndefined()
  })

  it('nuxt.config.ts imports the Vietnamese subsets for both faces', () => {
    expect(nuxtConfigSource).toContain('vt323/vietnamese-400.css')
    expect(nuxtConfigSource).toContain('nunito/vietnamese-400.css')
    expect(nuxtConfigSource).toContain('nunito/vietnamese-700.css')
  })

  it('tailwind.config.ts registers VT323/Nunito as the two faces', () => {
    const fontFamily = tailwindConfig.theme.extend.fontFamily as Record<string, string[]>
    expect(fontFamily.display[0]).toBe('VT323')
    expect(fontFamily.body[0]).toBe('Nunito')
  })
})
