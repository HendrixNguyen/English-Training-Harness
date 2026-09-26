import type { Config } from 'tailwindcss'

/**
 * UI kit v2 palette (design §1; harness/UI-KIT.md). `tokens` stays a flat
 * `Record<string, string>` — the contrast test iterates it and the aliases
 * below reference it by key, not by a nested path.
 *
 * v1-only tokens (`ink`, `paper`, `paper-dark`, `borderRadius.card/btn`) stay
 * until plan 6 deletes them along with the `html` rule in `main.css`.
 */
const v2 = {
  'ground-0': '#0B0A1F',
  'ground-1': '#151434',
  'ground-2': '#1F1D4A',
  'line-lit': '#C9C4F4',
  'line-dim': '#3B3A78',
  'ink-0': '#F4F1FF',
  'ink-1': '#B7B3DC',
  'ink-2': '#8783B5',
  growth: '#3DE1B0',
  'growth-deep': '#178A69',
  torch: '#F2A83B',
  'torch-deep': '#B8641E',
  ember: '#FF5A4E',
  'ember-deep': '#B3261E',
} as const

const v1Only = {
  ink: '#1E293B',
  paper: '#F8FAFC',
  'paper-dark': '#0F172A',
} as const

export const tokens = {
  ...v2,
  ...v1Only,
  // v1 aliases (design §1): kept until the last screen migrates, then
  // deleted in plan 6. Same hex as their v2 target so v1 components pick up
  // the new hue without a rename.
  streak: v2.torch,
  alert: v2.ember,
  mute: v2['ink-2'],
} as const

export default {
  // @nuxtjs/tailwindcss merges its own auto-detected content globs in
  // separately (module.mjs resolveContentConfig) — this file only needs
  // `content` to satisfy the resolved Tailwind 3.4 Config type.
  content: [],
  darkMode: 'media',
  theme: {
    extend: {
      colors: tokens,
      fontFamily: {
        display: ['VT323', 'monospace'],
        body: ['Nunito', 'system-ui', 'sans-serif'],
      },
      borderRadius: { card: '16px', btn: '12px' },
    },
  },
} satisfies Config
