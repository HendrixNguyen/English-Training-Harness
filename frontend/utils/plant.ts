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

/** Streak days that earn a line of their own in the bubble (design retro-hub.md Addendum H1, row 3). */
export const STREAK_MILESTONES = [3, 7, 14, 21, 28] as const

// --- streak-shield helpers, copied verbatim from
// origin/harness/2026-09-26-medium-pet-streak-shield-earned-by-target-days
// frontend/utils/plant.ts (design pet-streak-shield.md). The retro-hub plan
// binds the status-bar rack to a local `typeof shields === 'number'` guard
// instead of `pet.status.shields` until that branch merges into main — see
// the plan's "Order constraint" for the planned conflict resolution. ---

/** Migration 0004 CHECK and the award period (backend pet engine). */
export const MAX_SHIELDS = 2
export const SHIELD_EVERY_DAYS = 7

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

export interface SpeechInput {
  stage: string
  health: number
  targetMet: boolean
  /** GET /quests/daily accumulated_seconds; > 0 means the learner is mid-day. */
  accumulatedSeconds?: number
  streak?: number
  lastPracticedAt?: string | null
  now?: Date
  /** GET /pet/status shields (pet-streak-shield.md) — not a number until that branch merges (design amend H2); row 2 is skipped entirely until then. */
  shields?: number
  /** GET /quests/daily day_number — needed to tell "first day, zero minutes" (row 6) from any later day at zero minutes (falls through to the health rows). */
  dayNumber?: number
  /** The companion's name — used only by row 6 (design Addendum H1: "the name goes on the speaker tab, not into the line"); never hard-coded — falls back to "Mầm Non" only for a blank/absent value. */
  name?: string
}

/** Whole days between an ISO timestamp and now; null when absent or unparsable. Unchanged — `pages/revive.vue` keeps this exact semantics. */
export function daysSince(iso: string | null | undefined, now: Date = new Date()): number | null {
  if (!iso) return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  return Math.max(0, Math.floor((now.getTime() - t) / 86_400_000))
}

/**
 * One line, first match wins — design retro-hub.md Addendum H1 (replaces
 * the v1 growth-moment §4 table: kit register "tớ"/"cậu", no emoji, at
 * most one "!", and the name confined to row 6).
 */
export function speechLine(o: SpeechInput): string {
  if (o.stage === 'wilted' || o.health <= 0) return '…' // row 1: the revive band speaks instead

  const streak = o.streak ?? 0
  const accumulated = o.accumulatedSeconds ?? 0
  const now = o.now ?? new Date()

  // row 2: a shield was earned today. `targetMet` is required in addition to
  // the shield branch's own "streak % 7 == 0" rule — without it, the streak
  // stays a multiple of 7 all the next day too (until that day's target is
  // met), so the earn line would repeat every morning instead of appearing
  // once, on the day it was actually earned.
  if (typeof o.shields === 'number' && o.targetMet && streak > 0 && streak % SHIELD_EVERY_DAYS === 0 && o.shields > 0) {
    return `Tròn ${streak} ngày liên tiếp! Cậu có khiên giữ chuỗi ngày rồi.`
  }

  // rows 3-4: target met today.
  if (o.targetMet) {
    const milestone = (STREAK_MILESTONES as readonly number[]).includes(streak)
    return milestone
      ? `${streak} ngày liên tiếp! Hôm nay tớ đủ nước rồi, cảm ơn cậu.`
      : 'Hôm nay tớ đủ nước rồi, cảm ơn cậu.'
  }

  // row 5: mid-day, some progress already banked.
  if (accumulated > 0) {
    const minutesLeft = Math.max(1, Math.ceil((DAILY_TARGET_SECONDS - accumulated) / 60))
    return `Còn ${minutesLeft} phút nữa là xong phòng hôm nay!`
  }

  // row 6: day 1, nothing studied yet, no streak, never practiced before.
  if (o.dayNumber === 1 && accumulated === 0 && streak === 0 && (o.lastPracticedAt ?? null) === null) {
    const name = o.name?.trim() || 'Mầm Non'
    return `Chào cậu. Tớ là ${name} — cùng vào phòng đầu tiên nhé?`
  }

  // row 7: missed — the local calendar date of the last practice is 2 or
  // more days behind today's (design: "the day before yesterday" or
  // earlier), not a 24h-span floor — 25h since practice is still "yesterday".
  const lastDate = o.lastPracticedAt ? new Date(o.lastPracticedAt) : null
  const missedDays = lastDate && !Number.isNaN(lastDate.getTime())
    ? daysBetweenDates(localDateYmd(lastDate), localDateYmd(now))
    : null
  if (missedDays !== null && missedDays >= 2) return 'Hôm qua tớ nhớ cậu… Tưới 10 phút nhé?'

  // rows 8-10: health bands.
  if (o.health >= 60) return 'Tưới cho tớ 10 phút đi, cậu.'
  if (o.health >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return 'Tớ sắp héo mất. Học một chút nhé?'
}
