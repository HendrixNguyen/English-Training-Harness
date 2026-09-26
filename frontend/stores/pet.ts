import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'

/** Backend spec §6.3 GET /pet/status. */
export interface PetStatus {
  plant_name: string
  health_points: number
  stage: string
  current_streak: number
  last_practiced_at: string | null
  /** streak shield (backend spec §6.3, additive): 0–2 held; the local YYYY-MM-DD a shield was last spent for, or null. */
  shields: number
  last_shield_used_on: string | null
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

function storageOrNull(): Storage | null {
  return typeof localStorage === 'undefined' ? null : localStorage
}

export const usePetStore = defineStore('pet', {
  state: () => ({
    status: null as PetStatus | null,
    loading: false,
    error: null as string | null,
    challenge: null as Challenge | null,
    notWilted: false,
  }),
  getters: {
    isWilted: s => s.status !== null && (s.status.health_points <= 0 || s.status.stage === 'wilted'),
  },
  actions: {
    async load() {
      this.loading = true
      this.error = null
      try {
        this.status = await useApi().get<PetStatus>('/api/v1/pet/status')
      } catch (e) {
        this.error = e instanceof ApiError ? e.code : 'network_error'
      } finally {
        this.loading = false
      }
    },
    applyProgress(res: { pet_health: number, streak_count: number }) {
      if (!this.status) return
      this.status.health_points = res.pet_health
      this.status.current_streak = res.streak_count
    },
    hydrateChallenge() {
      const raw = storageOrNull()?.getItem(REVIVE_STORAGE_KEY)
      if (!raw) return
      try {
        const c = JSON.parse(raw) as Partial<Challenge>
        if (typeof c.date === 'string' && typeof c.startSeconds === 'number') this.challenge = { date: c.date, startSeconds: c.startSeconds }
      } catch {
        storageOrNull()?.removeItem(REVIVE_STORAGE_KEY)
      }
    },
    /**
     * §6.3 revive. `today` is GET /quests/daily `date`; `accumulatedSecondsNow`
     * its `accumulated_seconds` — the client-side anchor for the 15-minute bar
     * (the response carries no progress; pet plan Notes).
     */
    async revive(today: string, accumulatedSecondsNow: number): Promise<ReviveResponse | null> {
      this.error = null
      this.notWilted = false
      try {
        const res = await useApi().post<ReviveResponse>('/api/v1/pet/revive', { answers: {} })
        if (res.revival_passed) {
          if (this.status) Object.assign(this.status, res.pet_state)
          this.challenge = null
          storageOrNull()?.removeItem(REVIVE_STORAGE_KEY)
        } else if (!this.challenge || this.challenge.date !== today) {
          this.challenge = { date: today, startSeconds: accumulatedSecondsNow }
          storageOrNull()?.setItem(REVIVE_STORAGE_KEY, JSON.stringify(this.challenge))
        }
        return res
      } catch (e) {
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
