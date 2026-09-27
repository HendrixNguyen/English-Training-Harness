---
type: bug
status: proposed
source: reviewer
run: _inbox
priority: low
---
# Fontsource per-subset CSS has no unicode-range, so all nine VT323 and Nunito faces claim every codepoint

## Why
`nuxt.config.ts` imports nine per-subset fontsource files (`@fontsource/vt323/{latin,latin-ext,vietnamese}-400.css` and the Nunito 400/700 equivalents). Those files ship **without** `unicode-range` (`grep -c unicode-range node_modules/@fontsource/vt323/vietnamese-400.css` → 0). The combined weight files (`@fontsource/vt323/400.css`, `@fontsource/nunito/400.css`, `700.css`) do carry the ranges (3, 5 and 5 hits). In the browser, every loaded face reports `unicodeRange` `U+0-10FFFF`. Rendering still works: faces are tried last-declared first, then earlier faces fill missing glyphs, and `document.fonts.check` passes for `ặỆữ`. The cost is that the browser can no longer pick the one subset it needs. Each face whose glyphs it tries gets downloaded, and glyph selection depends on import order, not on the ranges. The SW precaches all nine anyway, so this mainly costs the first, uncached load on a phone.

## Expected output
- Each `@font-face` in the built CSS carries its subset's `unicode-range`: import the combined weight files, or add the ranges. A Latin-only page then fetches only the latin face(s).
- `fonts.test.ts` asserts `unicode-range` on each built face. The 9 woff2 in `.output/public/_nuxt` and `sw.js` stay.

## Evidence
- Plan: `harness/plans/2026-09-26-retro-adventure-ui-mobile-first-16-bit-jrpg-restyle-with-a-n.md`, branch @ `2e0125c`, `frontend/nuxt.config.ts:19-27`.
- Browser: `[...document.fonts].map(f => f.unicodeRange)` returns `U+0-10FFFF` for all loaded VT323/Nunito faces on `/_kit`.
