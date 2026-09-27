import { computed, onScopeDispose, ref } from 'vue'
import type { GrowthDelta } from '~/stores/pet'

/** design retro-hub.md Addendum H3: `streak` is renamed `torch` — the kit has no `streak` token. */
export interface GrowthChipSpec { tone: 'growth' | 'torch', text: string }

/** Design growth-moment §2 timeline, in ms from the hub's first paint. */
export const MOMENT_MS = { chipsIn: 150, grow: 400, chipsOut: 1800, end: 2000 } as const

/** The chest-item chips a delta earns (design H3: kit register, no emoji in
 * the health chip; the streak chip carries its flame icon in markup, not
 * in this text). Health first, streak second; nothing for an unchanged value. */
export function chipsFor(d: GrowthDelta): GrowthChipSpec[] {
  const out: GrowthChipSpec[] = []
  if (d.healthTo > d.healthFrom) out.push({ tone: 'growth', text: `+${d.healthTo - d.healthFrom} HP` })
  else if (d.targetMetNow && d.healthTo >= 100) out.push({ tone: 'growth', text: 'HP đầy' })
  if (d.streakTo > d.streakFrom) out.push({ tone: 'torch', text: `x${d.streakTo}` })
  return out
}

/** design H3: replaces the v1 `PlantSvg` scale-tween `grow` prop — the sprite
 * plays a frame-swap reaction instead of any tween. */
export type GrowthReact = 'levelup' | 'hit' | 'idle'

/**
 * The hub's growth moment: before-values for one paint, then the store's
 * values, chips in at 150 ms, the companion reacts at 400 ms (design H3:
 * `levelup` when the stage changed, `hit` when the target was newly met and
 * the stage did not, `idle` otherwise), chips out at 1800 ms, react resets to
 * idle at 2000 ms. All motion is CSS behind prefers-reduced-motion; this
 * only schedules state.
 */
export function useGrowthMoment(ms: typeof MOMENT_MS = MOMENT_MS) {
  const delta = ref<GrowthDelta | null>(null)
  const painted = ref(false)
  const chipsVisible = ref(false)
  const pulse = ref(false)
  const react = ref<GrowthReact>('idle')
  const timers: ReturnType<typeof setTimeout>[] = []

  function start(d: GrowthDelta | null) {
    if (!d) return
    delta.value = d
    painted.value = false
    const flip = () => { painted.value = true }
    if (typeof requestAnimationFrame === 'function') requestAnimationFrame(flip)
    else flip()
    timers.push(
      setTimeout(() => { chipsVisible.value = true; pulse.value = d.streakTo > d.streakFrom }, ms.chipsIn),
      setTimeout(() => {
        react.value = d.stageTo !== d.stageFrom ? 'levelup' : (d.targetMetNow ? 'hit' : 'idle')
      }, ms.grow),
      setTimeout(() => { chipsVisible.value = false }, ms.chipsOut),
      setTimeout(() => { delta.value = null; react.value = 'idle'; pulse.value = false }, ms.end),
    )
  }

  onScopeDispose(() => timers.forEach(clearTimeout))

  return {
    start,
    active: computed(() => delta.value !== null),
    chips: computed(() => (delta.value && chipsVisible.value ? chipsFor(delta.value) : [])),
    react,
    pulse,
    displayHealth: (live: number) => (delta.value && !painted.value ? delta.value.healthFrom : live),
    displayStage: (live: string) => (delta.value && !painted.value ? delta.value.stageFrom : live),
  }
}
