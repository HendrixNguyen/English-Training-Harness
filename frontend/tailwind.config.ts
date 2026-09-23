import type { Config } from 'tailwindcss'

/** Frontend spec §6.1 palette plus derived neutrals (design §1). */
export const tokens = {
  growth: '#10B981',
  streak: '#F59E0B',
  alert: '#EF4444',
  ink: '#1E293B',
  paper: '#F8FAFC',
  'paper-dark': '#0F172A',
  mute: '#64748B',
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
        display: ['"Fraunces Variable"', 'Georgia', 'serif'],
        body: ['"Source Sans 3"', 'system-ui', 'sans-serif'],
      },
      borderRadius: { card: '16px', btn: '12px' },
    },
  },
} satisfies Config
