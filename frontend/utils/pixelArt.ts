/**
 * Retro kit pixel art data (design §4, harness/designs/retro-kit.md).
 * `PixelArt.vue` renders any of these as an SVG `<rect>` grid; the palette
 * chars below are the only ones allowed in a `COMPANION`/`GLYPHS` row.
 */

/** char → token role name (`tailwind.config.ts` `tokens` key). */
export const PALETTE: Record<string, string> = {
  k: 'ground-0',
  g: 'growth',
  G: 'growth-deep',
  i: 'ink-0',
  t: 'torch',
  T: 'torch-deep',
  e: 'ember',
  E: 'ember-deep',
  d: 'line-dim',
  l: 'line-lit',
}

/** Build one square row of `width` chars from column-anchored segments over
 * a blank ('.') canvas — avoids manual dot-counting when authoring rows. */
function buildRow(width: number, ...placements: [col: number, seg: string][]): string {
  const chars: string[] = Array.from({ length: width }, () => '.')
  for (const [col, seg] of placements) {
    for (let i = 0; i < seg.length; i++) chars[col + i] = seg[i]
  }
  return chars.join('')
}

function blankRows(count: number, width: number): string[] {
  return Array.from({ length: count }, () => '.'.repeat(width))
}

const W = 32

/**
 * Rows 22-31: the pot. Identical across every stage (design §4) — the
 * face (eyes, catchlights, smile) lives here, on the pot, never on the
 * plant, so every stage reads with the same expression.
 */
const POT: readonly string[] = [
  '........kkkkkkkkkkkkkkkk........', // 22 rim outline
  '.......kttttttttttttttttk.......', // 23 rim light band
  '.......kTTTTTTTTTTTTTTTTk.......', // 24 rim shade
  '........kTTTTTTTTTTTTTTk........', // 25 body
  '........kTTTkkTTTTkkTTTk........', // 26 eyes (top)
  '........kTTTkiTTTTkiTTTk........', // 27 eyes (catchlight at 13,27 / 19,27)
  '........kTTTTkTTTTkTTTTk........', // 28 smile corners (13,28) (18,28)
  '.........kTTTTkkkkTTTTk.........', // 29 smile (cols 14-17)
  '.........kTTTTTTTTTTTTk.........', // 30 body taper
  '.........kkkkkkkkkkkkkk.........', // 31 base
]

function withPot(top21: string[]): string[] {
  if (top21.length !== 22) throw new Error(`companion top must be 22 rows, got ${top21.length}`)
  return [...top21, ...POT]
}

// --- seed: no stem; a 6x3 dome on the soil (rows 19-21) -------------------
const seedTop = [
  ...blankRows(19, W),
  buildRow(W, [13, 'kggggk']), // 19 dome top
  buildRow(W, [13, 'kigggk']), // 20 dome body, glint at (14,20)
  buildRow(W, [13, 'kkkkkk']), // 21 soil outline
]

// --- sprout: verbatim from the design §4 sketch (rows 9-21) --------------
const sproutTop = [
  ...blankRows(9, W),
  '...................kkk..........',
  '..................kgggk.........',
  '........kkk......kggigk.........',
  '.......kgggk.....kgggGk.........',
  '......kgggigk....kgGGk..........',
  '......kggggGk.kgGGGk............',
  '.......kgggGGkkgGk..............',
  '.........kkkkkkgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
  '..............kgGk..............',
]

