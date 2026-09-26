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

/** `name` replaces "tớ" in the two lines that address the plant by name (design plant-name §2); blank keeps every line as before. */
export function speechLine(o: { stage: string, health: number, targetMet: boolean, name?: string }): string {
  const name = o.name?.trim() || 'tớ'
  const Name = name === 'tớ' ? 'Tớ' : name
  if (o.stage === 'wilted' || o.health <= 0) return '…'
  if (o.targetMet) return `Cảm ơn bạn, hôm nay ${name} đủ nước rồi 🌿`
  if (o.health >= 60) return 'Tưới cho tớ 10 phút học đi!'
  if (o.health >= 30) return 'Tớ hơi khát rồi… 10 phút thôi?'
  return `${Name} sắp héo mất! Học một chút nhé?`
}

/** Whole days between an ISO timestamp and now; null when absent or unparsable. */
export function daysSince(iso: string | null | undefined, now: Date = new Date()): number | null {
  if (!iso) return null
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return null
  return Math.max(0, Math.floor((now.getTime() - t) / 86_400_000))
}
