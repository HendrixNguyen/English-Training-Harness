import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'
import { useApi } from '~/composables/useApi'
import { registerPrivateMemory } from '~/utils/session'
import { ApiError } from '~/utils/apiClient'
import type { RoadmapOutline } from '~/utils/roadmap'

function scope() {
  const auth = useAuthStore()
  return `${auth.user?.id ?? ''}:${auth.sessionEpoch}`
}
/** Memory only: /roadmap is never persisted or service-worker cached. */
export const useRoadmapStore = defineStore('roadmap', {
  state: () => ({ data: null as RoadmapOutline | null, ownerScope: scope(), serial: 0, loading: false, error: null as string | null, noRoadmap: false }),
  getters: {
    outline: s => s.ownerScope === scope() ? s.data : null,
  },
  actions: {
    cancelPending() { this.serial++; this.loading = false },
    invalidate() {
      this.serial++; this.ownerScope = scope(); this.data = null; this.loading = false; this.error = null; this.noRoadmap = false
    },
    async load() {
      registerPrivateMemory(this)
      if (this.ownerScope !== scope()) this.invalidate()
      const owner = scope(); const serial = ++this.serial
      this.loading = true; this.error = null
      try {
        const result = await useApi().get<RoadmapOutline>('/api/v1/roadmap')
        if (owner !== scope() || serial !== this.serial) return
        this.data = result; this.noRoadmap = false
      } catch (error) {
        if (owner !== scope() || serial !== this.serial) return
        if (error instanceof ApiError && error.code === 'no_active_roadmap') { this.data = null; this.noRoadmap = true }
        else this.error = error instanceof ApiError ? error.code : 'network_error'
      } finally {
        if (owner === scope() && serial === this.serial) this.loading = false
      }
    },
  },
})
