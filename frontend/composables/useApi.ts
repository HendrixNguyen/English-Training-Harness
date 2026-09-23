import { navigateTo, useRuntimeConfig } from '#app'
import { useAuthStore } from '~/stores/auth'
import { createApiClient, type ApiClient } from '~/utils/apiClient'

let client: ApiClient | null = null

/** The app's single API client: bearer from useAuthStore, sign-out + /login on 401. */
export function useApi(): ApiClient {
  if (client) return client
  const config = useRuntimeConfig()
  const auth = useAuthStore()
  client = createApiClient({
    baseURL: config.public.apiBase,
    getToken: () => auth.accessToken,
    onUnauthorized: () => {
      void auth.signOut()
      void navigateTo('/login')
    },
  })
  return client
}
