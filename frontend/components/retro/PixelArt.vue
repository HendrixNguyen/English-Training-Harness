<script setup lang="ts">
import { computed } from 'vue'

/**
 * One renderer, one test, for every sprite/icon/glyph in the kit (design
 * §4). `rows` are N strings of N chars from `utils/pixelArt.ts`; `.` is
 * transparent and emits no `<rect>`. Fill colours come from CSS variables
 * (`--px-<char>`) set on the `<svg>` from `palette`, so a parent recolours
 * by overriding a variable (see `CompanionSprite`'s health tint).
 */
const props = withDefaults(defineProps<{
  rows: string[]
  palette: Record<string, string>
  size: number
  label?: string
  /** Internal, additive to the design's documented prop list: lets
   * `CompanionSprite` crop to the 16×16 face region without PixelArt
   * knowing about companions. Defaults to the full `0 0 N N` box. */
  viewBox?: string
}>(), { label: undefined, viewBox: undefined })

const gridSize = computed(() => props.rows.length)
const snappedSize = computed(() => {
  const n = gridSize.value
  return Math.max(n, Math.floor(props.size / n) * n)
})
const box = computed(() => props.viewBox ?? `0 0 ${gridSize.value} ${gridSize.value}`)

const cssVars = computed(() => {
  const out: Record<string, string> = {}
  for (const [char, color] of Object.entries(props.palette)) out[`--px-${char}`] = color
  return out
})

/** Run-length merge each row into one `<rect>` per horizontal run of the
 * same non-'.' char. */
const rects = computed(() => {
  const out: { x: number, y: number, width: number, fill: string }[] = []
  props.rows.forEach((row, y) => {
    let x = 0
    while (x < row.length) {
      const ch = row[x]
      if (ch === '.') { x += 1; continue }
      let width = 1
      while (x + width < row.length && row[x + width] === ch) width += 1
      out.push({ x, y, width, fill: `var(--px-${ch})` })
      x += width
    }
  })
  return out
})
</script>

<template>
  <svg
    :width="snappedSize"
    :height="snappedSize"
    :viewBox="box"
    class="retro-pixel"
    shape-rendering="crispEdges"
    :role="label ? 'img' : undefined"
    :aria-label="label"
    :aria-hidden="label ? undefined : 'true'"
    :style="cssVars"
  >
    <rect v-for="(r, i) in rects" :key="i" :x="r.x" :y="r.y" :width="r.width" height="1" :fill="r.fill" />
  </svg>
</template>
