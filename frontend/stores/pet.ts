import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'
import { useAuthStore } from '~/stores/auth'
import { registerPrivateMemory } from '~/utils/session'
import { stageForStreak } from '~/utils/plant'

/** Backend spec §6.3 GET /pet/status. */
export interface PetStatus {
  plant_name: string
  health_points: number
  stage: string
  current_streak: number
  last_practiced_at: string | null
}

/** Backend spec §6.3 POST /pet/revive 200 body. */
export interface ReviveResponse {
  revival_passed: boolean
  pet_state: { health_points: number, stage: string, current_streak: number }
}

/** The pet slice's pass condition: 15 minutes of study recorded after the challenge starts. */
export const REVIVE_SECONDS = 900
export const REVIVE_STORAGE_KEY = 'aelp.revive'

interface Challenge {
  date: string
  startSeconds: number
}

/** What the last progress response changed — the hub's growth moment reads it once (design growth-moment §3). */
export interface GrowthDelta {
  healthFrom: number
  healthTo: number
  streakFrom: number
  streakTo: number
  stageFrom: string
  stageTo: string
  targetMetNow: boolean
}

function scope() { const auth = useAuthStore(); return `${auth.user?.id ?? ''}:${auth.sessionEpoch}` }
export function reviveStorageKey() { return `${REVIVE_STORAGE_KEY}:${useAuthStore().user?.id ?? 'anonymous'}` }

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

export const usePetStore = defineStore('pet', {
  state: () => ({
    ownerScope: scope(),
    serial: 0,
    status: null as PetStatus | null,
    loading: false,
    error: null as string | null,
    challenge: null as Challenge | null,
    notWilted: false,
    lastDelta: null as GrowthDelta | null,
  }),
  getters: {
    isWilted: s => s.status !== null && (s.status.health_points <= 0 || s.status.stage === 'wilted'),
  },
  actions: {
    invalidate() {
      this.serial++; this.ownerScope = scope(); this.status = null; this.challenge = null; this.notWilted = false; this.lastDelta = null; this.loading = false; this.error = null
    },
    prepare() { registerPrivateMemory(this); if (this.ownerScope !== scope()) this.invalidate(); return { owner: scope(), serial: this.serial } },
    current(ticket: { owner: string; serial: number }) { return ticket.owner === scope() && ticket.serial === this.serial },
    async load() {
      const ticket = this.prepare()
      this.loading = true
      this.error = null
      try {
        const status = await useApi().get<PetStatus>('/api/v1/pet/status')
        if (this.current(ticket)) this.status = status
      } catch (e) {
        if (this.current(ticket)) this.error = e instanceof ApiError ? e.code : 'network_error'
      } finally {
        if (this.current(ticket)) this.loading = false
      }
    },
    /**
     * §6.2 progress response. Either pet field is absent when the backend's pet
     * read failed (CODEMAP quests) — then it is neither applied nor counted.
     * `targetMetChanged` comes from `useQuestStore.complete()`. Records `lastDelta`
     * for the hub's growth moment; take it once with `consumeDelta()`.
     */
    applyProgress(res: { pet_health?: number, streak_count?: number, targetMetChanged?: boolean }) {
      this.lastDelta = null
      if (!this.status) return
      const from = { health: this.status.health_points, streak: this.status.current_streak, stage: this.status.stage }
      if (typeof res.pet_health === 'number') this.status.health_points = res.pet_health
      if (typeof res.streak_count === 'number') this.status.current_streak = res.streak_count
      // Stage after: the same table the backend's SaveTargetMet writes; GET /pet/status on the hub confirms it.
      // Recompute only when health or streak actually moved — a stored `stage` that predates this
      // table (or a load raced by the backend) must not flip on a call where nothing changed.
      if (typeof res.pet_health === 'number' && typeof res.streak_count === 'number' && (res.pet_health !== from.health || res.streak_count !== from.streak)) {
        this.status.stage = stageForStreak(res.streak_count, res.pet_health)
      }
      const to = { health: this.status.health_points, streak: this.status.current_streak, stage: this.status.stage }
      const targetMetNow = res.targetMetChanged === true
      if (to.health === from.health && to.streak === from.streak && to.stage === from.stage && !targetMetNow) return
      this.lastDelta = { healthFrom: from.health, healthTo: to.health, streakFrom: from.streak, streakTo: to.streak, stageFrom: from.stage, stageTo: to.stage, targetMetNow }
    },
    /** The hub takes the delta once; a reload or a second visit sees the steady state. */
    consumeDelta(): GrowthDelta | null {
      const d = this.lastDelta
      this.lastDelta = null
      return d
    },
    hydrateChallenge() {
      this.prepare()
      const raw = storageOrNull()?.getItem(reviveStorageKey())
      if (!raw) return
      try {
        const c = JSON.parse(raw) as Partial<Challenge>
        if (typeof c.date === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(c.date) && typeof c.startSeconds === 'number' && Number.isFinite(c.startSeconds) && c.startSeconds >= 0) this.challenge = { date: c.date, startSeconds: c.startSeconds }
      } catch {
        storageOrNull()?.removeItem(reviveStorageKey())
      }
    },
    /**
     * §6.3 revive. `today` is GET /quests/daily `date`; `accumulatedSecondsNow`
     * its `accumulated_seconds` — the client-side anchor for the 15-minute bar
     * (the response carries no progress; pet plan Notes).
     */
    async revive(today: string, accumulatedSecondsNow: number): Promise<ReviveResponse | null> {
      const ticket = this.prepare()
      const storageKey = reviveStorageKey()
      this.error = null
      this.notWilted = false
      try {
        const res = await useApi().post<ReviveResponse>('/api/v1/pet/revive', { answers: {} })
        if (!this.current(ticket)) return null
        if (res.revival_passed) {
          if (this.status) Object.assign(this.status, res.pet_state)
          this.challenge = null
          storageOrNull()?.removeItem(storageKey)
        } else if (!this.challenge || this.challenge.date !== today) {
          this.challenge = { date: today, startSeconds: accumulatedSecondsNow }
          storageOrNull()?.setItem(storageKey, JSON.stringify(this.challenge))
        }
        return res
      } catch (e) {
        if (!this.current(ticket)) return null
        if (e instanceof ApiError && e.code === 'pet_not_wilted') {
          this.notWilted = true
          return null
        }
        this.error = e instanceof ApiError ? e.code : 'network_error'
        throw e
      }
    },
    challengeProgress(accumulatedSecondsNow: number): number {
      return this.challenge ? Math.max(0, accumulatedSecondsNow - this.challenge.startSeconds) : 0
    },
  },
})
