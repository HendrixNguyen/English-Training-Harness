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

/** Migration 0005 CHECK and the award period (backend pet engine). */
export const MAX_SHIELDS = 2
export const SHIELD_EVERY_DAYS = 7

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
  /** Replaces "tớ" in the two lines that address the plant by name (design plant-name §2); blank keeps every line as before. */
  name?: string
  /** GET /pet/status shields (0–2, migration 0005); undefined treated as 0 — no earn line without it. */
  shields?: number
}

const metLine = (name: string) => `Cảm ơn bạn, hôm nay ${name} đủ nước rồi 🌿`

/** Whole days between an ISO timestamp and now; null when absent or unparsable. */
export function daysSince(iso: string | null | undefined, now: Date = new Date()): number | null {
  if (!iso) return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  return Math.max(0, Math.floor((now.getTime() - t) / 86_400_000))
}

/** The device's local calendar date as YYYY-MM-DD (local getters — toISOString would give the UTC day). */
export function localDateYmd(now: Date = new Date()): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}-${p(now.getMonth() + 1)}-${p(now.getDate())}`
}

function parseYmd(s: string | null | undefined): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s ?? '')
  return m ? Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : null
}

/** Calendar days from one YYYY-MM-DD to another (negative when `to` is earlier); null when either is malformed. Pure arithmetic — no timezone. */
export function daysBetweenDates(from: string | null | undefined, to: string): number | null {
  const a = parseYmd(from)
  const b = parseYmd(to)
  return a === null || b === null ? null : Math.round((b - a) / 86_400_000)
}

/** design §4.1: the caption under the shield rack while the spend is 0–7 days old. */
export function shieldSpentLine(lastUsedOn: string | null | undefined, today: string = localDateYmd()): string | null {
  const d = daysBetweenDates(lastUsedOn, today)
  if (d === null || d < 0 || d > SHIELD_EVERY_DAYS) return null
  const [, mm, dd] = (lastUsedOn as string).split('-')
  return `Khiên đã đỡ cho ngày ${dd}/${mm}.`
}

/** One line, first match wins — design growth-moment §4. */
export function speechLine(o: SpeechInput): string {
  const name = o.name?.trim() || 'tớ'
  const Name = name === 'tớ' ? 'Tớ' : name
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  if (o.targetMet) {
    const streak = o.streak ?? 0
    // design pet-streak-shield §4.2 (corrected 2026-09-27): only on the day the milestone was reached
    if (streak > 0 && streak % SHIELD_EVERY_DAYS === 0 && (o.shields ?? 0) > 0) return 'Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️'
    const milestone = o.streak !== undefined && (STREAK_MILESTONES as readonly number[]).includes(o.streak)
    return milestone ? `${o.streak} ngày liên tiếp! ${metLine(name)}` : metLine(name)
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
  return `${Name} sắp héo mất! Học một chút nhé?`
}
