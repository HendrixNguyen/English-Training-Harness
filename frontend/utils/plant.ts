import { DAILY_TARGET_SECONDS } from '~/utils/progress'

/** The DDL `pet_stage` enum (0001_init.up.sql). */
export const PLANT_STAGES = ['seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted'] as const
export type PlantStage = (typeof PLANT_STAGES)[number]

export type Tone = 'growth' | 'streak' | 'alert'

export function normalizeStage(stage: string | undefined | null): { stage: PlantStage, known: boolean } {
  if (stage && (PLANT_STAGES as readonly string[]).includes(stage)) return { stage: stage as PlantStage, known: true }
  return { stage: 'sprout', known: false }
}

/** design §3: 100–60 growth, 59–30 streak, 29–0 alert. */
export function healthTone(health: number): Tone {
  if (health >= 60) return 'growth'
  if (health >= 30) return 'streak'
  return 'alert'
}

/** CODEMAP `pet`: wilted at 0 health, otherwise by streak — the table `SaveTargetMet` writes (backend engine.go). */
export function stageForStreak(streak: number, health: number): PlantStage {
  if (health <= 0) return 'wilted'
  if (streak >= 14) return 'fruitful'
  if (streak >= 7) return 'flowering'
  if (streak >= 3) return 'sapling'
  return 'sprout'
}

/** Streak days that earn a line of their own in the bubble (design growth-moment §4). */
export const STREAK_MILESTONES = [3, 7, 14, 21, 28] as const

export interface SpeechInput {
  stage: string
  health: number
  targetMet: boolean
  /** GET /quests/daily accumulated_seconds; > 0 means the learner is mid-day. */
  accumulatedSeconds?: number
  streak?: number
  lastPracticedAt?: string | null
  now?: Date
}

const MET_LINE = 'Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿'

/** Whole days between an ISO timestamp and now; null when absent or unparsable. */
export function daysSince(iso: string | null | undefined, now: Date = new Date()): number | null {
  if (!iso) return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  return Math.max(0, Math.floor((now.getTime() - t) / 86_400_000))
}

/** One line, first match wins — design growth-moment §4. */
export function speechLine(o: SpeechInput): string {
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  if (o.targetMet) {
    const milestone = o.streak !== undefined && (STREAK_MILESTONES as readonly number[]).includes(o.streak)
    return milestone ? `${o.streak} ngày liên tiếp! ${MET_LINE}` : MET_LINE
  }
  const accumulated = o.accumulatedSeconds ?? 0
  if (accumulated > 0) return `Còn ${Math.max(1, Math.ceil((DAILY_TARGET_SECONDS - accumulated) / 60))} phút nữa thôi!`
  // daysSince floors elapsed ms/86_400_000 (unchanged — pages/revive.vue relies on that
  // semantics too), so a gap that reads as "2 calendar days" in the design's prose
  // (last practiced the day before yesterday) floors to 1 whole elapsed day, not 2.
  const missed = daysSince(o.lastPracticedAt, o.now)
  if (missed !== null && missed >= 1) return 'Hôm qua tớ nhớ bạn… Tưới 10 phút nhé?'
  if (o.health >= 60) return 'Tưới cho tớ 10 phút học đi!'
  if (o.health >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return 'Tớ sắp héo mất! Học một chút nhé?'
}