// --- sapling: 13-row stem (cols 15-16), four alternating leaves, a 3-row
// bud at the tip (rows 6-8, cols 14-17) -----------------------------------
const saplingTop = [
  ...blankRows(6, W),
  buildRow(W, [15, 'gg']), // 6 bud tip
  buildRow(W, [14, 'kggk']), // 7 bud shoulder
  buildRow(W, [14, 'kGGk'], [19, 'kgk']), // 8 bud base + right leaf 1 tip
  buildRow(W, [14, 'kgGk'], [18, 'kgigk']), // 9 stem + right leaf 1 mid
  buildRow(W, [14, 'kgGk'], [18, 'kGGGk']), // 10 stem + right leaf 1 base
  buildRow(W, [14, 'kgGk'], [10, 'kgk']), // 11 stem + left leaf 1 tip
  buildRow(W, [14, 'kgGk'], [9, 'kgigk']), // 12 stem + left leaf 1 mid
  buildRow(W, [14, 'kgGk'], [9, 'kGGGk']), // 13 stem + left leaf 1 base
  buildRow(W, [14, 'kgGk'], [19, 'kgk']), // 14 stem + right leaf 2 tip
  buildRow(W, [14, 'kgGk'], [18, 'kgigk']), // 15 stem + right leaf 2 mid
  buildRow(W, [14, 'kgGk'], [18, 'kGGGk']), // 16 stem + right leaf 2 base
  buildRow(W, [14, 'kgGk'], [10, 'kgk']), // 17 stem + left leaf 2 tip
  buildRow(W, [14, 'kgGk'], [9, 'kgigk']), // 18 stem + left leaf 2 mid
  buildRow(W, [14, 'kgGk'], [9, 'kGGGk']), // 19 stem + left leaf 2 base
  buildRow(W, [14, 'kgGk']), // 20 stem
  buildRow(W, [14, 'kgGk']), // 21 stem
]

// --- flowering: sapling + three 3x3 flowers (i petals, t centre) at the
// tip and on the two upper leaves' outer tips -----------------------------
const floweringTop = [
  ...blankRows(4, W),
  buildRow(W, [14, 'i.i']), // 4 tip flower top petals
  buildRow(W, [14, '.t.']), // 5 tip flower centre
  buildRow(W, [14, 'i.i']), // 6 tip flower bottom petals
  buildRow(W, [14, '.g.']), // 7 neck
  buildRow(W, [14, 'kggk'], [19, 'iti']), // 8 stem shoulder + right leaf 1 flower
  buildRow(W, [14, 'kgGk'], [18, 'kgigk']), // 9 stem + right leaf 1 mid
  buildRow(W, [14, 'kgGk'], [18, 'kGGGk']), // 10 stem + right leaf 1 base
  buildRow(W, [14, 'kgGk'], [10, 'iti']), // 11 stem + left leaf 1 flower
  buildRow(W, [14, 'kgGk'], [9, 'kgigk']), // 12 stem + left leaf 1 mid
  buildRow(W, [14, 'kgGk'], [9, 'kGGGk']), // 13 stem + left leaf 1 base
  buildRow(W, [14, 'kgGk'], [19, 'kgk']), // 14 stem + right leaf 2 tip (plain)
  buildRow(W, [14, 'kgGk'], [18, 'kgigk']), // 15 stem + right leaf 2 mid
  buildRow(W, [14, 'kgGk'], [18, 'kGGGk']), // 16 stem + right leaf 2 base
  buildRow(W, [14, 'kgGk'], [10, 'kgk']), // 17 stem + left leaf 2 tip (plain)
  buildRow(W, [14, 'kgGk'], [9, 'kgigk']), // 18 stem + left leaf 2 mid
  buildRow(W, [14, 'kgGk'], [9, 'kGGGk']), // 19 stem + left leaf 2 base
  buildRow(W, [14, 'kgGk']), // 20 stem
  buildRow(W, [14, 'kgGk']), // 21 stem
]

