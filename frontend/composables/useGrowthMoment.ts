import { computed, onScopeDispose, ref } from 'vue'
import { useReducedMotion } from '~/composables/useReducedMotion'
import type { GrowthDelta } from '~/stores/pet'

export interface GrowthChipSpec { tone: 'growth' | 'streak', text: string }

/** Design growth-moment §2 timeline, in ms from the hub's first paint. */
export const MOMENT_MS = { chipsIn: 150, grow: 400, chipsOut: 1800, end: 2000 } as const

/** The chips a delta earns (design §3). Health first, streak second; nothing for an unchanged value. */
export function chipsFor(d: GrowthDelta): GrowthChipSpec[] {
  const out: GrowthChipSpec[] = []
  if (d.healthTo > d.healthFrom) out.push({ tone: 'growth', text: `+${d.healthTo - d.healthFrom} máu` })
  else if (d.targetMetNow && d.healthTo >= 100) out.push({ tone: 'growth', text: 'Máu đầy' })
  if (d.streakTo > d.streakFrom) out.push({ tone: 'streak', text: `🔥 ${d.streakTo} ngày` })
  return out
}

/**
 * The hub's growth moment: before-values for one paint, then the store's
 * values, chips in at 150 ms, one stepped sprite reaction at 400 ms (hit for unchanged stage,
 * levelup for a changed stage), chips out at 1800 ms.
 * All motion is CSS behind prefers-reduced-motion; this only schedules classes.
 */
export function useGrowthMoment(ms: typeof MOMENT_MS = MOMENT_MS) {
  const reduced = useReducedMotion()
  const delta = ref<GrowthDelta | null>(null)
  const painted = ref(false)
  const chipsVisible = ref(false)
  const grow = ref(false)
  const pulse = ref(false)
  const reaction = ref<'idle' | 'hit' | 'levelup'>('idle')
  const timers: ReturnType<typeof setTimeout>[] = []

  function start(d: GrowthDelta | null) {
    if (!d) return
    delta.value = d
    painted.value = reduced.value
    chipsVisible.value = reduced.value
    const flip = () => { painted.value = true }
    if (typeof requestAnimationFrame === 'function') requestAnimationFrame(flip)
    else flip()
    timers.push(
      setTimeout(() => { chipsVisible.value = true; pulse.value = d.streakTo > d.streakFrom }, ms.chipsIn),
      setTimeout(() => { grow.value = d.targetMetNow && d.stageTo === d.stageFrom; reaction.value = d.stageTo !== d.stageFrom ? 'levelup' : d.targetMetNow ? 'hit' : 'idle' }, ms.grow),
      setTimeout(() => { reaction.value = 'idle' }, ms.grow + 300),
      setTimeout(() => { chipsVisible.value = false }, reduced.value ? ms.end : ms.chipsOut),
      setTimeout(() => { delta.value = null; grow.value = false; pulse.value = false }, ms.end),
    )
  }

  onScopeDispose(() => timers.forEach(clearTimeout))

  return {
    start,
    active: computed(() => delta.value !== null),
    chips: computed(() => (delta.value && chipsVisible.value ? chipsFor(delta.value) : [])),
    grow,
    reaction,
    pulse,
    displayHealth: (live: number) => (delta.value && !painted.value && !reduced.value ? delta.value.healthFrom : live),
    displayStage: (live: string) => (delta.value && !painted.value && !reduced.value ? delta.value.stageFrom : live),
  }
}
