/** Backend spec §6.2: 3 × 10-minute tasks make the 30-minute day. */
export const SEGMENT_COUNT = 3
export const SEGMENT_SECONDS = 600
export const DAILY_TARGET_SECONDS = SEGMENT_COUNT * SEGMENT_SECONDS

/** quests.MaxDurationSeconds — a POST /quests/progress body outside 1..3600 is a 400. */
export const MAX_DURATION_SECONDS = 3600

/** Fill ratio (0..1) of each segment, left to right (design §1). */
export function segmentFills(valueSeconds: number, segmentSeconds = SEGMENT_SECONDS, segments = SEGMENT_COUNT): number[] {
  const fills: number[] = []
  let left = Math.max(0, valueSeconds)
  for (let i = 0; i < segments; i++) {
    fills.push(Math.min(1, left / segmentSeconds))
    left = Math.max(0, left - segmentSeconds)
  }
  return fills
}

export function minutesOf(seconds: number): number {
  return Math.floor(Math.max(0, seconds) / 60)
}

export function percentOf(seconds: number, target = DAILY_TARGET_SECONDS): number {
  return Math.min(100, Math.floor((Math.max(0, seconds) / target) * 100))
}

export function clampDuration(seconds: number): number {
  return Math.min(MAX_DURATION_SECONDS, Math.max(1, Math.floor(seconds)))
}

export function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  const mm = String(Math.floor(s / 60)).padStart(2, '0')
  const ss = String(s % 60).padStart(2, '0')
  return `${mm}:${ss}`
}
