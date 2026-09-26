import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { ApiError } from '~/utils/apiClient'

/**
 * Backend spec §6.2 GET /roadmap. Outline only — no exercise content.
 */
export interface RoadmapTask {
  task_type: string
  title: string
  duration_minutes: number
}

export interface RoadmapDay {
  day_number: number
  date: string
  title: string
  tasks: RoadmapTask[]
  minutes_spent: number
  is_target_met: boolean
}

export interface RoadmapModule {
  week: number
  title: string
  focus: string
  days: RoadmapDay[]
}

export interface RoadmapOutline {
  roadmap_id: string
  title: string
  cefr_level: string
  created_at: string
  day_number: number
  modules: RoadmapModule[]
}

export const useRoadmapStore = defineStore('roadmap', {
  state: () => ({
    outline: null as RoadmapOutline | null,
    noRoadmap: false,
    loading: false,
    error: null as string | null,
  }),
  getters: {
    completedDays: (s): number =>
      s.outline ? s.outline.modules.reduce((n, m) => n + m.days.filter(d => d.is_target_met).length, 0) : 0,
  },
  actions: {
    async load() {
      this.loading = true
      this.error = null
      try {
        this.outline = await useApi().get<RoadmapOutline>('/api/v1/roadmap')
        this.noRoadmap = false
      } catch (e) {
        if (e instanceof ApiError && e.code === 'no_active_roadmap') {
          this.outline = null
          this.noRoadmap = true
        } else {
          this.error = e instanceof ApiError ? e.code : 'network_error'
        }
      } finally {
        this.loading = false
      }
    },
  },
})