// --- fruitful: flowering with leaves one px wider; flowers become 3x3
// fruits (t with a T bottom row); a fourth fruit on the lower right leaf --
const fruitfulTop = [
  ...blankRows(3, W),
  buildRow(W, [14, 'ttt']), // 3 tip fruit top
  buildRow(W, [14, 'ttt']), // 4 tip fruit mid
  buildRow(W, [14, 'TTT']), // 5 tip fruit bottom (deeper)
  buildRow(W, [14, '.g.']), // 6 neck
  buildRow(W, [14, 'kggk'], [19, 'ttt']), // 7 stem shoulder + right leaf 1 fruit top
  buildRow(W, [14, 'kGGk'], [19, 'TTT']), // 8 stem + right leaf 1 fruit bottom
  buildRow(W, [14, 'kgGk'], [18, 'kggigk']), // 9 stem + right leaf 1 base (widened)
  buildRow(W, [14, 'kgGk'], [18, 'kGGGGk']), // 10 stem + right leaf 1 base
  buildRow(W, [14, 'kgGk'], [10, 'ttt']), // 11 stem + left leaf 1 fruit top
  buildRow(W, [14, 'kgGk'], [10, 'TTT']), // 12 stem + left leaf 1 fruit bottom
  buildRow(W, [14, 'kgGk'], [8, 'kggigk']), // 13 stem + left leaf 1 base (widened)
  buildRow(W, [14, 'kgGk'], [8, 'kGGGGk']), // 14 stem + left leaf 1 base
  buildRow(W, [14, 'kgGk'], [19, 'ttt']), // 15 stem + right leaf 2 fruit top (fourth fruit)
  buildRow(W, [14, 'kgGk'], [19, 'TTT']), // 16 stem + right leaf 2 fruit bottom
  buildRow(W, [14, 'kgGk'], [18, 'kggigk']), // 17 stem + right leaf 2 base (widened)
  buildRow(W, [14, 'kgGk'], [18, 'kGGGGk']), // 18 stem + right leaf 2 base
  buildRow(W, [14, 'kgGk'], [10, 'kgk']), // 19 stem + left leaf 2 (plain)
  buildRow(W, [14, 'kgGk'], [8, 'kggigk']), // 20 stem + left leaf 2 base (widened)
  buildRow(W, [14, 'kgGk']), // 21 stem
]

// --- wilted: sapling's shape recoloured g->e, G->E; the top six stem rows
// bend one column right per row; small down-pointing leaves; no bud -------
const wiltedTop = [
  ...blankRows(9, W),
  buildRow(W, [14, 'keEk']), // 9 bend
  buildRow(W, [15, 'keEk']), // 10 bend
  buildRow(W, [16, 'keEk']), // 11 bend
  buildRow(W, [17, 'keEk']), // 12 bend
  buildRow(W, [18, 'keEk']), // 13 bend
  buildRow(W, [19, 'keEk']), // 14 bend settles
  buildRow(W, [19, 'keEk']), // 15 stem
  buildRow(W, [19, 'keEk']), // 16 stem
  buildRow(W, [19, 'keEk'], [23, 'keeek']), // 17 stem + right leaf base
  buildRow(W, [19, 'keEk'], [24, 'kek']), // 18 stem + right leaf tip (down)
  buildRow(W, [19, 'keEk']), // 19 stem
  buildRow(W, [19, 'keEk'], [12, 'keeek']), // 20 stem + left leaf base
  buildRow(W, [19, 'keEk'], [13, 'kek']), // 21 stem + left leaf tip (down)
]

export const PLANT_STAGE_ORDER = ['seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted'] as const

export const COMPANION: Record<(typeof PLANT_STAGE_ORDER)[number], string[]> = {
  seed: withPot(seedTop),
  sprout: withPot(sproutTop),
  sapling: withPot(saplingTop),
  flowering: withPot(floweringTop),
  fruitful: withPot(fruitfulTop),
  wilted: withPot(wiltedTop),
}

// --- glyphs: 16x16 icons (13-row `cursor` is 8x8) -------------------------

function canvas(size: number): string[][] {
  return Array.from({ length: size }, () => Array.from({ length: size }, () => '.'))
}
function toRows(grid: string[][]): string[] {
  return grid.map(r => r.join(''))
}
function setPx(grid: string[][], x: number, y: number, ch: string) {
  grid[y][x] = ch
}
/** An outlined rectangle from (x0,y0) to (x1,y1) inclusive. */
function fillRect(grid: string[][], x0: number, y0: number, x1: number, y1: number, outline: string, fill: string) {
  for (let y = y0; y <= y1; y++) {
    for (let x = x0; x <= x1; x++) {
      const edge = x === x0 || x === x1 || y === y0 || y === y1
      setPx(grid, x, y, edge ? outline : fill)
    }
  }
}

function bookGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 2, 3, 13, 12, 'k', 'i')
  for (let y = 3; y <= 12; y++) { setPx(g, 7, y, 'k'); setPx(g, 8, y, 'k') }
  return toRows(g)
}
function scrollGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 3, 4, 12, 11, 'k', 'i')
  for (let y = 4; y <= 11; y++) { setPx(g, 2, y, 'd'); setPx(g, 13, y, 'd') }
  return toRows(g)
}
function swordGlyph(): string[] {
  const g = canvas(16)
  for (let y = 1; y <= 9; y++) { setPx(g, 7, y, 'l'); setPx(g, 8, y, 'l') }
  for (let x = 4; x <= 11; x++) setPx(g, x, 10, 'T')
  for (let y = 11; y <= 14; y++) { setPx(g, 7, y, 'T'); setPx(g, 8, y, 'T') }
  setPx(g, 7, 15, 'k'); setPx(g, 8, 15, 'k')
  return toRows(g)
}
function flameGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 4, 5, 11, 13, 'T', 't')
  setPx(g, 7, 4, 't'); setPx(g, 8, 4, 't')
  setPx(g, 7, 3, 't'); setPx(g, 8, 3, 't')
  return toRows(g)
}
function shieldGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 3, 2, 12, 11, 'l', 'g')
  for (let x = 5; x <= 10; x++) setPx(g, x, 12, 'l')
  setPx(g, 6, 13, 'l'); setPx(g, 7, 13, 'l'); setPx(g, 8, 13, 'l'); setPx(g, 9, 13, 'l')
  setPx(g, 7, 14, 'l'); setPx(g, 8, 14, 'l')
  return toRows(g)
}
function starGlyph(): string[] {
  const g = canvas(16)
  for (let y = 2; y <= 13; y++) { setPx(g, 7, y, 't'); setPx(g, 8, y, 't') }
  for (let x = 2; x <= 13; x++) { setPx(g, x, 7, 't'); setPx(g, x, 8, 't') }
  setPx(g, 7, 7, 'T'); setPx(g, 8, 7, 'T'); setPx(g, 7, 8, 'T'); setPx(g, 8, 8, 'T')
  return toRows(g)
}
function padlockGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 5, 2, 10, 7, 'd', '.')
  fillRect(g, 3, 7, 12, 14, 'k', 'd')
  setPx(g, 7, 10, 'k'); setPx(g, 8, 10, 'k'); setPx(g, 7, 11, 'k')
  return toRows(g)
}
function ringGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 4, 4, 11, 11, 'd', '.')
  return toRows(g)
}
function chestClosedGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 2, 3, 13, 13, 'T', 't')
  for (let x = 2; x <= 13; x++) setPx(g, x, 7, 'k')
  setPx(g, 7, 8, 'k'); setPx(g, 8, 8, 'k')
  return toRows(g)
}
function chestOpenGlyph(): string[] {
  const g = canvas(16)
  fillRect(g, 2, 6, 13, 13, 'T', 't')
  fillRect(g, 2, 1, 13, 4, 'T', 't')
  setPx(g, 7, 9, 'i'); setPx(g, 9, 8, 'i')
  return toRows(g)
}
function checkGlyph(): string[] {
  const g = canvas(16)
  const pts: [number, number][] = [[4, 8], [5, 9], [6, 10], [7, 11], [8, 10], [9, 8], [10, 6], [11, 4], [12, 2]]
  for (const [x, y] of pts) setPx(g, x, y, 'g')
  return toRows(g)
}
function crossGlyph(): string[] {
  const g = canvas(16)
  for (let i = 3; i <= 12; i++) { setPx(g, i, i, 'e'); setPx(g, i, 15 - i, 'e') }
  return toRows(g)
}
function cursorGlyph(): string[] {
  const g = canvas(8)
  setPx(g, 1, 1, 'l')
  setPx(g, 1, 2, 'l'); setPx(g, 2, 2, 'l')
  setPx(g, 1, 3, 'l'); setPx(g, 2, 3, 'l'); setPx(g, 3, 3, 'l')
  setPx(g, 1, 4, 'l'); setPx(g, 2, 4, 'l'); setPx(g, 3, 4, 'l')
  setPx(g, 1, 5, 'l'); setPx(g, 2, 5, 'l')
  setPx(g, 1, 6, 'l')
  return toRows(g)
}

export const GLYPHS = {
  book: bookGlyph(),
  scroll: scrollGlyph(),
  sword: swordGlyph(),
  flame: flameGlyph(),
  shield: shieldGlyph(),
  star: starGlyph(),
  padlock: padlockGlyph(),
  ring: ringGlyph(),
  chestClosed: chestClosedGlyph(),
  chestOpen: chestOpenGlyph(),
  check: checkGlyph(),
  cross: crossGlyph(),
  cursor: cursorGlyph(),
} as const
