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

/** Migration 0004 CHECK and the award period (backend pet engine). */
export const MAX_SHIELDS = 2
export const SHIELD_EVERY_DAYS = 7

export function speechLine(o: { stage: string, health: number, targetMet: boolean, streak?: number, shields?: number }): string {
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  const streak = o.streak ?? 0
  // design §4.2: the earn line states the milestone and that a shield is held — true on day 14 and on day 21 at the cap alike.
  if (streak > 0 && streak % SHIELD_EVERY_DAYS === 0 && (o.shields ?? 0) > 0) return 'Tròn 7 ngày liên tiếp! Bạn có khiên bảo vệ streak rồi 🛡️'
  if (o.targetMet) return 'Cảm ơn bạn, hôm nay tớ đủ nước rồi 🌿'
  if (o.health >= 60) return 'Tưới cho tớ 10 phút học đi!'
  if (o.health >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return 'Tớ sắp héo mất! Học một chút nhé?'
}

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
